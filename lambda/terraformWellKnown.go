package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
)

func terraformWellknownMetadataHandler(config Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		host := req.Headers["Host"]
		if host == "" {
			host = req.RequestContext.DomainName
		}

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       fmt.Sprintf(`{"providers.v1": "https://%s/v1/providers/"}`, host),
		}, nil
	}
}
