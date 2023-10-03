package modules

import "time"

type Version struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url,omitempty"` // The direct URL to download the module.
}

// toVersionListResponse converts a Version to a Version that's friendly with the list response.
// this mainly strips out the DownloadURL
func (v Version) toVersionListResponse() Version {
	return Version{
		Version: v.Version,
	}
}

const allowedAge = (1 * time.Hour) - (5 * time.Minute) //nolint:gomnd // 55 minutes

// VersionList is a list of versions.
type VersionList []Version

// ToVersionListResponse converts a VersionList to a VersionList that's friendly with the list response.
func (v VersionList) ToVersionListResponse() VersionList {
	var versions VersionList
	for _, version := range v {
		versions = append(versions, version.toVersionListResponse())
	}
	return versions
}

func (v VersionList) FindVersion(version string) (*Version, bool) {
	for _, ver := range v {
		if ver.Version == version {
			return &ver, true
		}
	}
	return nil, false
}

// CacheItem is the item stored in the DynamoDB cache.
type CacheItem struct {
	Module      string      `json:"module"`   // The module name.
	Versions    VersionList `json:"versions"` // The versions of the module.
	LastUpdated time.Time   `json:"last_updated"`
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
