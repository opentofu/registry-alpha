package main

import (
	"context"
	"fmt"
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

		url, err := modules.GetVersionDownloadUrl(ctx, config.RawGithubv4Client, params.Namespace, params.Name, params.System, params.Version)
		if err != nil {
			// log the error too for dev
			fmt.Printf("error fetching version: %s\n", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		return events.APIGatewayProxyResponse{StatusCode: 200, Body: "", Headers: map[string]string{
			"X-Terraform-Get": *url,
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
