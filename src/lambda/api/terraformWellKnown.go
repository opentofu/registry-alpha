package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/opentffoundation/registry/internal/config"
)

const wellKnownMetadataResponse = `{
	  "modules.v1": "/v1/modules/",
	  "providers.v1": "/v1/providers/"
}`

func terraformWellKnownMetadataHandler(_ config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       wellKnownMetadataResponse,
		}, nil
	}
}
