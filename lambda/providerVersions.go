package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/providers"
)

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

func listProviderVersions(config Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getListProvidersPathParams(req)

		// Construct the repo name.
		repoName := providers.GetRepoName(params.Type)

		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, params.Namespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		if !exists {
			return NotFoundResponse, nil
		}

		versions, err := providers.GetVersions(ctx, config.RawGithubv4Client, params.Namespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		response := ListProviderVersionsResponse{
			Versions: versions,
		}

		resBody, err := json.Marshal(response)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
	}
}
