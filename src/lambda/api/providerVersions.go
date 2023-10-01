package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/opentofu/registry/internal/config"
	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/providers"
	"github.com/opentofu/registry/internal/warnings"
	"github.com/opentofu/registry/internal/providers/types"
	"golang.org/x/exp/slog"
)

type ListProvidersPathParams struct {
	Namespace string `json:"namespace"`
	Type      string `json:"name"`
}

func (p ListProvidersPathParams) AnnotateLogger() {
	logger := slog.Default()
	logger = logger.
		With("namespace", p.Namespace).
		With("type", p.Type)
	slog.SetDefault(logger)
}

func getListProvidersPathParams(req events.APIGatewayProxyRequest) ListProvidersPathParams {
	return ListProvidersPathParams{
		Namespace: req.PathParameters["namespace"],
		Type:      req.PathParameters["type"],
	}
}

type ListProviderVersionsResponse struct {
	Versions []types.Version `json:"versions"`
	Warnings []string            `json:"warnings,omitempty"`
}

func listProviderVersions(config config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getListProvidersPathParams(req)
		params.AnnotateLogger()

		effectiveNamespace := config.EffectiveProviderNamespace(params.Namespace)
		repoName := providers.GetRepoName(params.Type)

		// Warnings lookup: https://github.com/opentofu/registry/issues/108
		ctx = warnings.NewContext(ctx, warnings.ProviderWarnings(params.Namespace, params.Type))

		// For now, we will ignore errors from the cache and just fetch from GH instead
		document, _ := config.ProviderVersionCache.GetItem(ctx, fmt.Sprintf("%s/%s", effectiveNamespace, params.Type))
		if document != nil {
			return processDocumentForProviderListing(ctx, document, config, effectiveNamespace, params.Type)
		}

		// now that we know we don't have the document, we should check that the repo exists
		// if we checked the repo exists before then we are making extra calls to GitHub that we don't need to make.
		if exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, effectiveNamespace, repoName); !exists {
			if err != nil {
				slog.Error("Error checking if repo exists", "error", err)
				return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
			}
			slog.Info("Repo does not exist")
			// if the repo doesn't exist, there's no point in trying to fetch versions
			return NotFoundResponse, nil
		}

		// if the document didn't exist in the cache, trigger the lambda to populate it and return the current results from GH
		if err := triggerPopulateProviderVersions(ctx, config, effectiveNamespace, params.Type); err != nil {
			slog.Error("Error triggering lambda", "error", err)
		}

		return fetchFromGithub(ctx, config, effectiveNamespace, repoName)
	}
}

func processDocumentForProviderListing(ctx context.Context, document *types.CacheItem, config config.Config, namespace, providerType string) (events.APIGatewayProxyResponse, error) {
	slog.Info("Found document in cache", "last_updated", document.LastUpdated, "versions", len(document.Versions))

	if !document.IsStale() {
		slog.Info("Document is not too old, returning cached versions", "last_updated", document.LastUpdated)
		return versionsResponse(document.Versions.ToVersions(), warnings.FromContext(ctx))
	}

	slog.Info("Document is too old, triggering lambda to update dynamodb", "last_updated", document.LastUpdated)
	if err := triggerPopulateProviderVersions(ctx, config, namespace, providerType); err != nil {
		// if we can't trigger the lambda, we should still return the cached versions and just log the error
		slog.Error("Error triggering lambda", "error", err)
	}

	return versionsResponse(document.Versions.ToVersions(), warnings.FromContext(ctx))
}

func fetchFromGithub(ctx context.Context, config config.Config, namespace, repoName string) (events.APIGatewayProxyResponse, error) {
	slog.Info("Fetching versions from github\n")

	versionList, err := providers.GetVersions(ctx, config.RawGithubv4Client, namespace, repoName)
	if err != nil {
		slog.Error("Error fetching versions from github", "error", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return versionsResponse(versions.ToVersions(), warnings.FromContext(ctx))
}

func triggerPopulateProviderVersions(ctx context.Context, config config.Config, effectiveNamespace string, effectiveType string) error {
	slog.Info("Invoking populate provider versions lambda asynchronously to update dynamodb document\n")
	// invoke the async lambda to update the dynamodb document
	_, err := config.LambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(os.Getenv("POPULATE_PROVIDER_VERSIONS_FUNCTION_NAME")),
		InvocationType: "Event", // Event == async
		Payload:        []byte(fmt.Sprintf("{\"namespace\": \"%s\", \"type\": \"%s\"}", effectiveNamespace, effectiveType)),
	})
	if err != nil {
		slog.Error("Error invoking lambda", "error", err)
		return err
	}
	return nil
}

func versionsResponse(versions []providers.Version, warnings []string) (events.APIGatewayProxyResponse, error) {
	response := ListProviderVersionsResponse{
		Versions: versions,
	}

	if len(warnings) > 0 {
		response.Warnings = warnings
	}

	resBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: string(resBody)}, nil
}
