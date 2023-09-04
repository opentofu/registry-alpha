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
	Type         string `json:"type"`
	Version      string `json:"version"`
}

func downloadProviderVersion(config Config) LambdaFunc {
	return func(_ context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getDownloadPathParams(req)

		reqJson, _ := json.Marshal(params)
		fmt.Println(string(reqJson))

		time := time.Now().UTC()
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: fmt.Sprintf("Provider download, generated at %s", time.String())}, nil
	}
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
