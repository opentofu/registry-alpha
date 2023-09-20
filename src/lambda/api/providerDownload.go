package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"

	"github.com/opentofu/registry/internal/config"
	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/providers"
)

type DownloadHandlerPathParams struct {
	Architecture string `json:"arch"`
	OS           string `json:"os"`
	Namespace    string `json:"namespace"`
	Type         string `json:"type"`
	Version      string `json:"version"`
}

func getDownloadPathParams(req events.APIGatewayProxyRequest) DownloadHandlerPathParams {
	return DownloadHandlerPathParams{
		Architecture: req.PathParameters["arch"],
		OS:           req.PathParameters["os"],
		Namespace:    req.PathParameters["namespace"],
		Type:         req.PathParameters["type"],
		Version:      req.PathParameters["version"],
	}
}

func downloadProviderVersion(config config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getDownloadPathParams(req)
		effectiveNamespace := config.EffectiveProviderNamespace(params.Namespace)

		// Construct the repo name.
		repoName := providers.GetRepoName(params.Type)

		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, effectiveNamespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		if !exists {
			return NotFoundResponse, nil
		}

		versionDownloadResponse, err := providers.GetVersion(ctx, config.RawGithubv4Client, effectiveNamespace, repoName, params.Version, params.OS, params.Architecture)
		if err != nil {
			// log the error too for dev
			fmt.Printf("error fetching version: %s\n", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		resBody, err := json.Marshal(versionDownloadResponse)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
	}
}
