package providers

import (
	"context"
	"fmt"
	"github.com/shurcooL/githubv4"
	"strings"
)

// GetVersions fetches and returns a list of available versions of a given  provider hosted on GitHub.
// The returned versions also include information about supported platforms and the Terraform protocol versions they are compatible with.
//
// Parameters:
// - ctx: The context used to control cancellations and timeouts.
// - ghClient: The GitHub GraphQL client to interact with the GitHub GraphQL API.
// - namespace: The GitHub namespace (typically, the organization or user) under which the provider repository is hosted.
// - name: The name of the provider without the "terraform-provider-" prefix.
//
// Returns a slice of Version structures detailing each available version. If an error occurs during fetching or processing, it returns an error.
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

		// Extract supported platforms from the release assets.
		platforms, err := getSupportedArchAndOS(assets)
		if err != nil {
			return nil, err
		}

		// Find and parse the manifest associated with the assets.
		manifest, err := findAndParseManifest(ctx, assets)
		if err != nil {
			return nil, err
		}

		// Normalize the version name.
		versionName := release.TagName
		if strings.HasPrefix(versionName, "v") {
			versionName = versionName[1:]
		}

		// Construct the Version struct.
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

func GetVersion(ctx context.Context, ghClient *githubv4.Client, namespace string, name string, version string, OS string, arch string) (*VersionDetails, error) {
	// Construct the repo name.
	repoName := fmt.Sprintf("terraform-provider-%s", name)

	// Fetch the specific release for the given version.
	release, err := findRelease(ctx, ghClient, namespace, repoName, version)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, fmt.Errorf("release not found")
	}

	// Initialize the VersionDetails struct.
	result := &VersionDetails{
		OS:   OS,
		Arch: arch,
	}

	// Find and parse the manifest from the release assets.
	manifest, err := findAndParseManifest(ctx, release.ReleaseAssets.Nodes)
	if err != nil {
		return nil, err
	}
	if manifest != nil {
		result.Protocols = manifest.Metadata.ProtocolVersions
	} else {
		result.Protocols = []string{"5.0"}
	}

	// Identify the appropriate asset for download based on OS and architecture.
	assetToDownload := findAssetBySuffix(release.ReleaseAssets.Nodes, fmt.Sprintf("_%s_%s.zip", OS, arch))
	if assetToDownload == nil {
		return nil, fmt.Errorf("could not find asset to download")
	}
	result.Filename = assetToDownload.Name
	result.DownloadURL = assetToDownload.DownloadURL

	// Locate the SHA256 checksums and its signature from the release assets.
	shaSumsAsset := findAssetBySuffix(release.ReleaseAssets.Nodes, "_SHA256SUMS")
	shasumsSigAsset := findAssetBySuffix(release.ReleaseAssets.Nodes, "_SHA256SUMS.sig")
	if shaSumsAsset == nil || shasumsSigAsset == nil {
		return nil, fmt.Errorf("could not find shasums or its signature asset")
	}
	result.SHASumsURL = shaSumsAsset.DownloadURL
	result.SHASumsSignatureURL = shasumsSigAsset.DownloadURL

	// Extract the SHA256 checksum for the asset to download.
	shaSum, err := getShaSum(ctx, shaSumsAsset.DownloadURL, result.Filename)
	if err != nil {
		return nil, err
	}
	result.SHASum = shaSum

	// TODO: Handle GPG keys.
	result.SigningKeys = SigningKeys{
		GPGPublicKeys: []GPGPublicKey{},
	}

	return result, nil
}
