package main

import (
	"context"
	"encoding/json"
	"github.com/opentffoundation/registry/internal/config"

	"github.com/aws/aws-lambda-go/events"

	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/modules"
)

type ListModuleVersionsPathParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	System    string `json:"system"`
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
		repoName := modules.GetRepoName(params.System, params.Name)

		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, params.Namespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		if !exists {
			return NotFoundResponse, nil
		}

		// fetch all the versions
		versions, err := modules.GetVersions(ctx, config.RawGithubv4Client, params.Namespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		response := ListModuleVersionsResponse{
			Modules: []ModulesResponse{
				{
					Versions: versions,
				},
			},
		}

		resBody, err := json.Marshal(response)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
	}
}
