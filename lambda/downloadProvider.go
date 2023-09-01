package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"time"
)

type DownloadHandlerPathParams struct {
	Architecture string `json:"arch"`
	OS           string `json:"os"`
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Version      string `json:"version"`
}

func downloadProviderVersion(_ context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	params := getDownloadPathParams(req)

	reqJson, _ := json.Marshal(params)
	fmt.Println(string(reqJson))

	time := time.Now().UTC()

	return events.APIGatewayProxyResponse{StatusCode: 200, Body: fmt.Sprintf("Hello World, generated at %s", time.String())}, nil
}

func getDownloadPathParams(req events.APIGatewayProxyRequest) DownloadHandlerPathParams {
	return DownloadHandlerPathParams{
		Architecture: req.PathParameters["arch"],
		OS:           req.PathParameters["os"],
		Namespace:    req.PathParameters["namespace"],
		Name:         req.PathParameters["name"],
		Version:      req.PathParameters["version"],
	}
}
