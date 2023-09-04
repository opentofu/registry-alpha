package platform

import "regexp"

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

var platformPattern = regexp.MustCompile(`.*_(?P<Os>[a-zA-Z0-9]+)_(?P<Arch>[a-zA-Z0-9]+)`)

func ExtractPlatformFromArtifact(releaseArtifact string) *Platform {
	matches := platformPattern.FindStringSubmatch(releaseArtifact)

	if matches == nil {
		return nil
	}

	platform := Platform{
		OS:   matches[platformPattern.SubexpIndex("Os")],
		Arch: matches[platformPattern.SubexpIndex("Arch")],
	}

	return &platform
}
