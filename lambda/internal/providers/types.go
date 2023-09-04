package providers

import "github.com/opentffoundation/registry/internal/platform"

// Version represents an individual provider version.
// It provides details such as the version number, supported Terraform protocol versions, and platforms the provider is available for.
// This is made to match the registry v1 API response format for listing provider versions.
type Version struct {
	Version   string              `json:"version"`   // The version number of the provider.
	Protocols []string            `json:"protocols"` // The protocol versions the provider supports.
	Platforms []platform.Platform `json:"platforms"` // A list of platforms for which this provider version is available.
}

// VersionDetails provides comprehensive details about a specific provider version.
// This includes the OS, architecture, download URLs, SHA sums, and the signing keys used for the version.
// This is made to match the registry v1 API response format for the download details.
type VersionDetails struct {
	Protocols           []string    `json:"protocols"`             // The protocol versions the provider supports.
	OS                  string      `json:"os"`                    // The operating system for which the provider is built.
	Arch                string      `json:"arch"`                  // The architecture for which the provider is built.
	Filename            string      `json:"filename"`              // The filename of the provider binary.
	DownloadURL         string      `json:"download_url"`          // The direct URL to download the provider binary.
	SHASumsURL          string      `json:"shasums_url"`           // The URL to the SHA checksums file.
	SHASumsSignatureURL string      `json:"shasums_signature_url"` // The URL to the GPG signature of the SHA checksums file.
	SHASum              string      `json:"shasum"`                // The SHA checksum of the provider binary.
	SigningKeys         SigningKeys `json:"signing_keys"`          // The signing keys used for this provider version.
}

// SigningKeys represents the GPG public keys used to sign a provider version.
type SigningKeys struct {
	GPGPublicKeys []GPGPublicKey `json:"gpg_public_keys"` // A list of GPG public keys.
}

// GPGPublicKey represents an individual GPG public key.
type GPGPublicKey struct {
	KeyID      string `json:"key_id"`      // The ID of the GPG key.
	AsciiArmor string `json:"ascii_armor"` // The ASCII armored representation of the GPG public key.
}

// GHRepository encapsulates GitHub repository details with a focus on its releases.
// This is structured to align with the expected response format from GitHub's GraphQL API.
type GHRepository struct {
	Repository struct {
		Releases struct {
			PageInfo struct {
				HasNextPage bool   // Indicates if there are more pages of releases.
				EndCursor   string // The cursor for pagination.
			}
			Nodes []GHRelease // A list of GitHub releases.
		} `graphql:"releases(first: $perPage, orderBy: {field: CREATED_AT, direction: DESC}, after: $endCursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

// GHRelease represents a release on GitHub.
// This provides details about the release, including its tag name, release assets, and its release status (draft, prerelease, etc.).
type GHRelease struct {
	ID            string // The ID of the release.
	TagName       string // The tag name associated with the release.
	ReleaseAssets struct {
		Nodes []ReleaseAsset // A list of assets for the release.
	} `graphql:"releaseAssets(first:100)"`
	IsDraft      bool // Indicates if the release is a draft.
	IsLatest     bool // Indicates if the release is the latest.
	IsPrerelease bool // Indicates if the release is a prerelease.
}

// ReleaseAsset represents a single asset within a GitHub release.
// This includes details such as the download URL and the name of the asset.
type ReleaseAsset struct {
	ID          string // The ID of the asset.
	DownloadURL string // The URL to download the asset.
	Name        string // The name of the asset.
}
