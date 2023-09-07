package modules

import "fmt"

// GetRepoName returns the repo name for a module
// The repo name should match the format `terraform-<system>-<name>`
func GetRepoName(system, name string) string {
	return fmt.Sprintf("terraform-%s-%s", system, name)
}
