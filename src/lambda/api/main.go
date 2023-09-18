package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opentffoundation/registry/internal/github"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
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

func buildConfig(ctx context.Context, githubTokenSecretName string) (config *Config, err error) {
	if err = xray.Configure(xray.Config{ServiceVersion: "1.2.3"}); err != nil {
		err = fmt.Errorf("could not configure X-Ray: %w", err)
		return
	}

	// At this point we're not part of a Lambda request execution, so let's
	// explicitly create a segment to represent the configuration process.
	ctx, segment := xray.BeginSegment(ctx, "registry.config")
	defer func() { segment.Close(err) }()

	var secretsmanager *secretsmanager.Client
	if secretsmanager, err = getSecretsManager(ctx); err != nil {
		err = fmt.Errorf("could not get secrets manager client: %w", err)
		return
	}

	var githubAPIToken string
	if githubAPIToken, err = getSecretValue(ctx, secretsmanager, githubTokenSecretName); err != nil {
		err = fmt.Errorf("could not get GitHub API token: %w", err)
		return
	}

	if githubAPIToken == "" {
		err = fmt.Errorf("empty GitHub API token fetched from secrets manager")
		return
	}

	config = &Config{
		ManagedGithubClient: github.NewManagedGithubClient(githubAPIToken),
		RawGithubv4Client:   github.NewRawGithubv4Client(githubAPIToken),
	}

	return
}

func getSecretsManager(ctx context.Context) (*secretsmanager.Client, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return nil, fmt.Errorf("could not load AWS configuration: %w", err)
	}

	awsv2.AWSV2Instrumentor(&awsConfig.APIOptions)

	return secretsmanager.NewFromConfig(awsConfig), nil
}

func getSecretValue(ctx context.Context, sm *secretsmanager.Client, secretName string) (string, error) {
	value, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return "", err
	}
	return *value.SecretString, nil
}
