package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

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
	client.BaseURL, _ = url.Parse(fmt.Sprintf("https://%s/github/rest/", os.Getenv("GITHUB_API_GW_URL")))
	return client
}

func NewRawGithubv4Client(token string) *githubv4.Client {
	return githubv4.NewEnterpriseClient(fmt.Sprintf("https://%s/github/graphql/", os.Getenv("GITHUB_API_GW_URL")), getGithubOauth2Client(token))
}
