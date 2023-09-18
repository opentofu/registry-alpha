package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"

	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/modules"
)

type DownloadModuleHandlerPathParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	System    string `json:"system"`
	Version   string `json:"version"`
}

func downloadModuleVersion(config Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getDownloadModuleHandlerPathParams(req)

		repoName := modules.GetRepoName(params.System, params.Name)

		// check if the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, params.Namespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		if !exists {
			return NotFoundResponse, nil
		}

		return events.APIGatewayProxyResponse{StatusCode: 200, Body: "", Headers: map[string]string{
			"X-Terraform-Get": fmt.Sprintf("git::https://github.com/%s/%s?ref=v%s", params.Namespace, repoName, params.Version),
		}}, nil
	}
}

func getDownloadModuleHandlerPathParams(req events.APIGatewayProxyRequest) DownloadModuleHandlerPathParams {
	return DownloadModuleHandlerPathParams{
		Namespace: req.PathParameters["namespace"],
		Name:      req.PathParameters["name"],
		System:    req.PathParameters["system"],
		Version:   req.PathParameters["version"],
	}
}
