package providers

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/platform"
)

func getShaSum(ctx context.Context, downloadURL string, filename string) (shaSum string, err error) {
	err = xray.Capture(ctx, "filename.shasum", func(tracedCtx context.Context) error {
		xray.AddAnnotation(tracedCtx, "filename", filename)

		assetContents, assetErr := github.DownloadAssetContents(tracedCtx, downloadURL)
		if assetErr != nil {
			return fmt.Errorf("failed to download asset contents: %w", assetErr)
		}

		contents, contentsErr := io.ReadAll(assetContents)
		if err != nil {
			return fmt.Errorf("failed to read asset contents: %w", contentsErr)
		}

		lines := strings.Split(string(contents), "\n")
		for _, line := range lines {
			if strings.HasSuffix(line, filename) {
				shaSum = strings.Split(line, " ")[0]
				break
			}
		}

		return nil
	})

	return
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
