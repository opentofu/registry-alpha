package providers

import (
	"context"
	"encoding/json"
	"io"

	"github.com/opentofu/registry/internal/github"
	"golang.org/x/exp/slog"
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
		slog.Warn("No manifest found in release assets")
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

	slog.Info("Found manifest")

	return manifest, nil
}

func parseManifestContents(assetContents io.ReadCloser) (*Manifest, error) {
	contents, err := io.ReadAll(assetContents)
	if err != nil {
		slog.Error("Failed to read manifest contents")
		return nil, err
	}

	var manifest *Manifest
	err = json.Unmarshal(contents, &manifest)
	if err != nil {
		slog.Error("Failed to parse manifest contents")
		return nil, err
	}

	return manifest, nil
}
