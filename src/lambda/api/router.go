package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentofu/registry/internal/config"
	"golang.org/x/exp/slog"

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

		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		logger = logger.
			With("request_id", req.RequestContext.RequestID).
			With("path", req.Path)
		slog.SetDefault(logger)

		handler := getRouteHandler(config, req.Path)
		if handler == nil {
			slog.Error("No route handler found for path")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusNotFound, Body: fmt.Sprintf("No route handler found for path %s", req.Path)}, nil
		}

		response, err := handler(ctx, req)
		segment.Close(err)

		slog.Info("Returning response", "status_code", response.StatusCode)
		return response, err
	}
}
