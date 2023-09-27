package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opentofu/registry/internal/config"
	"golang.org/x/exp/slog"

	"github.com/aws/aws-lambda-go/events"

	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/providers"
)

type DownloadHandlerPathParams struct {
	Architecture string `json:"arch"`
	OS           string `json:"os"`
	Namespace    string `json:"namespace"`
	Type         string `json:"type"`
	Version      string `json:"version"`
}

func (p DownloadHandlerPathParams) AnnotateLogger() {
	logger := slog.Default()
	logger = logger.
		With("namespace", p.Namespace).
		With("type", p.Type).
		With("version", p.Version).
		With("os", p.OS).
		With("arch", p.Architecture)
	slog.SetDefault(logger)
}

func getDownloadPathParams(req events.APIGatewayProxyRequest) DownloadHandlerPathParams {
	return DownloadHandlerPathParams{
		Architecture: req.PathParameters["arch"],
		OS:           req.PathParameters["os"],
		Namespace:    req.PathParameters["namespace"],
		Type:         req.PathParameters["type"],
		Version:      req.PathParameters["version"],
	}
}

func downloadProviderVersion(config config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getDownloadPathParams(req)
		params.AnnotateLogger()
		effectiveNamespace := config.EffectiveProviderNamespace(params.Namespace)

		// Construct the repo name.
		repoName := providers.GetRepoName(params.Type)

		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, effectiveNamespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}
		if !exists {
			return NotFoundResponse, nil
		}

		versionDownloadResponse, err := providers.GetVersion(ctx, config.RawGithubv4Client, effectiveNamespace, repoName, params.Version, params.OS, params.Architecture)
		if err != nil {
			// log the error too for dev
			slog.Error("Error getting version", "error", err)
			var ghErr *github.ProviderError
			if errors.As(err, &ghErr) {
				return events.APIGatewayProxyResponse{StatusCode: ghErr.Code}, err
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		resBody, err := json.Marshal(versionDownloadResponse)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}
		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: string(resBody)}, nil
	}
}
