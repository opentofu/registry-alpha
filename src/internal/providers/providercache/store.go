package providercache

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/opentofu/registry/internal/providers/types"
	"golang.org/x/exp/slog"
)

type CompressedCacheItem struct {
	Provider    string    `dynamodbav:"provider"`
	Data        string    `dynamodbav:"data"`
	LastUpdated time.Time `dynamodbav:"last_updated"`
}

func compress(data []byte) (string, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(data)
	if err != nil {
		return "", err
	}
	err = gz.Close()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

func (p *Handler) Store(ctx context.Context, key string, versions types.VersionList) error {
	jsonData, err := json.Marshal(versions)
	if err != nil {
		slog.Error("got error marshalling item to JSON", "error", err)
		return fmt.Errorf("got error marshalling item to JSON: %w", err)
	}

	compressedData, err := compress(jsonData)
	if err != nil {
		slog.Error("got error compressing JSON data", "error", err)
		return fmt.Errorf("got error compressing JSON data: %w", err)
	}

	// make an anonymous type to satisfy the MarshalMap function
	toCache := CompressedCacheItem{
		Provider:    key,
		Data:        compressedData,
		LastUpdated: time.Now(),
	}

	marshalledItem, err := attributevalue.MarshalMap(toCache)
	if err != nil {
		slog.Error("got error marshalling dynamodb item", "error", err)
		return fmt.Errorf("got error marshalling dynamodb item: %w", err)
	}

	putItemInput := &dynamodb.PutItemInput{
		Item:      marshalledItem,
		TableName: p.TableName,
	}

	slog.Info("Storing provider versions", "key", key, "versions", len(versions))
	_, err = p.Client.PutItem(ctx, putItemInput)
	if err != nil {
		slog.Error("got error calling PutItem", "error", err)
		return fmt.Errorf("got error calling PutItem: %w", err)
	}

	slog.Info("Successfully stored provider versions", "key", key, "versions", len(versions))
	return nil
}
