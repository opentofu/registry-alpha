package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentofu/registry/internal/config"
	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/providers"
	"github.com/opentofu/registry/internal/providers/providercache"
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

			// check if the document exists in dynamodb, if it does, and it's newer than the allowed max age,
			// we should treat it as a noop and just return
			document, err := config.ProviderVersionCache.GetItem(tracedCtx, fmt.Sprintf("%s/%s", e.Namespace, e.Type))
			if err != nil {
				// if there was an error getting the document, that's fine. we'll just log it and carry on
				fmt.Printf("Error: failed to get item from cache: %s", err.Error())
			}
			if document != nil {
				if time.Since(document.LastUpdated) < providercache.AllowedAge {
					fmt.Printf("Document is up to date, not updating\n")
					return nil
				}
			}

			fetchedVersions, err := fetchFromGithub(tracedCtx, e, config)
			if err != nil {
				return err
			}

			versions = fetchedVersions
			return nil
		})

		if err != nil {
			fmt.Printf("error fetching provider versions: %s\n", err.Error())
			return "", err
		}

		err = storeVersions(ctx, e, versions, config)
		if err != nil {
			return "", err
		}

		return "", nil
	}
}

func storeVersions(ctx context.Context, e PopulateProviderVersionsEvent, versions []providers.Version, config *config.Config) error {
	if len(versions) == 0 {
		return fmt.Errorf("no versions found")
	}

	key := fmt.Sprintf("%s/%s", e.Namespace, e.Type)

	err := config.ProviderVersionCache.Store(ctx, key, versions)
	if err != nil {
		return fmt.Errorf("failed to store provider listing: %w", err)
	}
	return nil
}

func fetchFromGithub(ctx context.Context, e PopulateProviderVersionsEvent, config *config.Config) ([]providers.Version, error) {
	// Construct the repo name.
	repoName := providers.GetRepoName(e.Type)

	// check the repo exists
	exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, e.Namespace, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("repo %s/%s does not exist", e.Namespace, repoName)
	}

	fmt.Printf("Repo %s/%s exists\n", e.Namespace, repoName)

	v, err := providers.GetVersions(ctx, config.RawGithubv4Client, e.Namespace, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	fmt.Printf("Found %d versions\n", len(v))
	return v, nil
}
