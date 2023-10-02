package modules

import "fmt"

func GetDownloadURL(namespace, repoName, version string) string {
	return fmt.Sprintf("git::https://github.com/%s/%s?ref=v%s", namespace, repoName, version)
}
