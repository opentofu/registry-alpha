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

		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, effectiveNamespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		if !exists {
			return NotFoundResponse, nil
		}

		document, err := config.ProviderVersionCache.GetItem(ctx, fmt.Sprintf("%s/%s", effectiveNamespace, params.Type))
		if err != nil {
			// log the error but carry on. If there is an error fetching the document, we'll just fetch it from github.
			// we want to be fault-tolerant here so that we don't fail to serve the request if there is an error
			// fetching the document.
			fmt.Printf("Error fetching document from dynamodb: %s\n", err.Error())
		}
		if document != nil {
			fmt.Printf("Found document with %d versions, last_updated: %s\n", len(document.Versions), document.LastUpdated.String())
			// if the document is within our accepted age range, return it
			if document.LastUpdated.After(time.Now().Add(-providerCacheAge)) {
				fmt.Printf("Document is recent enough, returning it\n")
			}

			// else the document is too old, we should update the cache
			fmt.Printf("Document is too old, invoking lambda to update dynamodb\n")
			err = triggerPopulateProviderVersions(ctx, config, effectiveNamespace, params.Type)
			if err != nil {
				// log the error but carry on. If there is an error triggering the lambda, we'll just fetch it from github.
				fmt.Printf("Error triggering lambda to update dynamodb: %s\n", err.Error())
			}

			// no matter what, if we had a document we should return it
			return foundDocumentResponse(document)
		}
		fmt.Printf("Document not found in dynamodb, invoking lambda and loading from github\n")

		err = triggerPopulateProviderVersions(ctx, config, effectiveNamespace, params.Type)
		if err != nil {
			// log the error but carry on. If there is an error triggering the lambda, we'll just fetch it from github.
			fmt.Printf("Error triggering lambda to update dynamodb: %s\n", err.Error())
		}

		// fetch from GH
		versions, err := providers.GetVersions(ctx, config.RawGithubv4Client, effectiveNamespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		return providerVersionsResponse(versions, err)
	}
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

func providerVersionsResponse(versions []providers.Version, err error) (events.APIGatewayProxyResponse, error) {
	response := ListProviderVersionsResponse{
		Versions: versions,
	}

	resBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
}

func foundDocumentResponse(document *providercache.VersionListingItem) (events.APIGatewayProxyResponse, error) {
	// we found the document, return it
	response := ListProviderVersionsResponse{
		Versions: document.Versions,
	}

	resBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
}
