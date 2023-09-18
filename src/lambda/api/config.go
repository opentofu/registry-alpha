package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/xray"
	gogithub "github.com/google/go-github/v54/github"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/secrets"
	"github.com/shurcooL/githubv4"
)

type Config struct {
	ManagedGithubClient *gogithub.Client
	RawGithubv4Client   *githubv4.Client
	ProviderRedirects   map[string]string
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
	if secretsmanager, err = secrets.GetClient(ctx); err != nil {
		err = fmt.Errorf("could not get secrets manager client: %w", err)
		return
	}

	var githubAPIToken string
	if githubAPIToken, err = secrets.GetValue(ctx, secretsmanager, githubTokenSecretName); err != nil {
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
