package providercache

import (
	"time"

	"github.com/opentofu/registry/internal/providers"
)

type VersionListingItem struct {
	Provider    string                       `dynamodbav:"provider"`
	Versions    []providers.VersionCacheItem `dynamodbav:"versions"`
	LastUpdated time.Time                    `dynamodbav:"last_updated"`
}

func (i *VersionListingItem) IsStale() bool {
	return time.Since(i.LastUpdated) > allowedAge
}

func (i *VersionListingItem) ToVersionListing() []providers.Version {
	results := make([]providers.Version, len(i.Versions))
	for i, version := range i.Versions {
		results[i] = version.ToVersion()
	}
	return results
}

func (i *VersionListingItem) GetVersionDetails(version string, os string, arch string) *providers.VersionDetails {
	for _, v := range i.Versions {
		if v.Version == version {
			return v.GetVersionDetails(os, arch)
		}
	}
	return nil
}
