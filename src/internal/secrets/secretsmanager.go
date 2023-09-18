package secrets

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"os"
)

func GetClient(ctx context.Context) (*secretsmanager.Client, error) {
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return nil, fmt.Errorf("could not load AWS configuration: %w", err)
	}

	awsv2.AWSV2Instrumentor(&awsConfig.APIOptions)

	return secretsmanager.NewFromConfig(awsConfig), nil
}

func GetValue(ctx context.Context, sm *secretsmanager.Client, secretName string) (string, error) {
	value, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return "", err
	}
	return *value.SecretString, nil
}
