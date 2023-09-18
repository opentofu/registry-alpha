package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/providers"
	"os"
	"time"
)

type PopulateProviderVersionsEvent struct {
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
}

type ProviderVersionListingItem struct {
	Provider    string              `json:"provider"`
	Versions    []providers.Version `json:"versions"`
	LastUpdated time.Time           `json:"last_updated"`
}

func (p PopulateProviderVersionsEvent) Validate() error {
	if p.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if p.Type == "" {
		return fmt.Errorf("type is required")
	}
	return nil
}

func main() {
	ctx := context.Background()
	config, err := buildConfig(ctx)
	if err != nil {
		panic(fmt.Errorf("could not build config: %w", err))
	}

	lambda.Start(HandleRequest(config))
}

func StoreProviderListingInDynamo(providerNamespace string, providerType string, versions []providers.Version) error {
	provider := fmt.Sprintf("%s/%s", providerNamespace, providerType)
	// Create Session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION"))},
	)

	ddbClient := dynamodb.New(sess)

	item := ProviderVersionListingItem{
		Provider:    provider,
		Versions:    versions,
		LastUpdated: time.Now(),
	}

	marshalledItem, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("Got error marshalling dynamodb item: %w", err)
	}

	// Create item in table Movies
	tableName := "provider-versions"

	input := &dynamodb.PutItemInput{
		Item:      marshalledItem,
		TableName: aws.String(tableName),
	}

	_, err = ddbClient.PutItem(input)
	if err != nil {
		return fmt.Errorf("got error calling PutItem: %w", err)
	}

	return nil
}

func HandleRequest(config *Config) func(ctx context.Context, e PopulateProviderVersionsEvent) (string, error) {
	return func(ctx context.Context, e PopulateProviderVersionsEvent) (string, error) {
		var versions []providers.Version

		fmt.Printf("Fetching %s/%s\n", e.Namespace, e.Type)
		err := xray.Capture(ctx, "populate_provider_versions.handle", func(tracedCtx context.Context) error {
			xray.AddAnnotation(tracedCtx, "namespace", e.Namespace)
			xray.AddAnnotation(tracedCtx, "type", e.Type)

			err := e.Validate()
			if err != nil {
				return fmt.Errorf("invalid event: %w", err)
			}

			// Construct the repo name.
			repoName := providers.GetRepoName(e.Type)

			// check the repo exists
			exists, err := github.RepositoryExists(ctx, config.ManagedGithubClient, e.Namespace, repoName)
			if err != nil {
				return fmt.Errorf("failed to check if repo exists: %w", err)
			}
			if !exists {
				return fmt.Errorf("repo does not exist")
			}

			fmt.Printf("Repo %s/%s exists\n", e.Namespace, repoName)

			v, err := providers.GetVersions(tracedCtx, config.RawGithubv4Client, e.Namespace, repoName)
			if err != nil {
				return fmt.Errorf("failed to get versions: %w", err)
			}

			fmt.Printf("Found %d versions\n", len(v))

			versions = v
			return nil
		})

		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
			return "", err
		}

		err = StoreProviderListingInDynamo(e.Namespace, e.Type, versions)
		if err != nil {
			return "", fmt.Errorf("failed to cache provider listing: %w", err)
		}

		return "", nil
	}
}
