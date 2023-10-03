package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/opentofu/registry/internal/config"
	"golang.org/x/exp/slog"

	"github.com/aws/aws-lambda-go/events"

	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/modules"
)

type DownloadModuleHandlerPathParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	System    string `json:"system"`
	Version   string `json:"version"`
}

func (p DownloadModuleHandlerPathParams) AnnotateLogger() {
	logger := slog.Default()
	logger = logger.
		With("namespace", p.Namespace).
		With("name", p.Name).
		With("system", p.System).
		With("version", p.Version)
	slog.SetDefault(logger)
}

func downloadModuleVersion(config config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getDownloadModuleHandlerPathParams(req)
		params.AnnotateLogger()
		effectiveNamespace := config.EffectiveProviderNamespace(params.Namespace)
		repoName := modules.GetRepoName(params.System, params.Name)

		key := fmt.Sprintf("%s/%s", params.Namespace, repoName)
		document, _ := config.ModuleVersionCache.GetItem(ctx, key)
		if document != nil {
			return processDocumentForModuleDownload(ctx, config, document, params)
		}

		// if we don't have the document, we should check that the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, params.Namespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		if !exists {
			return NotFoundResponse, nil
		}

		releaseTag, err := getReleaseTag(ctx, config, effectiveNamespace, repoName, params.Version)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "", Headers: map[string]string{
			"X-Terraform-Get": fmt.Sprintf("git::https://github.com/%s/%s?ref=%s", params.Namespace, repoName, releaseTag),
		}}, nil
	}
}

func processDocumentForModuleDownload(ctx context.Context, config config.Config, document *modules.CacheItem, params DownloadModuleHandlerPathParams) (events.APIGatewayProxyResponse, error) {
	if document.IsStale() {
		err := triggerPopulateModuleVersions(ctx, config, params.Namespace, params.Name, params.System)
		if err != nil {
			// just log the error and still return the document
			slog.Error("Error triggering populate module versions", "error", err)
		}
	}

	version, ok := document.Versions.FindVersion(params.Version)
	if !ok {
		return NotFoundResponse, nil
	}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "", Headers: map[string]string{
		"X-Terraform-Get": version.DownloadURL,
	}}, nil
}

func getDownloadModuleHandlerPathParams(req events.APIGatewayProxyRequest) DownloadModuleHandlerPathParams {
	return DownloadModuleHandlerPathParams{
		Namespace: req.PathParameters["namespace"],
		Name:      req.PathParameters["name"],
		System:    req.PathParameters["system"],
		Version:   req.PathParameters["version"],
	}
}

func getReleaseTag(ctx context.Context, config config.Config, namespace string, repoName string, version string) (string, error) {
	versionWithPrefix := fmt.Sprintf("v%s", version)
	release, err := github.FindRelease(ctx, config.RawGithubv4Client, namespace, repoName, versionWithPrefix)
	if err != nil {
		return "", err
	}

	// If the release exists, then the tag does have the "v" prefix
	// If it does not, then we assume the tag exists without the "v" prefix
	if release != nil {
		return versionWithPrefix, nil
	}

	return version, nil
}
