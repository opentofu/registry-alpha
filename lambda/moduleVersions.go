package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"

	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/modules"
)

type ListModuleVersionsPathParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	System    string `json:"system"`
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

func listModuleVersions(config Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getListModuleVersionsPathParams(req)
		repoName := modules.GetRepoName(params.System, params.Name)
		log.Printf("[INFO] Request for %s/%s", params.Namespace, repoName)
		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, params.Namespace, repoName)
		if err != nil {
			log.Printf("[ERROR] Something bad happened - %s", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		if !exists {
			log.Printf("[WARN] Did not find such repository %s/%s", params.Namespace, repoName)
			return NotFoundResponse, nil
		}

		// fetch all the versions
		log.Printf("[INFO] Fetching verions for %s/%s", params.Namespace, repoName)
		versions, err := modules.GetVersions(ctx, config.RawGithubv4Client, params.Namespace, repoName)
		if err != nil {
			log.Printf("[ERROR] Something bad happened while fetching versions - %s", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		response := ListModuleVersionsResponse{
			Modules: []ModulesResponse{
				{
					Versions: versions,
				},
			},
		}
		log.Printf("[INFO] Recieved %v versions for %s/%s", len(versions), params.Namespace, repoName)

		resBody, err := json.Marshal(response)
		if err != nil {
			log.Printf("[ERROR] Something bad happened while trying to marshel response - %s", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
	}
}
