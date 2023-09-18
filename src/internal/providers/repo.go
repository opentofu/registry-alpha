package providers

import "fmt"

// GetRepoName returns the repo name for a provider
// The repo name should match the format `terraform-provider-<name>`
func GetRepoName(name string) string {
	return fmt.Sprintf("terraform-provider-%s", name)
}
