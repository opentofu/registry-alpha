package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type LambdaFunc func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func main() {
	ctx := context.Background()

	config, err := buildConfig(ctx)
	if err != nil {
		panic(err)
	}

	lambda.Start(Router(*config))
}
