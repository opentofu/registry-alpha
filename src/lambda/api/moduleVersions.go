package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/opentofu/registry/internal/config"
	"golang.org/x/exp/slog"

	"github.com/aws/aws-lambda-go/events"

	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/modules"
)

type ListModuleVersionsPathParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	System    string `json:"system"`
}

func (p ListModuleVersionsPathParams) AnnotateLogger() {
	logger := slog.Default()
	logger = logger.
		With("namespace", p.Namespace).
		With("name", p.Name).
		With("system", p.System)
	slog.SetDefault(logger)
}

func getListModuleVersionsPathParams(req events.APIGatewayProxyRequest) ListModuleVersionsPathParams {
	return ListModuleVersionsPathParams{
		Namespace: req.PathParameters["namespace"],
		Name:      req.PathParameters["name"],
		System:    req.PathParameters["system"],
	}
}

type ListModuleVersionsResponse struct {
	Modules []ModulesResponse `json:"modules"`
}

type ModulesResponse struct {
	Versions []modules.Version `json:"versions"`
}

func listModuleVersions(config config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getListModuleVersionsPathParams(req)
		params.AnnotateLogger()
		repoName := modules.GetRepoName(params.System, params.Name)

		// try and fetch the document from the cache
		key := fmt.Sprintf("%s/%s", params.Namespace, repoName)
		document, _ := config.ModuleVersionCache.GetItem(ctx, key)
		if document != nil {
			return processDocumentForModuleVersionListing(ctx, document, config, params.Namespace, params.Name, params.System)
		}

		slog.Info("Document not found in cache, fetching from github")
		return fetchModuleVersionsFromGitHub(ctx, config, params, repoName)
	}
}

func processDocumentForModuleVersionListing(ctx context.Context, document *modules.CacheItem, config config.Config, namespace, name, system string) (events.APIGatewayProxyResponse, error) {
	slog.Info("Found document in cache", "document", document)

	// if it's not stale. return it!
	if document.IsStale() {
		slog.Info("Document is stale, triggering lambda to populate")
		err := triggerPopulateModuleVersions(ctx, config, namespace, name, system)
		if err != nil {
			// if we can't trigger the lambda, that's fine. we'll just log it and carry on
			slog.Error("Error triggering lambda", "error", err)
		}
	}

	return moduleVersionsResponse(document)
}

func moduleVersionsResponse(document *modules.CacheItem) (events.APIGatewayProxyResponse, error) {
	response := ListModuleVersionsResponse{
		Modules: []ModulesResponse{
			{
				Versions: document.Versions.ToVersionListResponse(),
			},
		},
	}
	resBody, err := json.Marshal(response)
	if err != nil {
		slog.Error("Error marshalling response", "error", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}
	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: string(resBody)}, nil
}

func fetchModuleVersionsFromGitHub(ctx context.Context, config config.Config, params ListModuleVersionsPathParams, repoName string) (events.APIGatewayProxyResponse, error) {
	// check the repo exists
	exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, params.Namespace, repoName)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}
	if !exists {
		return NotFoundResponse, nil
	}

	// fetch all the versions
	versions, err := modules.GetVersions(ctx, config.RawGithubv4Client, params.Namespace, repoName)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	responseVersions := versions.ToVersionListResponse()

	response := ListModuleVersionsResponse{
		Modules: []ModulesResponse{
			{
				Versions: responseVersions,
			},
		},
	}

	resBody, err := json.Marshal(response)
	if err != nil {
		slog.Error("Error marshalling response", "error", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}
	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: string(resBody)}, nil
}

func triggerPopulateModuleVersions(ctx context.Context, config config.Config, namespace, name, system string) error {
	slog.Info("Invoking populate module versions lambda asynchronously to update dynamodb document\n")
	// invoke the async lambda to update the dynamodb document
	_, err := config.LambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(os.Getenv("POPULATE_MODULE_VERSIONS_FUNCTION_NAME")),
		InvocationType: "Event", // Event == async
		Payload:        []byte(fmt.Sprintf("{\"namespace\": \"%s\", \"name\": \"%s\", \"sytstem\": \"%s\"}", namespace, name, system)),
	})
	if err != nil {
		slog.Error("Error invoking lambda", "error", err)
		return err
	}
	return nil
}
