package providercache

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/opentffoundation/registry/internal/providers"
	"time"
)

type Handler struct {
	TableName *string
	Client    *dynamodb.Client
}

func NewHandler(awsConfig aws.Config, tableName string) *Handler {
	ddbClient := dynamodb.NewFromConfig(awsConfig)

	return &Handler{
		TableName: aws.String(tableName),
		Client:    ddbClient,
	}
}

type VersionListingItem struct {
	Provider    string              `dynamodbav:"provider"`
	Versions    []providers.Version `dynamodbav:"versions"`
	LastUpdated time.Time           `dynamodbav:"last_updated"`
}

func (p *Handler) Store(ctx context.Context, key string, versions []providers.Version) error {
	item := VersionListingItem{
		Provider:    key,
		Versions:    versions,
		LastUpdated: time.Now(),
	}

	marshalledItem, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("got error marshalling dynamodb item: %w", err)
	}

	putItemInput := &dynamodb.PutItemInput{
		Item:      marshalledItem,
		TableName: p.TableName,
	}

	_, err = p.Client.PutItem(ctx, putItemInput)
	if err != nil {
		return fmt.Errorf("got error calling PutItem: %w", err)
	}

	return nil
}
