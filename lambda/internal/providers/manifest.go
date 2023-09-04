package providers

type Manifest struct {
	Version  float64          `json:"version"`
	Metadata ManifestMetadata `json:"metadata"`
}
type ManifestMetadata struct {
	ProtocolVersions []string `json:"protocol_versions"`
}
