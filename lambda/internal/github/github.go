package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/google/go-github/v54/github"
	"github.com/shurcooL/githubv4"
)

// GHRepository encapsulates GitHub repository details with a focus on its releases.
// This is structured to align with the expected response format from GitHub's GraphQL API.
type GHRepository struct {
	Repository struct {
		Releases struct {
			PageInfo struct {
				HasNextPage bool   // Indicates if there are more pages of releases.
				EndCursor   string // The cursor for pagination.
			}
			Nodes []GHRelease // A list of GitHub releases.
		} `graphql:"releases(first: $perPage, orderBy: {field: CREATED_AT, direction: DESC}, after: $endCursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

// GHRelease represents a release on GitHub.
// This provides details about the release, including its tag name, release assets, and its release status (draft, prerelease, etc.).
type GHRelease struct {
	ID            string // The ID of the release.
	TagName       string // The tag name associated with the release.
	ReleaseAssets struct {
		Nodes []ReleaseAsset // A list of assets for the release.
	} `graphql:"releaseAssets(first:100)"`
	IsDraft      bool     // Indicates if the release is a draft.
	IsLatest     bool     // Indicates if the release is the latest.
	IsPrerelease bool     // Indicates if the release is a prerelease.
	TagCommit    struct { // The commit associated with the release tag.
		TarballUrl string // The URL to download the release tarball.
	}
}

// ReleaseAsset represents a single asset within a GitHub release.
// This includes details such as the download URL and the name of the asset.
type ReleaseAsset struct {
	ID          string // The ID of the asset.
	DownloadURL string // The URL to download the asset.
	Name        string // The name of the asset.
}

func RepositoryExists(ctx context.Context, managedGhClient *github.Client, namespace, name string) (bool, error) {
	_, response, err := managedGhClient.Repositories.Get(ctx, namespace, name)
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to get repository: %v", err)
	}
	return true, nil
}

func FindRelease(ctx context.Context, ghClient *githubv4.Client, namespace, name, versionNumber string) (*GHRelease, error) {
	variables := initVariables(namespace, name)
	for {
		nodes, endCursor, err := FetchReleaseNodes(ctx, ghClient, variables)
		if err != nil {
			return nil, err
		}
		for _, r := range nodes {
			if r.IsDraft || r.IsPrerelease {
				continue
			}
			if r.TagName == fmt.Sprintf("v%s", versionNumber) {
				return &r, nil
			}
		}
		if endCursor == nil {
			break
		}
		variables["endCursor"] = githubv4.String(*endCursor)
	}
	return nil, nil
}

func FetchReleases(ctx context.Context, ghClient *githubv4.Client, namespace, name string) ([]GHRelease, error) {
	variables := initVariables(namespace, name)
	var releases []GHRelease
	for {
		nodes, endCursor, err := FetchReleaseNodes(ctx, ghClient, variables)
		if err != nil {
			return nil, err
		}

		for _, r := range nodes {
			if r.IsDraft || r.IsPrerelease {
				continue
			}
			releases = append(releases, r)
		}
		if endCursor == nil {
			break
		}
		variables["endCursor"] = githubv4.String(*endCursor)
	}
	return releases, nil
}

func initVariables(namespace, name string) map[string]interface{} {
	perPage := 100 // TODO: make this configurable
	return map[string]interface{}{
		"owner":     githubv4.String(namespace),
		"name":      githubv4.String(name),
		"perPage":   githubv4.Int(perPage),
		"endCursor": (*githubv4.String)(nil),
	}
}

// FetchReleaseNodes will fetch a page of releases from the github api and return the nodes, endCursor, and an error
// endCursor will be nil if there are no more pages
func FetchReleaseNodes(ctx context.Context, ghClient *githubv4.Client, variables map[string]interface{}) ([]GHRelease, *string, error) {
	var query GHRepository
	if err := ghClient.Query(ctx, &query, variables); err != nil {
		return nil, nil, err
	}
	var endCursor *string
	if query.Repository.Releases.PageInfo.HasNextPage {
		endCursor = &query.Repository.Releases.PageInfo.EndCursor
	}
	return query.Repository.Releases.Nodes, endCursor, nil
}

func FindAssetBySuffix(assets []ReleaseAsset, suffix string) *ReleaseAsset {
	for _, asset := range assets {
		if strings.HasSuffix(asset.Name, suffix) {
			return &asset
		}
	}
	return nil
}

func DownloadAssetContents(ctx context.Context, downloadURL string) (io.ReadCloser, error) {
	httpClient := xray.Client(&http.Client{Timeout: 60 * time.Second})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download asset, status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
