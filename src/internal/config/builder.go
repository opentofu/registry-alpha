package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/modules/modulecache"
	"github.com/opentofu/registry/internal/providers/providercache"
	"github.com/opentofu/registry/internal/secrets"
)

type Builder struct {
	IncludeProviderRedirects bool
	IncludeProviderCache     bool
	IncludeModuleCache       bool

	AWSConfig         aws.Config
	SecretsHandler    *secrets.Handler
	GithubAPIToken    string
	ProviderTableName string
	ModuleTableName   string
	ProviderRedirects map[string]string
}

func NewBuilder(options ...func(*Builder)) *Builder {
	configBuilder := &Builder{}
	for _, option := range options {
		option(configBuilder)
	}
	return configBuilder
}

func WithProviderRedirects() func(*Builder) {
	return func(builder *Builder) {
		builder.IncludeProviderRedirects = true
	}
}

func WithProviderCache() func(*Builder) {
	return func(builder *Builder) {
		builder.IncludeProviderCache = true
	}
}

func WithModuleCache() func(*Builder) {
	return func(builder *Builder) {
		builder.IncludeModuleCache = true
	}
}

func (b *Builder) SetupAWS(ctx context.Context) error {
	var err error
	b.AWSConfig, err = awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(os.Getenv("AWS_REGION")))
	return err
}

func (b *Builder) SetupSecrets() {
	b.SecretsHandler = secrets.NewHandler(b.AWSConfig)
}

func (b *Builder) SetupProviderCache() *providercache.Handler {
	if !b.IncludeProviderCache {
		return nil
	}

	providerTableName := os.Getenv("PROVIDER_VERSIONS_TABLE_NAME")
	if providerTableName == "" {
		panic("PROVIDER_VERSIONS_TABLE_NAME environment variable not set")
	}
	return providercache.NewHandler(b.AWSConfig, providerTableName)
}

func (b *Builder) SetupModuleCache() *modulecache.Handler {
	if !b.IncludeModuleCache {
		return nil
	}

	moduleTableName := os.Getenv("MODULE_VERSIONS_TABLE_NAME")
	if moduleTableName == "" {
		panic("MODULE_VERSIONS_TABLE_NAME environment variable not set")
	}
	return modulecache.NewHandler(b.AWSConfig, moduleTableName)
}

func (b *Builder) FetchGithubToken(ctx context.Context) error {
	var err error
	b.GithubAPIToken, err = b.SecretsHandler.GetSecretValueFromEnvReference(ctx, "GITHUB_TOKEN_SECRET_ASM_NAME")
	return err
}

func (b *Builder) SetupProviderRedirects() {
	if !b.IncludeProviderRedirects {
		return
	}

	if redirectsJSON, ok := os.LookupEnv("PROVIDER_NAMESPACE_REDIRECTS"); ok {
		if err := json.Unmarshal([]byte(redirectsJSON), &b.ProviderRedirects); err != nil {
			panic(fmt.Errorf("could not parse PROVIDER_NAMESPACE_REDIRECTS: %w", err))
		}
	}
}

func (b *Builder) BuildConfig(ctx context.Context, xraySegmentName string) (*Config, error) {
	var err error
	if err = xray.Configure(xray.Config{ServiceVersion: "1.2.3"}); err != nil {
		return nil, fmt.Errorf("could not configure X-Ray: %w", err)
	}

	ctx, segment := xray.BeginSegment(ctx, xraySegmentName)
	defer func() { segment.Close(err) }()

	if err = b.SetupAWS(ctx); err != nil {
		return nil, fmt.Errorf("could not load AWS configuration: %w", err)
	}

	b.SetupSecrets()

	if err = b.FetchGithubToken(ctx); err != nil {
		return nil, fmt.Errorf("could not get GitHub API token: %w", err)
	}

	b.SetupProviderRedirects()

	providerCache := b.SetupProviderCache()
	moduleCache := b.SetupModuleCache()

	return &Config{
		ManagedGithubClient:  github.NewManagedGithubClient(b.GithubAPIToken),
		RawGithubv4Client:    github.NewRawGithubv4Client(b.GithubAPIToken),
		SecretsHandler:       b.SecretsHandler,
		ProviderVersionCache: providerCache,
		ModuleVersionCache:   moduleCache,
		LambdaClient:         lambda.NewFromConfig(b.AWSConfig),
		ProviderRedirects:    b.ProviderRedirects,
	}, nil
}
