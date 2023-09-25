package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/opentffoundation/registry/internal/config"
)

func main() {
	configBuilder := config.NewConfigBuilder()
	config, err := configBuilder.BuildConfig(context.Background(), "populate_provider_versions.buildconfig")
	if err != nil {
		panic(fmt.Errorf("could not build config: %w", err))
	}

	lambda.Start(HandleRequest(config))
}
