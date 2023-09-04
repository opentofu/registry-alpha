package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opentffoundation/registry/internal/platform"
	"github.com/shurcooL/githubv4"
	"io"
	"net/http"
	"strings"
	"time"
)

type Version struct {
	Version   string              `json:"version"`
	Protocols []string            `json:"protocols"`
	Platforms []platform.Platform `json:"platforms"`
}

func GetVersions(ctx context.Context, ghClient *githubv4.Client, namespace string, name string) ([]Version, error) {
	// the repo name should match the format `terraform-provider-<name>`
	repoName := fmt.Sprintf("terraform-provider-%s", name)

	releases, err := fetchReleases(ctx, ghClient, namespace, repoName)
	if err != nil {
		return nil, err
	}

	var versions []Version
	for _, release := range releases {
		assets := release.ReleaseAssets.Nodes
		// get the supported platforms for this release based on the filenames in the release assets
		platforms, err := getSupportedArchAndOS(assets)
		if err != nil {
			return nil, err
		}

		manifest, err := findAndParseManifest(ctx, assets)
		if err != nil {
			return nil, err
		}

		versionName := release.TagName
		if strings.HasPrefix(versionName, "v") {
			versionName = versionName[1:]
		}

		version := Version{
			Version:   versionName,
			Platforms: platforms,
		}
		if manifest != nil {
			version.Protocols = manifest.Metadata.ProtocolVersions
		} else {
			version.Protocols = []string{"5.0"}
		}

		versions = append(versions, version)
	}
	return versions, nil
}

type Release struct {
	TagName       string
	ReleaseAssets struct {
		Nodes []ReleaseAsset
	} `graphql:"releaseAssets(first:100)"`
	IsDraft      bool
	IsLatest     bool
	IsPrerelease bool
}

type ReleaseAsset struct {
	ID          string
	DownloadURL string
	Name        string
}

func fetchReleases(ctx context.Context, ghClient *githubv4.Client, namespace string, name string) ([]Release, error) {
	// Use the graphql api to fetch the release versions and their artifacts, we do this instead of the managed client call because
	// we want to reduce the amount of info we fetch from github
	type responseData struct {
		Repository struct {
			Releases struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				}
				Nodes []Release
			} `graphql:"releases(first: $perPage, orderBy: {field: CREATED_AT, direction: DESC}, after: $endCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	// hardcode page size for now
	// TODO: make this configurable
	perPage := 100

	variables := map[string]interface{}{
		"owner":     githubv4.String(namespace),
		"name":      githubv4.String(name),
		"perPage":   githubv4.Int(perPage),
		"endCursor": (*githubv4.String)(nil),
	}
	var releases []Release
	for {
		var query responseData
		err := ghClient.Query(ctx, &query, variables)
		if err != nil {
			return nil, err
		}

		for _, r := range query.Repository.Releases.Nodes {
			if r.IsDraft || r.IsPrerelease {
				continue
			}
			releases = append(releases, r)
		}

		if !query.Repository.Releases.PageInfo.HasNextPage {
			break
		}
		variables["endCursor"] = githubv4.String(query.Repository.Releases.PageInfo.EndCursor)
	}

	return releases, nil
}

func findAndParseManifest(ctx context.Context, assets []ReleaseAsset) (*Manifest, error) {
	for _, asset := range assets {
		if strings.HasSuffix(asset.Name, "_manifest.json") {
			assetContents, err := downloadAssetContents(ctx, asset.DownloadURL)
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
	}
	return nil, nil
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

func getSupportedArchAndOS(assets []ReleaseAsset) ([]platform.Platform, error) {
	var platforms []platform.Platform
	for _, asset := range assets {
		platform := platform.ExtractPlatformFromArtifact(asset.Name)
		if platform == nil {
			continue
		}
		platforms = append(platforms, *platform)
	}
	return platforms, nil
}
