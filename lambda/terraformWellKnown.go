package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
)

func terraformWellknownMetadataHandler(config Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		// TODO: update the list of urls so that they are full urls and not just relative paths once we have a registry domain set in stone
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       `{"providers.v1": "/v1/providers"}`,
		}, nil
	}
}
