package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/opentffoundation/registry/internal/providers"
	"os"
	"time"
)

func storeProviderListingInDynamo(providerNamespace string, providerType string, versions []providers.Version) error {
	tableName := os.Getenv("PROVIDER_VERSIONS_TABLE_NAME")
	provider := fmt.Sprintf("%s/%s", providerNamespace, providerType)

	// Create AWS Session
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
		return fmt.Errorf("got error marshalling dynamodb item: %w", err)
	}

	putItemInput := &dynamodb.PutItemInput{
		Item:      marshalledItem,
		TableName: aws.String(tableName),
	}

	_, err = ddbClient.PutItem(putItemInput)
	if err != nil {
		return fmt.Errorf("got error calling PutItem: %w", err)
	}

	return nil
}
