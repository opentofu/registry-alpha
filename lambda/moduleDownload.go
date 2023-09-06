package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
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

		url := modules.GetVersionDownloadUrl(ctx, params.Namespace, params.Name, params.System, params.Version)

		// TODO : check that the repo does exist

		return events.APIGatewayProxyResponse{StatusCode: 200, Body: "", Headers: map[string]string{
			"X-Terraform-Get": url,
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
