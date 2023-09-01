package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"regexp"
)

func RouteHandlers() map[string]LambdaFunc {
	return map[string]LambdaFunc{
		// `/v1/providers/{namespace}/{type}/{version}/download/{os}/{arch}`
		".*/v1/providers/.*/.*/.*/download/.*/.*": downloadProviderVersion,
	}
}

func getRouteHandler(path string) LambdaFunc {
	// check if any of the routes match the regex of the path

	// We will replace this with some sort of actual router (chi, gorilla, etc)
	// for now regex is fine
	for route, handler := range RouteHandlers() {
		if match, _ := regexp.MatchString(route, path); match {
			return handler
		}
	}
	return nil
}

func Router() LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		fmt.Printf("Request: %+v\n", req)
		fmt.Printf("Path: %s\n", req.Path)
		handler := getRouteHandler(req.Path)
		if handler == nil {
			return events.APIGatewayProxyResponse{StatusCode: 404}, nil
		}

		return handler(ctx, req)
	}
}
