package providers

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentofu/registry/internal/github"
	"github.com/opentofu/registry/internal/platform"
	"github.com/opentofu/registry/internal/providers/types"
	"github.com/shurcooL/githubv4"
	"golang.org/x/exp/slog"
)

type versionResult struct {
	Version types.CacheVersion
	Err     error
}

// GetVersions fetches and returns a list of available versions of a given provider hosted on GitHub.
// The returned versions also include information about supported platforms and the Terraform protocol versions they are compatible with.
//
// Parameters:
// - ctx: The context used to control cancellations and timeouts.
// - ghClient: The GitHub GraphQL client to interact with the GitHub GraphQL API.
// - namespace: The GitHub namespace (typically, the organization or user) under which the provider repository is hosted.
// - name: The name of the provider repository.
//
// Returns a slice of Version structures detailing each available version. If an error occurs during fetching or processing, it returns an error.
func GetVersions(ctx context.Context, ghClient *githubv4.Client, namespace string, name string) (versions types.VersionList, err error) {
	err = xray.Capture(ctx, "provider.versions", func(tracedCtx context.Context) error {
		xray.AddAnnotation(tracedCtx, "namespace", namespace)
		xray.AddAnnotation(tracedCtx, "name", name)

		slog.Info("Fetching versions")

		releases, releasesErr := github.FetchReleases(tracedCtx, ghClient, namespace, name)
		if releasesErr != nil {
			return fmt.Errorf("failed to fetch releases: %w", releasesErr)
		}

		versionCh := make(chan versionResult, len(releases))

		var wg sync.WaitGroup

		for _, release := range releases {
			wg.Add(1)
			go func(r github.GHRelease) {
				defer wg.Done()
				getVersionFromGithubRelease(tracedCtx, r, versionCh)
			}(release)
		}

		// Close the channel when all goroutines are done.
		wg.Wait()
		close(versionCh)

		for vr := range versionCh {
			if vr.Err != nil {
				slog.Error("Failed to process some releases", "error", vr.Err)
				// we should not fail the entire operation if we can't process a single release
				// this is because some GitHub releases may not have the correct assets attached,
				// and therefore we should just log and skip them
				xrayErr := xray.AddError(tracedCtx, fmt.Errorf("failed to process some releases: %w", vr.Err))
				if xrayErr != nil {
					return fmt.Errorf("failed to add error to trace: %w", err)
				}
			} else if vr.Version.Version != "" && len(vr.Version.DownloadDetails) > 0 {
				// only add the final list of versions if it's populated and has platforms attached
				versions = append(versions, vr.Version)
			}
		}
		return nil
	})

	slog.Info("Successfully found versions", "versions", len(versions))
	return versions, nil
}

// getVersionFromGithubRelease fetches and returns detailed information about a specific version of a provider hosted on GitHub.
// all results are passed back to the versionCh channel.
func getVersionFromGithubRelease(ctx context.Context, r github.GHRelease, versionCh chan versionResult) {
	result := versionResult{}

	logger := slog.Default().With("version", r.TagName)

	logger.Info("Processing release")

	assets := r.ReleaseAssets.Nodes
	platforms := getSupportedArchAndOS(assets)

	// if there are no platforms, we can't do anything with this release
	// so, we should just skip
	if len(platforms) == 0 {
		return
	}

	protocols := []string{"5.0"}

	logger.Info("Fetching manifest")
	// Read the manifest so that we can get the protocol versions.
	manifest, manifestErr := findAndParseManifest(ctx, assets)
	if manifestErr != nil {
		logger.Error("Failed to find and parse manifest", "error", manifestErr)
		result.Err = fmt.Errorf("failed to find and parse manifest: %w", manifestErr)
		versionCh <- result
		return
	}

	// attach the protocol versions to the version result
	if manifest != nil {
		slog.Info("Found manifest", "protocols", manifest.Metadata.ProtocolVersions)
		protocols = manifest.Metadata.ProtocolVersions
	}

	slog.Info("Fetching shasums")
	// download the shasums file so that we can get the checksum for each platform
	shaSums, err := downloadShaSums(ctx, assets)
	if err != nil {
		slog.Error("Failed to download shasums", "error", err)
		result.Err = fmt.Errorf("failed to download shasums: %w", err)
		versionCh <- result
		return
	}

	slog.Info("Found shasums", "shasums", len(shaSums))

	shaSumsURL := github.FindAssetBySuffix(assets, "_SHA256SUMS")
	shaSumsSignatureURL := github.FindAssetBySuffix(assets, "_SHA256SUMS.sig")

	if shaSumsSignatureURL == nil {
		// make an empty one
		shaSumsSignatureURL = &github.ReleaseAsset{
			DownloadURL: "",
		}
	}

	downloadDetails := make([]types.CacheVersionDownloadDetails, 0, len(platforms))
	// for each of the supported platforms, we need to find the appropriate assets
	// and add them to the version result
	for _, platform := range platforms {
		slog.Info("Fetching download details", "platform", fmt.Sprintf("%s_%s", platform.OS, platform.Arch))
		details := getVersionDownloadDetails(platform, assets, shaSums)
		if details != nil {
			details.SHASumsURL = shaSumsURL.DownloadURL
			details.SHASumsSignatureURL = shaSumsSignatureURL.DownloadURL
			downloadDetails = append(downloadDetails, *details)
		}
	}

	// only populate the version if we have all download details
	result.Version = types.CacheVersion{
		Version:         strings.TrimPrefix(r.TagName, "v"),
		Protocols:       protocols,
		DownloadDetails: downloadDetails,
	}

	versionCh <- result
}

func getVersionDownloadDetails(platform platform.Platform, assets []github.ReleaseAsset, shaSums map[string]string) *types.CacheVersionDownloadDetails {
	// find the asset for the given platform
	asset := github.FindAssetBySuffix(assets, fmt.Sprintf("_%s_%s.zip", platform.OS, platform.Arch))
	if asset == nil {
		slog.Warn("Could not find asset for platform", "platform", platform)
		return nil
	}

	// get the shasum for the asset
	shasum, ok := shaSums[asset.Name]
	if !ok {
		slog.Warn("Could not find shasum for asset", "asset", asset.Name)
		return nil
	}

	return &types.CacheVersionDownloadDetails{
		Platform:            platform,
		Filename:            asset.Name,
		DownloadURL:         asset.DownloadURL,
		SHASumsURL:          "",
		SHASumsSignatureURL: "",
		SHASum:              shasum,
	}
}

func downloadShaSums(ctx context.Context, assets []github.ReleaseAsset) (map[string]string, error) {
	asset := github.FindAssetBySuffix(assets, "_SHA256SUMS")
	if asset == nil {
		return nil, fmt.Errorf("could not find shasums asset")
	}

	// download the asset
	sumsContent, assetErr := github.DownloadAssetContents(ctx, asset.DownloadURL)
	if assetErr != nil {
		return nil, fmt.Errorf("failed to download asset: %w", assetErr)
	}
	defer sumsContent.Close()

	sums := make(map[string]string)

	// read the contents of the shasums file
	scanner := bufio.NewScanner(sumsContent)
	for scanner.Scan() {
		// read the line
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 { //nolint:gomnd // we expect 2 parts
			continue
		}

		// the first part is the shasum, the second part is the filename
		// we want to return a map of filename -> shasum
		// so we can easily look up the shasum for a given filename
		// when we are processing the release assets
		if len(parts[1]) > 0 {
			sums[parts[1]] = parts[0]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read asset contents: %w", err)
	}
	return sums, nil
}

// GetVersion fetches and returns detailed information about a specific version of a provider hosted on GitHub.
// The returned information includes the download URL, the filename, SHA sums, and more details pertinent to the specific version, OS, and architecture.
//
// Parameters:
// - ctx: The context used to control cancellations and timeouts.
// - ghClient: The GitHub GraphQL client to interact with the GitHub GraphQL API.
// - namespace: The GitHub namespace (typically, the organization or user) under which the provider repository is hosted.
// - name: The name of the provider without the "terraform-provider-" prefix.
// - version: The specific version of the Terraform provider to fetch details for.
// - os: The operating system for which the provider binary is intended.
// - arch: The architecture for which the provider binary is intended.
//
// Returns a VersionDetails structure with detailed information about the specified version. If an error occurs during fetching or processing, it returns an error.

func GetVersion(ctx context.Context, ghClient *githubv4.Client, namespace string, name string, version string, os string, arch string) (versionDetails *types.VersionDetails, err error) {
	err = xray.Capture(ctx, "provider.versiondetails", func(tracedCtx context.Context) error {
		xray.AddAnnotation(tracedCtx, "namespace", namespace)
		xray.AddAnnotation(tracedCtx, "name", name)
		xray.AddAnnotation(tracedCtx, "version", version)
		xray.AddAnnotation(tracedCtx, "OS", os)
		xray.AddAnnotation(tracedCtx, "arch", arch)

		slog.Info("Fetching version")

		// TODO: Replace this with a GetRelease, iterating all the releases is not efficient at all!
		// Fetch the specific release for the given version.
		release, releaseErr := github.FindRelease(tracedCtx, ghClient, namespace, name, version)
		if releaseErr != nil {
			return fmt.Errorf("failed to find release: %w", releaseErr)
		}

		if release == nil {
			return newFetchError("failed to find release", ErrCodeReleaseNotFound, nil)
		}

		// Initialize the VersionDetails struct.
		versionDetails = &types.VersionDetails{
			OS:   os,
			Arch: arch,
		}

		// Find and parse the manifest from the release assets.
		manifest, manifestErr := findAndParseManifest(tracedCtx, release.ReleaseAssets.Nodes)
		if manifestErr != nil {
			return newFetchError("failed to find and parse manifest", ErrCodeManifestNotFound, manifestErr)
		}

		if manifest != nil {
			versionDetails.Protocols = manifest.Metadata.ProtocolVersions
		} else {
			versionDetails.Protocols = []string{"5.0"}
		}

		// Identify the appropriate asset for download based on OS and architecture.
		assetToDownload := github.FindAssetBySuffix(release.ReleaseAssets.Nodes, fmt.Sprintf("_%s_%s.zip", os, arch))
		if assetToDownload == nil {
			return newFetchError("failed to find asset to download", ErrCodeAssetNotFound, nil)
		}
		versionDetails.Filename = assetToDownload.Name
		versionDetails.DownloadURL = assetToDownload.DownloadURL

		// Locate the SHA256 checksums and its signature from the release assets.
		shaSumsAsset := github.FindAssetBySuffix(release.ReleaseAssets.Nodes, "_SHA256SUMS")
		shasumsSigAsset := github.FindAssetBySuffix(release.ReleaseAssets.Nodes, "_SHA256SUMS.sig")

		if shaSumsAsset == nil || shasumsSigAsset == nil {
			slog.Error("Could not find shasums or its signature asset")
			return newFetchError("failed to find shasums or its signature asset", ErrCodeSHASumsNotFound, nil)
		}

		versionDetails.SHASumsURL = shaSumsAsset.DownloadURL
		versionDetails.SHASumsSignatureURL = shasumsSigAsset.DownloadURL

		// Extract the SHA256 checksum for the asset to download.
		shaSum, shaSumErr := getShaSum(tracedCtx, shaSumsAsset.DownloadURL, versionDetails.Filename)
		if shaSumErr != nil {
			slog.Error("Could not get shasum", "error", shaSumErr)
			return newFetchError("failed to get shasum: %w", ErrCodeSHASumsNotFound, shaSumErr)
		}
		versionDetails.SHASum = shaSum

		publicKeys, keysErr := KeysForNamespace(namespace)
		if keysErr != nil {
			slog.Error("Could not get public keys", "error", keysErr)
			return newFetchError("failed to get public keys", ErrCodeCouldNotGetPublicKeys, keysErr)
		}

		versionDetails.SigningKeys = types.SigningKeys{
			GPGPublicKeys: publicKeys,
		}

		return nil
	})

	slog.Info("Successfully found version details")
	return versionDetails, err
}
