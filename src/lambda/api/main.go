package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type LambdaFunc func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func main() {
	// fetch the github token ASM name from the environment
	githubTokenSecretName := os.Getenv("GITHUB_TOKEN_SECRET_ASM_NAME")
	if githubTokenSecretName == "" {
		panic("GITHUB_TOKEN_SECRET_ASM_NAME environment variable not set")
	}

	providerRedirects := make(map[string]string)
	if redirectsJSON, ok := os.LookupEnv("PROVIDER_NAMESPACE_REDIRECTS"); ok {
		if err := json.Unmarshal([]byte(redirectsJSON), &providerRedirects); err != nil {
			panic(fmt.Errorf("could not parse PROVIDER_NAMESPACE_REDIRECTS: %w", err))
		}
	}

	ctx := context.Background()

	config, err := buildConfig(ctx, githubTokenSecretName)
	if err != nil {
		panic(err)
	}

	config.ProviderRedirects = providerRedirects

	lambda.Start(Router(*config))
}
