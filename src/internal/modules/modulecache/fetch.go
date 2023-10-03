package modulecache

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/opentofu/registry/internal/modules"
	"golang.org/x/exp/slog"
)

func decompress(data string) ([]byte, error) {
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	rdata := bytes.NewReader(decodedData)
	r, err := gzip.NewReader(rdata)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

func (p *Handler) GetItem(ctx context.Context, key string) (*modules.CacheItem, error) {
	slog.Info("Getting item from cache", "key", key)

	result, err := p.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: p.TableName,
		Key: map[string]types.AttributeValue{
			"module": &types.AttributeValueMemberS{Value: key},
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

	var compressedItem CompressedCacheItem
	err = attributevalue.UnmarshalMap(result.Item, &compressedItem)
	if err != nil {
		slog.Error("Failed to unmarshal compressed item from cache", "key", key, "error", err)
		return nil, err
	}

	decompressedData, err := decompress(compressedItem.Data)
	if err != nil {
		slog.Error("Failed to decompress item data", "key", key, "error", err)
		return nil, err
	}

	var item modules.CacheItem
	err = json.Unmarshal(decompressedData, &item.Versions)
	if err != nil {
		slog.Error("Failed to unmarshal decompressed item to CacheItem", "key", key, "error", err)
		return nil, err
	}

	item.Module = compressedItem.Module
	item.LastUpdated = compressedItem.LastUpdated

	slog.Info("Successfully decompressed and unmarshalled item from cache", "key", key)
	return &item, nil
}
