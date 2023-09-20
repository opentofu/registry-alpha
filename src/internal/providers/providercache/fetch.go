package providercache

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (p *Handler) GetItem(ctx context.Context, key string) (*VersionListingItem, error) {
	result, err := p.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: p.TableName,
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

	var item VersionListingItem
	err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}
