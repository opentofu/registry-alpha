package providercache

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"golang.org/x/exp/slog"
)

func (p *Handler) GetItem(ctx context.Context, key string) (*VersionListingItem, error) {
	slog.Info("Getting item from cache", "key", key)

	result, err := p.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: p.TableName,
		Key: map[string]types.AttributeValue{
			"provider": &types.AttributeValueMemberS{Value: key},
		},
	})
	if err != nil {
		slog.Error("Failed to get item from cache", "key", key, "error", err)
		return nil, err
	}

	// check if the item is empty, if so return nil, this makes it easier to consume in other places
	if len(result.Item) == 0 {
		slog.Info("Item not found in cache", "key", key)
		return nil, nil //nolint:nilnil // This is not an error, it just means there is no manifest.
	}

	var item VersionListingItem
	err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		slog.Error("Failed to unmarshal item from cache", "key", key, "error", err)
		return nil, err
	}

	slog.Info("Got item from cache", "key", key)
	return &item, nil
}
