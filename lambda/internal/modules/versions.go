package modules

import (
	"context"
	"github.com/opentffoundation/registry/internal/github"
	"github.com/shurcooL/githubv4"

	"strings"
)

// TODO: doc
func GetVersions(ctx context.Context, ghClient *githubv4.Client, namespace string, name string) ([]Version, error) {
	releases, err := github.FetchReleases(ctx, ghClient, namespace, name)
	if err != nil {
		return nil, err
	}

	var versions []Version
	for _, release := range releases {
		// Normalize the version name.
		versionName := release.TagName
		if strings.HasPrefix(versionName, "v") {
			versionName = versionName[1:]
		}

		// Construct the Version struct.
		version := Version{
			Version: versionName,
		}

		versions = append(versions, version)
	}
	return versions, nil
}
