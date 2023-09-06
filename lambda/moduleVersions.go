package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/opentffoundation/registry/internal/modules"
)

import (
	"context"
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

func listModuleVersions(config Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getListModuleVersionsPathParams(req)

		versions, err := modules.GetVersions(ctx, config.RawGithubv4Client, params.Namespace, params.Name, params.System)
		if err != nil {
			// TODO: handle missing repo
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
