package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-github/v54/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
)

type LambdaFunc func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func main() {
	// fetch the github token ASM name from the environment
	githubTokenSecretName := os.Getenv("GITHUB_TOKEN_SECRET_ASM_NAME")
	if githubTokenSecretName == "" {
		panic("GITHUB_TOKEN_SECRET_ASM_NAME environment variable not set")
	}

	secretsmanager := getSecretsManager()

	githubAPIToken, err := getSecretValue(secretsmanager, githubTokenSecretName)
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

func getSecretsManager() *secretsmanager.SecretsManager {
	awsSession, err := session.NewSession(&aws.Config{
		Region:     aws.String(os.Getenv("AWS_REGION")),
		MaxRetries: aws.Int(3),
		HTTPClient: &http.Client{},
	})
	if err != nil {
		log.Fatal(err)
	}

	return secretsmanager.New(awsSession)
}

func getSecretValue(sm *secretsmanager.SecretsManager, secretName string) (string, error) {
	value, err := sm.GetSecretValue(&secretsmanager.GetSecretValueInput{
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
