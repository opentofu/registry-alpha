package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentffoundation/registry/internal/config"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
)

func RouteHandlers(config config.Config) map[string]LambdaFunc {
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

		// Download module version
		// `/v1/modules/{namespace}/{name}/{system}/{version}/download`
		"^/v1/modules/[^/]+/[^/]+/[^/]+/[^/]+/download$": downloadModuleVersion(config),

		// .well-known/terraform.json
		"^/.well-known/terraform.json$": terraformWellKnownMetadataHandler(config),
	}
}

func getRouteHandler(config config.Config, path string) LambdaFunc {
	// We will replace this with some sort of actual router (chi, gorilla, etc)
	// for now regex is fine
	for route, handler := range RouteHandlers(config) {
		if match, _ := regexp.MatchString(route, path); match {
			return handler
		}
	}
	return nil
}

func Router(config config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		ctx, segment := xray.BeginSubsegment(ctx, "registry.handle")
		handler := getRouteHandler(config, req.Path)
		if handler == nil {
			return events.APIGatewayProxyResponse{StatusCode: 404, Body: fmt.Sprintf("No route handler found for path %s", req.Path)}, nil
		}

		response, err := handler(ctx, req)
		segment.Close(err)

		return response, err
	}
}
