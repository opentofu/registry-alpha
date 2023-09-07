package main

import "github.com/aws/aws-lambda-go/events"

var NotFoundResponse = events.APIGatewayProxyResponse{StatusCode: 404, Body: `{"errors":["not found"]}`}
