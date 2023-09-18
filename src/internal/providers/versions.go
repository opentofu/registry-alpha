package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/shurcooL/githubv4"
)

// GetVersions fetches and returns a list of available versions of a given  provider hosted on GitHub.
// The returned versions also include information about supported platforms and the Terraform protocol versions they are compatible with.
//
// Parameters:
// - ctx: The context used to control cancellations and timeouts.
// - ghClient: The GitHub GraphQL client to interact with the GitHub GraphQL API.
// - namespace: The GitHub namespace (typically, the organization or user) under which the provider repository is hosted.
// - name: The name of the provider repository.
//
// Returns a slice of Version structures detailing each available version. If an error occurs during fetching or processing, it returns an error.
func GetVersions(ctx context.Context, ghClient *githubv4.Client, namespace string, name string) (versions []Version, err error) {
	type versionResult struct {
		Version Version
		Err     error
	}

	err = xray.Capture(ctx, "provider.versions", func(tracedCtx context.Context) error {
		xray.AddAnnotation(tracedCtx, "namespace", namespace)
		xray.AddAnnotation(tracedCtx, "name", name)

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
				result := versionResult{}

				assets := r.ReleaseAssets.Nodes
				platforms, platformsErr := getSupportedArchAndOS(assets)
				if platformsErr != nil {
					result.Err = fmt.Errorf("failed to get supported platforms: %w", platformsErr)
					versionCh <- result
					return
				}

				// if there are no platforms, we can't do anything with this release
				// so, we should just skip

				if len(platforms) == 0 {
					return
				}

				manifest, manifestErr := findAndParseManifest(tracedCtx, assets)
				if manifestErr != nil {
					result.Err = fmt.Errorf("failed to find and parse manifest: %w", manifestErr)
					versionCh <- result
					return
				}

				result.Version = Version{
					Version:   strings.TrimPrefix(r.TagName, "v"),
					Platforms: platforms,
				}

				if manifest != nil {
					result.Version.Protocols = manifest.Metadata.ProtocolVersions
				} else {
					result.Version.Protocols = []string{"5.0"}
				}

				versionCh <- result
			}(release)
		}

		// Close the channel when all goroutines are done.
		wg.Wait()
		close(versionCh)

		for vr := range versionCh {
			if vr.Err != nil {
				// we don't want to fail the entire operation if one version fails, just trace the error and continue
				xray.AddError(tracedCtx, fmt.Errorf("failed to process some releases: %w", vr.Err))
			}
			if vr.Version.Version != "" {
				versions = append(versions, vr.Version)
			}
		}

		return nil
	})

	return
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
// - OS: The operating system for which the provider binary is intended.
// - arch: The architecture for which the provider binary is intended.
//
// Returns a VersionDetails structure with detailed information about the specified version. If an error occurs during fetching or processing, it returns an error.

func GetVersion(ctx context.Context, ghClient *githubv4.Client, namespace string, name string, version string, OS string, arch string) (versionDetails *VersionDetails, err error) {
	err = xray.Capture(ctx, "provider.versiondetails", func(tracedCtx context.Context) error {
		xray.AddAnnotation(tracedCtx, "namespace", namespace)
		xray.AddAnnotation(tracedCtx, "name", name)
		xray.AddAnnotation(tracedCtx, "version", version)
		xray.AddAnnotation(tracedCtx, "OS", OS)
		xray.AddAnnotation(tracedCtx, "arch", arch)

		// Fetch the specific release for the given version.
		release, releaseErr := github.FindRelease(tracedCtx, ghClient, namespace, name, version)
		if releaseErr != nil {
			return fmt.Errorf("failed to find release: %w", releaseErr)
		}

		if release == nil {
			return fmt.Errorf("release not found")
		}

		// Initialize the VersionDetails struct.
		versionDetails = &VersionDetails{
			OS:   OS,
			Arch: arch,
		}

		// Find and parse the manifest from the release assets.
		manifest, manifestErr := findAndParseManifest(tracedCtx, release.ReleaseAssets.Nodes)
		if manifestErr != nil {
			return fmt.Errorf("failed to find and parse manifest: %w", manifestErr)
		}

		if manifest != nil {
			xray.AddAnnotation(tracedCtx, "protocols", fmt.Sprintf("%v", manifest.Metadata.ProtocolVersions))
			versionDetails.Protocols = manifest.Metadata.ProtocolVersions
		} else {
			xray.AddAnnotation(tracedCtx, "protocols", "default")
			versionDetails.Protocols = []string{"5.0"}
		}

		// Identify the appropriate asset for download based on OS and architecture.
		assetToDownload := github.FindAssetBySuffix(release.ReleaseAssets.Nodes, fmt.Sprintf("_%s_%s.zip", OS, arch))
		if assetToDownload == nil {
			return fmt.Errorf("could not find asset to download")
		}
		versionDetails.Filename = assetToDownload.Name
		versionDetails.DownloadURL = assetToDownload.DownloadURL

		// Locate the SHA256 checksums and its signature from the release assets.
		shaSumsAsset := github.FindAssetBySuffix(release.ReleaseAssets.Nodes, "_SHA256SUMS")
		shasumsSigAsset := github.FindAssetBySuffix(release.ReleaseAssets.Nodes, "_SHA256SUMS.sig")

		if shaSumsAsset == nil || shasumsSigAsset == nil {
			return fmt.Errorf("could not find shasums or its signature asset")
		}

		versionDetails.SHASumsURL = shaSumsAsset.DownloadURL
		versionDetails.SHASumsSignatureURL = shasumsSigAsset.DownloadURL

		// Extract the SHA256 checksum for the asset to download.
		shaSum, shaSumErr := getShaSum(tracedCtx, shaSumsAsset.DownloadURL, versionDetails.Filename)
		if shaSumErr != nil {
			return fmt.Errorf("failed to get shasum: %w", shaSumErr)
		}
		versionDetails.SHASum = shaSum

		publicKeys, keysErr := KeysForNamespace(namespace)
		if keysErr != nil {
			return fmt.Errorf("failed to get public keys: %w", keysErr)
		}

		versionDetails.SigningKeys = SigningKeys{
			GPGPublicKeys: publicKeys,
		}

		return nil
	})

	return
}
