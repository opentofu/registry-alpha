package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	gogithub "github.com/google/go-github/v54/github"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/providers"
	"github.com/shurcooL/githubv4"
	"os"
)

type PopulateProviderVersionsEvent struct {
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
}

type Config struct {
	ManagedGithubClient *gogithub.Client
	RawGithubv4Client   *githubv4.Client
	ProviderRedirects   map[string]string
}

var config *Config

func init() {
	ctx := context.Background()

	c, err := buildConfig(ctx)
	if err != nil {
		panic(fmt.Errorf("could not build config: %w", err))
	}

	config = c
}

func buildConfig(ctx context.Context) (config *Config, err error) {
	if err = xray.Configure(xray.Config{ServiceVersion: "1.2.3"}); err != nil {
		err = fmt.Errorf("could not configure X-Ray: %w", err)
		return
	}

	// fetch the github token ASM name from the environment
	githubTokenSecretName := os.Getenv("GITHUB_TOKEN_SECRET_ASM_NAME")
	if githubTokenSecretName == "" {
		panic("GITHUB_TOKEN_SECRET_ASM_NAME environment variable not set")
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

	fmt.Printf("GitHub API token: %s\n", githubAPIToken)

	config = &Config{
		ManagedGithubClient: github.NewManagedGithubClient(githubAPIToken),
		RawGithubv4Client:   github.NewRawGithubv4Client(githubAPIToken),
	}

	return
}

func getSecretsManager(ctx context.Context) (*secretsmanager.Client, error) {
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(os.Getenv("AWS_REGION")))
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

func main() {
	lambda.Start(HandleRequest)
}

func HandleRequest(ctx context.Context, e PopulateProviderVersionsEvent) (string, error) {
	var versions []providers.Version

	fmt.Printf("Fetching %s/%s\n", e.Namespace, e.Type)
	err := xray.Capture(ctx, "populate_provider_versions.handle", func(tracedCtx context.Context) error {
		xray.AddAnnotation(tracedCtx, "namespace", e.Namespace)
		xray.AddAnnotation(tracedCtx, "type", e.Type)

		if e.Namespace == "" {
			return fmt.Errorf("namespace is required")
		}
		if e.Type == "" {
			return fmt.Errorf("type is required")
		}

		// Construct the repo name.
		repoName := providers.GetRepoName(e.Type)

		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, e.Namespace, repoName)
		if err != nil {
			return fmt.Errorf("failed to check if repo exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("repo does not exist")
		}

		fmt.Printf("Repo %s/%s exists\n", e.Namespace, repoName)

		v, err := providers.GetVersions(tracedCtx, config.RawGithubv4Client, e.Namespace, repoName)
		if err != nil {
			return fmt.Errorf("failed to get versions: %w", err)
		}

		fmt.Printf("Found %d versions\n", len(v))

		versions = v
		return nil
	})

	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		return "", err
	}

	// TODO: Send to dynamodb

	marshalled, err := json.Marshal(versions)
	if err != nil {
		return "", fmt.Errorf("failed to marshal versions: %w", err)
	}

	return string(marshalled), nil
}
