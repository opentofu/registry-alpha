package providers

import (
	"context"
	"fmt"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/opentffoundation/registry/internal/platform"
	"io"
	"strings"
)

func getShaSum(ctx context.Context, downloadURL string, filename string) (string, error) {
	assetContents, err := github.DownloadAssetContents(ctx, downloadURL)
	if err != nil {
		return "", err
	}

	contents, err := io.ReadAll(assetContents)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		if strings.HasSuffix(line, filename) {
			return strings.Split(line, " ")[0], nil
		}
	}

	return "", fmt.Errorf("could not find shasum for %s", filename)
}

func getSupportedArchAndOS(assets []github.ReleaseAsset) ([]platform.Platform, error) {
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
