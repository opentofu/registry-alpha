package modules

type Version struct {
	Version string `json:"version"`
}

// VersionDetails provides comprehensive details about a specific provider version.
// This includes the OS, architecture, download URLs, SHA sums, and the signing keys used for the version.
// This is made to match the registry v1 API response format for the download details.
type VersionDetails struct {
	Protocols   []string `json:"protocols"`    // The protocol versions the provider supports.
	Filename    string   `json:"filename"`     // The filename of the provider binary.
	DownloadURL string   `json:"download_url"` // The direct URL to download the provider binary.
}
