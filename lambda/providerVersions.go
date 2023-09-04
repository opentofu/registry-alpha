package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
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

		versions, err := providers.GetVersions(ctx, config.GithubClient, params.Namespace, params.Type)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		response := ListProviderVersionsResponse{
			Versions: versions,
		}

		resBody, _ := json.Marshal(response)
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
	}
}
