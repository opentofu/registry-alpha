package providers

import (
	"github.com/opentofu/registry/internal/platform"
)

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
	ASCIIArmor string `json:"ascii_armor"` // The ASCII armored representation of the GPG public key.
}

// VersionCacheItem provides comprehensive details about a specific provider version.
// This includes the OS, architecture, download URLs, SHA sums, and the signing keys used for the version.
// This is made to store data in our cache for both provider version listing and provider download endpoints
type VersionCacheItem struct {
	Version         string                   `json:"version"` // The version number of the provider.
	DownloadDetails []VersionDownloadDetails `json:"download_details"`
	Protocols       []string                 `json:"protocols"` // The protocol versions the provider supports.
}

// VersionDownloadDetails provides comprehensive details about a specific provider version.
type VersionDownloadDetails struct {
	Platform            platform.Platform `json:"platform"`              // The platform
	Filename            string            `json:"filename"`              // The filename of the provider binary.
	DownloadURL         string            `json:"download_url"`          // The direct URL to download the provider binary.
	SHASumsURL          string            `json:"shasums_url"`           // The URL to the SHA checksums file.
	SHASumsSignatureURL string            `json:"shasums_signature_url"` // The URL to the GPG signature of the SHA checksums file.
	SHASum              string            `json:"shasum"`                // The SHA checksum of the provider binary.
}

// ToVersion converts a VersionCacheItem to a Version to be used in the provider version listing endpoint.
func (v *VersionCacheItem) ToVersion() Version {
	platforms := make([]platform.Platform, len(v.DownloadDetails))
	for i, d := range v.DownloadDetails {
		platforms[i] = d.Platform
	}

	return Version{
		Version:   v.Version,
		Protocols: v.Protocols,
		Platforms: platforms,
	}
}

// GetVersionDetails gets a VersionCacheItem to a VersionDetails for a specific OS and architecture.
// Note: This will be missing the SigningKeys field.
func (v *VersionCacheItem) GetVersionDetails(os, arch string) *VersionDetails {
	for _, d := range v.DownloadDetails {
		if d.Platform.OS == os && d.Platform.Arch == arch {
			return &VersionDetails{
				Protocols:           v.Protocols,
				OS:                  d.Platform.OS,
				Arch:                d.Platform.Arch,
				Filename:            d.Filename,
				DownloadURL:         d.DownloadURL,
				SHASumsURL:          d.SHASumsURL,
				SHASumsSignatureURL: d.SHASumsSignatureURL,
				SHASum:              d.SHASum,
				SigningKeys:         SigningKeys{},
			}
		}
	}

	return nil
}
