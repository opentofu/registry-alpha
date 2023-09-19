package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/providers"
	"github.com/opentffoundation/registry/internal/providers/providercache"
	"os"
)

type ListProvidersPathParams struct {
	Namespace string `json:"namespace"`
	Type      string `json:"name"`
}

func getListProvidersPathParams(req events.APIGatewayProxyRequest) ListProvidersPathParams {
	return ListProvidersPathParams{
		Namespace: req.PathParameters["namespace"],
		Type:      req.PathParameters["type"],
	}
}

type ListProviderVersionsResponse struct {
	Versions []providers.Version `json:"versions"`
}

func listProviderVersions(config Config) LambdaFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		params := getListProvidersPathParams(req)
		effectiveNamespace := config.EffectiveProviderNamespace(params.Namespace)

		// Construct the repo name.
		repoName := providers.GetRepoName(params.Type)

		// check the repo exists
		exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, effectiveNamespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		if !exists {
			return NotFoundResponse, nil
		}

		// TODO: Move this to a shared package that can be used across all lambdas.
		cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(os.Getenv("AWS_REGION")))
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("could not load AWS configuration: %w", err)
		}

		ddbClient := dynamodb.NewFromConfig(cfg)

		tableName := os.Getenv("PROVIDER_VERSIONS_TABLE_NAME")
		if tableName == "" {
			panic(fmt.Errorf("missing environment variable PROVIDER_VERSIONS_TABLE_NAME"))
		}

		document, err := getDocument(ctx, ddbClient, tableName, fmt.Sprintf("%s/%s", effectiveNamespace, params.Type))
		if err != nil {
			// log the error but carry on. If there is an error fetching the document, we'll just fetch it from github.
			// we want to be fault-tolerant here so that we don't fail to serve the request if there is an error
			// fetching the document.
			fmt.Printf("Error fetching document from dynamodb: %s\n", err.Error())
		}
		if document != nil {
			// TODO: If the document is more than an hour old, invoke the lambda to update it.

			fmt.Printf("Found document with %d versions\n", len(document.Versions))
			return foundDocumentResponse(document)
		}

		fmt.Printf("Document not found in dynamodb, invoking lambda and loading from github\n")
		lambdaClient := lambda.NewFromConfig(cfg)

		fmt.Printf("Invoking populate provider versions lambda asynchronously to update dynamodb document\n")
		// invoke the async lambda to update the dynamodb document
		_, err = lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName:   aws.String(os.Getenv("POPULATE_PROVIDER_VERSIONS_FUNCTION_NAME")),
			InvocationType: "Event", // Event == async
			Payload:        []byte(fmt.Sprintf("{\"namespace\": \"%s\", \"type\": \"%s\"}", effectiveNamespace, params.Type)),
		})
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		versions, err := providers.GetVersions(ctx, config.RawGithubv4Client, effectiveNamespace, repoName)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		response := ListProviderVersionsResponse{
			Versions: versions,
		}

		resBody, err := json.Marshal(response)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
	}
}

func foundDocumentResponse(document *providercache.VersionListingItem) (events.APIGatewayProxyResponse, error) {
	// we found the document, return it
	response := ListProviderVersionsResponse{
		Versions: document.Versions,
	}

	resBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(resBody)}, nil
}

func getDocument(ctx context.Context, client *dynamodb.Client, tableName string, key string) (*providercache.VersionListingItem, error) {
	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"provider": &types.AttributeValueMemberS{Value: key},
		},
	})
	if err != nil {
		return nil, err
	}

	// check if the item is empty, if so return nil, this makes it easier to consume in other places
	if len(result.Item) == 0 {
		return nil, nil
	}

	// unmarshal the item
	var item providercache.VersionListingItem
	err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}
