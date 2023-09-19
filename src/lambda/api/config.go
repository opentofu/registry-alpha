package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-xray-sdk-go/xray"
	gogithub "github.com/google/go-github/v54/github"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/secrets"
	"github.com/shurcooL/githubv4"
	"os"
)

type Config struct {
	ManagedGithubClient *gogithub.Client
	RawGithubv4Client   *githubv4.Client
	ProviderRedirects   map[string]string

	SecretsHandler *secrets.Handler
}

func buildConfig(ctx context.Context) (config *Config, err error) {
	if err = xray.Configure(xray.Config{ServiceVersion: "1.2.3"}); err != nil {
		err = fmt.Errorf("could not configure X-Ray: %w", err)
		return
	}

	// At this point we're not part of a Lambda request execution, so let's
	// explicitly create a segment to represent the configuration process.
	ctx, segment := xray.BeginSegment(ctx, "registry.config")
	defer func() { segment.Close(err) }()

	var awsConfig aws.Config
	awsConfig, err = awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		err = fmt.Errorf("could not load AWS configuration: %w", err)
		return
	}

	secretsmanager := secrets.NewHandler(awsConfig)

	githubAPIToken, err := secretsmanager.GetValueFromEnvVar(ctx, "GITHUB_TOKEN_SECRET_ASM_NAME")
	if err != nil {
		err = fmt.Errorf("could not get GitHub API token: %w", err)
		return
	}

	providerRedirects := make(map[string]string)
	if redirectsJSON, ok := os.LookupEnv("PROVIDER_NAMESPACE_REDIRECTS"); ok {
		if err := json.Unmarshal([]byte(redirectsJSON), &providerRedirects); err != nil {
			panic(fmt.Errorf("could not parse PROVIDER_NAMESPACE_REDIRECTS: %w", err))
		}
	}

	config = &Config{
		ManagedGithubClient: github.NewManagedGithubClient(githubAPIToken),
		RawGithubv4Client:   github.NewRawGithubv4Client(githubAPIToken),
		ProviderRedirects:   providerRedirects,

		SecretsHandler: secretsmanager,
	}

	return
}
