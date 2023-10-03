package config

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	gogithub "github.com/google/go-github/v54/github"
	"github.com/opentofu/registry/internal/modules/modulecache"
	"github.com/opentofu/registry/internal/providers/providercache"
	"github.com/opentofu/registry/internal/secrets"
	"github.com/shurcooL/githubv4"
)

type Config struct {
	ManagedGithubClient *gogithub.Client
	RawGithubv4Client   *githubv4.Client

	LambdaClient         *lambda.Client
	ProviderVersionCache *providercache.Handler
	ModuleVersionCache   *modulecache.Handler
	SecretsHandler       *secrets.Handler

	ProviderRedirects map[string]string
}
