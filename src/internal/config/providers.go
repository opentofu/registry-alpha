package config

// EffectiveProviderNamespace will map namespaces for providers in situations
// where the author (owner of the namespace) does not release artifacts as
// GitHub Releases.
func (c Config) EffectiveProviderNamespace(namespace string) string {
	if redirect, ok := c.ProviderRedirects[namespace]; ok {
		return redirect
	}

	return namespace
}