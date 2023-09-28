package github

import (
	"context"
	"net/http"
	"net/url"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/google/go-github/v54/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func getGithubOauth2Client(token string) *http.Client {
	return xray.Client(oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)))
}

func NewManagedGithubClient(token string) *github.Client {
	client := github.NewClient(getGithubOauth2Client(token))
	client.BaseURL, _ = url.Parse("https://registry.opentofu.org/github/rest/")
	return client
}

func NewRawGithubv4Client(token string) *githubv4.Client {
	return githubv4.NewEnterpriseClient("https://registry.opentofu.org/github/graphql/", getGithubOauth2Client(token))
}
