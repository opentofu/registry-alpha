package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/opentffoundation/registry/internal/config"
)

type LambdaFunc func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func main() {
	config, err := config.BuildConfig(context.Background(), "registry.buildconfig")
	if err != nil {
		panic(err)
	}

	lambda.Start(Router(*config))
}
