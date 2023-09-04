package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/v54/github"
	"github.com/opentffoundation/registry/internal/platform"
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

func GetVersions(ctx context.Context, ghClient *github.Client, namespace string, name string) ([]Version, error) {

	// the repo name should match the format `terraform-provider-<name>`
	repoName := fmt.Sprintf("terraform-provider-%s", name)

	releases, err := fetchReleases(ctx, ghClient, namespace, repoName)
	if err != nil {
		return nil, err
	}

	var versions []Version
	for _, release := range releases {
		platforms, err := getSupportedArchAndOS(release)
		if err != nil {
			return nil, err
		}

		manifest, err := findAndParseManifest(ctx, ghClient, namespace, repoName, release.Assets)
		if err != nil {
			return nil, err
		}

		versionName := *release.TagName
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

func fetchReleases(ctx context.Context, ghClient *github.Client, namespace string, name string) ([]*github.RepositoryRelease, error) {
	releases, _, err := ghClient.Repositories.ListReleases(ctx, namespace, name, nil)
	if err != nil {
		return nil, err
	}
	return releases, nil
}

func findAndParseManifest(ctx context.Context, ghClient *github.Client, namespace string, name string, assets []*github.ReleaseAsset) (*Manifest, error) {
	for _, asset := range assets {
		if strings.HasSuffix(*asset.Name, "_manifest.json") {

			assetContents, err := downloadAssetContents(ctx, ghClient, namespace, name, *asset.ID)
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

func downloadAssetContents(ctx context.Context, ghClient *github.Client, namespace string, name string, assetID int64) (io.ReadCloser, error) {
	httpClient := &http.Client{Timeout: 60 * time.Second}

	assetContents, downloadUrl, err := ghClient.Repositories.DownloadReleaseAsset(ctx, namespace, name, assetID, httpClient)
	if err != nil {
		return nil, err
	}

	if assetContents == nil && downloadUrl != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadUrl, nil)
		if err != nil {
			return nil, err
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		assetContents = resp.Body
	}

	if assetContents == nil {
		return nil, fmt.Errorf("Unable to download github asset contents for assetID : %s", assetID)
	}

	return assetContents, nil
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

func getSupportedArchAndOS(release *github.RepositoryRelease) ([]platform.Platform, error) {
	if release == nil {
		return nil, nil
	}

	var platforms []platform.Platform
	for _, asset := range release.Assets {
		platform := platform.ExtractPlatformFromArtifact(*asset.Name)
		if platform == nil {
			continue
		}
		platforms = append(platforms, *platform)
	}
	return platforms, nil
}
