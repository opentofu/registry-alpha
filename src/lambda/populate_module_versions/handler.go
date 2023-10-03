package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentofu/registry/internal/config"
	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/modules"
	"golang.org/x/exp/slog"
)

type PopulateModuleVersionsEvent struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	System    string `json:"system"`
}

func (p PopulateModuleVersionsEvent) Validate() error {
	if p.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if p.System == "" {
		return fmt.Errorf("system is required")
	}
	return nil
}

type LambdaFunc func(ctx context.Context, e PopulateModuleVersionsEvent) (string, error)

func setupLogging(e PopulateModuleVersionsEvent) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger = logger.
		With("namespace", e.Namespace).
		With("name", e.Name).
		With("system", e.System)
	slog.SetDefault(logger)
}

func HandleRequest(config *config.Config) LambdaFunc {
	return func(ctx context.Context, e PopulateModuleVersionsEvent) (string, error) {
		setupLogging(e)

		var versions modules.VersionList

		slog.Info("Populating module versions")
		err := xray.Capture(ctx, "populate_module_versions.handle", func(tracedCtx context.Context) error {
			xray.AddAnnotation(tracedCtx, "namespace", e.Namespace)
			xray.AddAnnotation(tracedCtx, "name", e.Name)
			xray.AddAnnotation(tracedCtx, "system", e.System)

			err := e.Validate()
			if err != nil {
				slog.Error("invalid event", "error", err)
				return fmt.Errorf("invalid event: %w", err)
			}

			repoName := modules.GetRepoName(e.System, e.Name)

			// check if the document exists in dynamodb, if it does, and it's newer than the allowed max age,
			// we should treat it as a noop and just return
			document, err := config.ModuleVersionCache.GetItem(tracedCtx, fmt.Sprintf("%s/%s", e.Namespace, repoName))
			if err != nil {
				// if there was an error getting the document, that's fine. we'll just log it and carry on
				slog.Error("Error getting document from cache", "error", err)
			}
			if document != nil {
				if !document.IsStale() {
					slog.Info("Document is up to date, not updating")
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
			slog.Error("Error fetching versions", "error", err)
			return "", err
		}

		err = storeVersions(ctx, e, versions, config)
		if err != nil {
			return "", err
		}

		return "", nil
	}
}

func storeVersions(ctx context.Context, e PopulateModuleVersionsEvent, versions modules.VersionList, config *config.Config) error {
	if len(versions) == 0 {
		slog.Error("No versions found, skipping storage")
		return fmt.Errorf("no versions found")
	}
	// Construct the repo name.
	repoName := modules.GetRepoName(e.System, e.Name)

	key := fmt.Sprintf("%s/%s", e.Namespace, repoName)

	err := config.ModuleVersionCache.Store(ctx, key, versions)
	if err != nil {
		return fmt.Errorf("failed to store module listing: %w", err)
	}
	return nil
}

func fetchFromGithub(ctx context.Context, e PopulateModuleVersionsEvent, config *config.Config) (modules.VersionList, error) {
	// Construct the repo name.
	repoName := modules.GetRepoName(e.System, e.Name)

	// check the repo exists
	exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, e.Namespace, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("repo %s/%s does not exist", e.Namespace, repoName)
	}

	slog.Info("Fetching versions")

	v, err := modules.GetVersions(ctx, config.RawGithubv4Client, e.Namespace, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	return v, nil
}
