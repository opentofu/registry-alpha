package github

import (
	"context"
	"net/http"

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
	client, err := github.NewEnterpriseClient("https://registry.opentofu.org/rest/", "https://uploads.github.com/", getGithubOauth2Client(token))
	if err != nil {
		panic("we got error")
	}
	return client
	//return github.NewClient(getGithubOauth2Client(token))
}

func NewRawGithubv4Client(token string) *githubv4.Client {
	return githubv4.NewEnterpriseClient("https://registry.opentofu.org/graphql/", getGithubOauth2Client(token))
}
