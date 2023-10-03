package modulecache

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
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
