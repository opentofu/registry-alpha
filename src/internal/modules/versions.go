package modules

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/shurcooL/githubv4"

	"github.com/opentofu/registry/internal/github"
)

// GetVersions fetches a list of versions for a GitHub repository identified by its namespace and name.
func GetVersions(ctx context.Context, ghClient *githubv4.Client, namespace string, name string) (versions []Version, err error) {
	err = xray.Capture(ctx, "module.versions", func(tracedCtx context.Context) error {
		xray.AddAnnotation(tracedCtx, "namespace", namespace)
		xray.AddAnnotation(tracedCtx, "name", name)

		releases, fetchErr := github.FetchReleases(tracedCtx, ghClient, namespace, name)
		if err != nil {
			return fmt.Errorf("failed to fetch releases: %w", fetchErr)
		}

		for _, release := range releases {
			versions = append(versions, Version{
				// Normalize the version string to remove the leading "v" if it exists.
				Version: strings.TrimPrefix(release.TagName, "v"),
			})
		}

		return nil
	})

	return versions, err
}
