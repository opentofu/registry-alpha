package providers

import (
	"context"
	"encoding/json"
	"io"

	"github.com/opentffoundation/registry/internal/github"
)

type Manifest struct {
	Version  float64          `json:"version"`
	Metadata ManifestMetadata `json:"metadata"`
}
type ManifestMetadata struct {
	ProtocolVersions []string `json:"protocol_versions"`
}

func findAndParseManifest(ctx context.Context, assets []github.ReleaseAsset) (*Manifest, error) {
	manifestAsset := github.FindAssetBySuffix(assets, "_manifest.json")
	if manifestAsset == nil {
		return nil, nil //nolint:nilnil // This is not an error, it just means there is no manifest.
	}

	assetContents, err := github.DownloadAssetContents(ctx, manifestAsset.DownloadURL)
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
