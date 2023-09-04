package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shurcooL/githubv4"
	"io"
	"net/http"
	"strings"
	"time"
)

func findRelease(ctx context.Context, ghClient *githubv4.Client, namespace, name, versionNumber string) (*GHRelease, error) {
	variables := initVariables(namespace, name)
	for {
		nodes, endCursor, err := fetchReleaseNodes(ctx, ghClient, variables)
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

func fetchReleases(ctx context.Context, ghClient *githubv4.Client, namespace, name string) ([]GHRelease, error) {
	variables := initVariables(namespace, name)
	var releases []GHRelease
	for {
		nodes, endCursor, err := fetchReleaseNodes(ctx, ghClient, variables)
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

// fetchReleaseNodes will fetch a page of releases from the github api and return the nodes, endCursor, and an error
// endCursor will be nil if there are no more pages
func fetchReleaseNodes(ctx context.Context, ghClient *githubv4.Client, variables map[string]interface{}) ([]GHRelease, *string, error) {
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

func findAssetBySuffix(assets []ReleaseAsset, suffix string) *ReleaseAsset {
	for _, asset := range assets {
		if strings.HasSuffix(asset.Name, suffix) {
			return &asset
		}
	}
	return nil
}

func findAndParseManifest(ctx context.Context, assets []ReleaseAsset) (*Manifest, error) {
	manifestAsset := findAssetBySuffix(assets, "_manifest.json")
	if manifestAsset == nil {
		return nil, nil
	}

	assetContents, err := downloadAssetContents(ctx, manifestAsset.DownloadURL)
	if err != nil {
		return nil, err
	}

	manifest, err := parseManifestContents(assetContents)
	assetContents.Close()
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func downloadAssetContents(ctx context.Context, downloadURL string) (io.ReadCloser, error) {
	httpClient := &http.Client{Timeout: 60 * time.Second}
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

func parseManifestContents(assetContents io.ReadCloser) (*Manifest, error) {
	contents, err := io.ReadAll(assetContents)
	if err != nil {
		return nil, err
	}

	var manifest *Manifest
	err = json.Unmarshal(contents, &manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}
