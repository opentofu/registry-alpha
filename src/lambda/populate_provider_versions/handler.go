package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentffoundation/registry/internal/config"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/providers"
)

type PopulateProviderVersionsEvent struct {
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
}

func (p PopulateProviderVersionsEvent) Validate() error {
	if p.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if p.Type == "" {
		return fmt.Errorf("type is required")
	}
	return nil
}

type LambdaFunc func(ctx context.Context, e PopulateProviderVersionsEvent) (string, error)

func HandleRequest(config *config.Config) LambdaFunc {
	return func(ctx context.Context, e PopulateProviderVersionsEvent) (string, error) {
		var versions []providers.Version

		fmt.Printf("Fetching versions for  %s/%s\n", e.Namespace, e.Type)
		err := xray.Capture(ctx, "populate_provider_versions.handle", func(tracedCtx context.Context) error {
			xray.AddAnnotation(tracedCtx, "namespace", e.Namespace)
			xray.AddAnnotation(tracedCtx, "type", e.Type)

			err := e.Validate()
			if err != nil {
				return fmt.Errorf("invalid event: %w", err)
			}

			// Construct the repo name.
			repoName := providers.GetRepoName(e.Type)

			// check the repo exists
			exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, e.Namespace, repoName)
			if err != nil {
				return fmt.Errorf("failed to check if repo exists: %w", err)
			}
			if !exists {
				return fmt.Errorf("repo %s/%s does not exist", e.Namespace, repoName)
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
			fmt.Printf("error fetching provider versions: %s\n", err.Error())
			return "", err
		}

		key := fmt.Sprintf("%s/%s", e.Namespace, e.Type)

		err = config.ProviderVersionCache.Store(ctx, key, versions)
		if err != nil {
			return "", fmt.Errorf("failed to store provider listing: %w", err)
		}

		return "", nil
	}
}
