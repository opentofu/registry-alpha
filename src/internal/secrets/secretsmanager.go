package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Handler struct {
	client *secretsmanager.Client
}

func NewHandler(awsConfig aws.Config) *Handler {
	client := secretsmanager.NewFromConfig(awsConfig)
	return &Handler{client: client}
}

func (s *Handler) GetValue(ctx context.Context, secretName string) (string, error) {
	value, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return "", err
	}
	return *value.SecretString, nil
}

func (s *Handler) GetSecretValueFromEnvReference(ctx context.Context, envVarName string) (string, error) {
	envVarValue := os.Getenv(envVarName)
	if envVarValue == "" {
		return "", fmt.Errorf("%s environment variable not set", envVarName)
	}

	var value string
	value, err := s.GetValue(ctx, envVarValue)
	if err != nil {
		return "", fmt.Errorf("could not get secret: %w", err)
	}

	if value == "" {
		return "", fmt.Errorf("empty value fetched from secrets manager")
	}
	return value, nil
}
