package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/opentffoundation/registry/internal/config"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/providers"
	"github.com/opentffoundation/registry/internal/providers/providercache"
	"os"
	"time"
)

const providerCacheAge = 1 * time.Hour

type ListProvidersPathParams struct {
	Namespace string `json:"namespace"`
	Type      string `json:"name"`
}

func getListProvidersPathParams(req events.APIGatewayProxyRequest) ListProvidersPathParams {
	return ListProvidersPathParams{
		Namespace: req.PathParameters["namespace"],
		Type:      req.PathParameters["type"],
	}
}

type ListProviderVersionsResponse struct {
	Versions []providers.Version `json:"versions"`
}

func listProviderVersions(config config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getListProvidersPathParams(req)
		effectiveNamespace := config.EffectiveProviderNamespace(params.Namespace)
		repoName := providers.GetRepoName(params.Type)

		if exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, effectiveNamespace, repoName); !exists {
			if err != nil {
				fmt.Printf("Error checking if repo exists: %s\n", err.Error())
				return events.APIGatewayProxyResponse{StatusCode: 500}, err
			}
			fmt.Printf("Repo %s/%s does not exist\n", effectiveNamespace, repoName)
			// if the repo doesn't exist, there's no point in trying to fetch versions
			return NotFoundResponse, nil
		}

		// For now, we will ignore errors from the cache and just fetch from GH instead
		document, _ := config.ProviderVersionCache.GetItem(ctx, fmt.Sprintf("%s/%s", effectiveNamespace, params.Type))
		if document != nil {
			return processDocument(ctx, document, config, effectiveNamespace, params.Type)
		}

		// if the document didn't exist in the cache, trigger the lambda to populate it and return the current results from GH
		if err := triggerPopulateProviderVersions(ctx, config, effectiveNamespace, params.Type); err != nil {
			fmt.Printf("Error triggering lambda to update dynamodb: %s\n", err.Error())
		}

		return fetchFromGithub(ctx, config, effectiveNamespace, repoName)
	}
}

func processDocument(ctx context.Context, document *providercache.VersionListingItem, config config.Config, namespace, providerType string) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("Found document with %d versions, last_updated: %s\n", len(document.Versions), document.LastUpdated.String())

	if document.LastUpdated.After(time.Now().Add(-providerCacheAge)) {
		fmt.Printf("Document is recent enough, returning it\n")
		return versionsResponse(document.Versions)
	}

	fmt.Printf("Document is too old, invoking lambda to update dynamodb\n")
	if err := triggerPopulateProviderVersions(ctx, config, namespace, providerType); err != nil {
		fmt.Printf("Error triggering lambda to update dynamodb: %s\n", err.Error())
	}

	return versionsResponse(document.Versions)
}

func fetchFromGithub(ctx context.Context, config config.Config, namespace, repoName string) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("Document not found in dynamodb, for %s%s invoking lambda and loading from github\n", namespace, repoName)

	versions, err := providers.GetVersions(ctx, config.RawGithubv4Client, namespace, repoName)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return versionsResponse(versions)
}

func triggerPopulateProviderVersions(ctx context.Context, config config.Config, effectiveNamespace string, effectiveType string) error {
	fmt.Printf("Invoking populate provider versions lambda asynchronously to update dynamodb document\n")
	// invoke the async lambda to update the dynamodb document
	_, err := config.LambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(os.Getenv("POPULATE_PROVIDER_VERSIONS_FUNCTION_NAME")),
		InvocationType: "Event", // Event == async
		Payload:        []byte(fmt.Sprintf("{\"namespace\": \"%s\", \"type\": \"%s\"}", effectiveNamespace, effectiveType)),
	})
	if err != nil {
		return err
	}
	return nil
}

func versionsResponse(versions []providers.Version) (events.APIGatewayProxyResponse, error) {
	response := ListProviderVersionsResponse{
		Versions: versions,
	}

	resBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
}
