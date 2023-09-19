package providers

import (
	"github.com/opentffoundation/registry/internal/platform"
	"time"
)

// Version represents an individual provider version.
// It provides details such as the version number, supported Terraform protocol versions, and platforms the provider is available for.
// This is made to match the registry v1 API response format for listing provider versions.
type Version struct {
	Version   string              `json:"version"`   // The version number of the provider.
	Protocols []string            `json:"protocols"` // The protocol versions the provider supports.
	Platforms []platform.Platform `json:"platforms"` // A list of platforms for which this provider version is available.
}

// ProviderVersionListingItem represents a single item in the DynamoDB table for provider versions.
// This is made to match the registry v1 API response format for listing provider versions.
type ProviderVersionListingItem struct {
	Provider    string    `json:"provider"`
	Versions    []Version `json:"versions"`
	LastUpdated time.Time `json:"last_updated"`
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
