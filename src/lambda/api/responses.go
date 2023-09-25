package main

import "github.com/aws/aws-lambda-go/events"

//nolint:gochecknoglobals // This should be treated as a constant.
var NotFoundResponse = events.APIGatewayProxyResponse{StatusCode: 404, Body: `{"errors":["not found"]}`}
