package main

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

//nolint:gochecknoglobals // This should be treated as a constant.
var NotFoundResponse = events.APIGatewayProxyResponse{StatusCode: http.StatusNotFound, Body: `{"errors":["not found"]}`}
