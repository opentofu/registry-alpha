package versions_cache

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/opentffoundation/registry/internal/providers"
	"os"
	"time"
)

type ProviderVersionListingItem struct {
	Provider    string              `dynamodbav:"provider"`
	Versions    []providers.Version `dynamodbav:"versions"`
	LastUpdated time.Time           `dynamodbav:"last_updated"`
}

func StoreProviderListingInDynamo(ctx context.Context, providerNamespace string, providerType string, versions []providers.Version) error {
	tableName := os.Getenv("PROVIDER_VERSIONS_TABLE_NAME")
	if tableName == "" {
		panic(fmt.Errorf("missing environment variable PROVIDER_VERSIONS_TABLE_NAME"))
	}

	provider := fmt.Sprintf("%s/%s", providerNamespace, providerType)

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return fmt.Errorf("could not load AWS configuration: %w", err)
	}

	ddbClient := dynamodb.NewFromConfig(awsConfig)

	item := ProviderVersionListingItem{
		Provider:    provider,
		Versions:    versions,
		LastUpdated: time.Now(),
	}

	marshalledItem, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("got error marshalling dynamodb item: %w", err)
	}

	putItemInput := &dynamodb.PutItemInput{
		Item:      marshalledItem,
		TableName: aws.String(tableName),
	}

	_, err = ddbClient.PutItem(ctx, putItemInput)
	if err != nil {
		return fmt.Errorf("got error calling PutItem: %w", err)
	}

	return nil
}
