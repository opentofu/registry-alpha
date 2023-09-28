package providercache

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// AllowedAge is the maximum age of a cache item before it is considered stale.
const AllowedAge = (1 * time.Hour) - (5 * time.Minute) //nolint:gomnd // 55 minutes

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
