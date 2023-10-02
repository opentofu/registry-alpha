package modules

import "time"

type Version struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"` // The direct URL to download the module.
}

// ToVersionListResponse converts a Version to a Version that's friendly with the list response.
// this mainly strips out the DownloadURL
func (v Version) ToVersionListResponse() Version {
	return Version{
		Version: v.Version,
	}
}

const allowedAge = (1 * time.Hour) - (5 * time.Minute) //nolint:gomnd // 55 minutes

// CacheItem is the item stored in the DynamoDB cache.
type CacheItem struct {
	Module      string    `json:"module"`   // The module name.
	Versions    []Version `json:"versions"` // The versions of the module.
	LastUpdated time.Time `json:"last_updated"`
}

func (i *CacheItem) IsStale() bool {
	return time.Since(i.LastUpdated) > allowedAge
}

type CacheVersionDetails struct {
	Version     string   `json:"version"`      // The version of the module.
	Protocols   []string `json:"protocols"`    // The protocol versions the module supports.
	Filename    string   `json:"filename"`     // The filename of the module binary.
	DownloadURL string   `json:"download_url"` // The direct URL to download the module binary.
}
