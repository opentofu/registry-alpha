package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/opentofu/registry/internal/config"
	"golang.org/x/exp/slog"

	"github.com/aws/aws-lambda-go/events"

	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/modules"
)

type ListModuleVersionsPathParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	System    string `json:"system"`
}

func (p ListModuleVersionsPathParams) AnnotateLogger() {
	logger := slog.Default()
	logger = logger.
		With("namespace", p.Namespace).
		With("name", p.Name).
		With("system", p.System)
	slog.SetDefault(logger)
}

func getListModuleVersionsPathParams(req events.APIGatewayProxyRequest) ListModuleVersionsPathParams {
	return ListModuleVersionsPathParams{
		Namespace: req.PathParameters["namespace"],
		Name:      req.PathParameters["name"],
		System:    req.PathParameters["system"],
	}
}

type ListModuleVersionsResponse struct {
	Modules []ModulesResponse `json:"modules"`
}

type ModulesResponse struct {
	Versions []modules.Version `json:"versions"`
}

func listModuleVersions(config config.Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getListModuleVersionsPathParams(req)
		params.AnnotateLogger()
		repoName := modules.GetRepoName(params.System, params.Name)

		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, params.Namespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}
		if !exists {
			return NotFoundResponse, nil
		}

		// fetch all the versions
		versions, err := modules.GetVersions(ctx, config.RawGithubv4Client, params.Namespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		response := ListModuleVersionsResponse{
			Modules: []ModulesResponse{
				{
					Versions: versions,
				},
			},
		}

		resBody, err := json.Marshal(response)
		if err != nil {
			slog.Error("Error marshalling response", "error", err)
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}
		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: string(resBody)}, nil
	}
}
