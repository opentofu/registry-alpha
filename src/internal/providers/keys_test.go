package providers_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opentffoundation/registry/internal/providers"
)

func TestKeysForNamespace(t *testing.T) {
	t.Run("for an existing organization", func(t *testing.T) {
		keys, err := providers.KeysForNamespace("spacelift-io")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(keys) != 1 {
			t.Fatalf("expected 1 key, got %d", len(keys))
		}

		if keys[0].KeyID != "E302FB5AA29D88F7" {
			t.Fatalf("expected key ID to be E302FB5AA29D88F7, got %s", keys[0].KeyID)
		}

		if !strings.HasPrefix(keys[0].ASCIIArmor, "-----BEGIN PGP PUBLIC KEY BLOCK-----") {
			t.Fatalf("expected key to have ascii armor, got empty string")
		}
	})

	t.Run("for a non-existing organization", func(t *testing.T) {
		keys, err := providers.KeysForNamespace("baconsoft")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(keys) != 0 {
			t.Fatalf("expected no keys, got %v", keys)
		}

		if keys == nil {
			t.Fatalf("expected keys to be an empty slice, got nil")
		}
	})
}

func TestAllNamespaces(t *testing.T) {
	namespaces, err := providers.NamespacesWithKeys()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, namespace := range namespaces {
		t.Run(fmt.Sprintf("keys for %s", namespace), func(t *testing.T) {
			if _, err := providers.KeysForNamespace(namespace); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
