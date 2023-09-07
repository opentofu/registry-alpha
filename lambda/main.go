package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/google/go-github/v54/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type LambdaFunc func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func main() {
	// fetch the github token ASM name from the environment
	githubTokenSecretName := os.Getenv("GITHUB_TOKEN_SECRET_ASM_NAME")
	if githubTokenSecretName == "" {
		panic("GITHUB_TOKEN_SECRET_ASM_NAME environment variable not set")
	}

	ctx := context.Background()

	secretsmanager, err := getSecretsManager(ctx)
	if err != nil {
		panic(err)
	}

	githubAPIToken, err := getSecretValue(ctx, secretsmanager, githubTokenSecretName)
	if err != nil {
		panic(err)
	}

	if githubAPIToken == "" {
		panic("empty github api token fetched from secrets manager")
	}

	managedGithubClient := getManagedGithubClient(githubAPIToken)
	rawGithubClient := getRawGithubv4Client(githubAPIToken)

	lambda.Start(Router(Config{
		ManagedGithubClient: managedGithubClient,
		RawGithubv4Client:   rawGithubClient,
	}))
}

func getSecretsManager(ctx context.Context) (*secretsmanager.Client, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return nil, fmt.Errorf("could not load AWS configuration: %w", err)
	}

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

func getGithubOauth2Client(token string) *http.Client {
	return oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))
}

func getManagedGithubClient(token string) *github.Client {
	return github.NewClient(getGithubOauth2Client(token))
}

func getRawGithubv4Client(token string) *githubv4.Client {
	return githubv4.NewClient(getGithubOauth2Client(token))
}
