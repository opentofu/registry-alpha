package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/go-github/v54/github"
	"github.com/shurcooL/githubv4"
	"regexp"
)

type Config struct {
	ManagedGithubClient *github.Client
	RawGithubv4Client   *githubv4.Client
}

func RouteHandlers(config Config) map[string]LambdaFunc {
	return map[string]LambdaFunc{
		// Download provider version
		// `/v1/providers/{namespace}/{type}/{version}/download/{os}/{arch}`
		"^/v1/providers/[^/]+/[^/]+/[^/]+/download/[^/]+/[^/]+$": downloadProviderVersion(config),

		// List provider versions
		// `/v1/providers/{namespace}/{type}/versions`
		"^/v1/providers/[^/]+/[^/]+/versions$": listProviderVersions(config),

		// List module versions
		// `/v1/modules/{namespace}/{name}/{system}/versions`
		"^/v1/modules/[^/]+/[^/]+/[^/]+/versions$": listModuleVersions(config),

		// .well-known/terraform.json
		"^/.well-known/terraform.json$": terraformWellknownMetadataHandler(config),
	}
}

func getRouteHandler(config Config, path string) LambdaFunc {
	// We will replace this with some sort of actual router (chi, gorilla, etc)
	// for now regex is fine
	for route, handler := range RouteHandlers(config) {
		if match, _ := regexp.MatchString(route, path); match {
			return handler
		}
	}
	return nil
}

func Router(config Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		handler := getRouteHandler(config, req.Path)
		if handler == nil {
			return events.APIGatewayProxyResponse{StatusCode: 404, Body: fmt.Sprintf("No route handler found for path %s", req.Path)}, nil
		}

		return handler(ctx, req)
	}
}
