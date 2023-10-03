// Package warnings defines the warnings associated with the provider
package warnings

import "context"

// ProviderWarnings return the list of warnings for a given provider identified by its namespace and type
//
// Example: registry.terraform.io/hashicorp/terraform
//
// warn := ProviderWarnings("hashicorp", "terraform")
// fmt.Println(warn)
// >> [This provider is archived and no longer needed. The terraform_remote_state data source is built into the latest OpenTofu release.]
func ProviderWarnings(providerNamespace, providerType string) []string {
	switch providerNamespace { //nolint:gocritic // Switch is more appropriate than 'if' for the use case
	case "hashicorp":
		switch providerType { //nolint:gocritic // Switch is more appropriate than 'if' for the use case
		case "terraform":
			return []string{`This provider is archived and no longer needed. The terraform_remote_state data source is built into the latest OpenTofu release.`}
		}
	}

	return nil
}

var contextKey = struct{}{} //nolint:gochecknoglobals // This is a commonly used pattern for context binding

// NewContext adds warnings to the parent context.
func NewContext(ctx context.Context, warnings []string) context.Context {
	if ctx == nil {
		panic("ctx must be provided")
	}
	if len(warnings) == 0 {
		return ctx
	}
	return context.WithValue(ctx, contextKey, warnings)
}

// FromContext extracts the list of warnings from the context.
func FromContext(ctx context.Context) []string {
	if v := ctx.Value(contextKey); v != nil {
		return v.([]string)
	}
	return nil
}
