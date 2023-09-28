package providers

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/opentofu/registry/internal/providers/types"
)

//go:embed keys/*
var keys embed.FS

// KeysForNamespace returns the GPG public keys for the given namespace.
func KeysForNamespace(namespace string) ([]types.GPGPublicKey, error) {
	dirName := filepath.Join("keys", namespace)

	entries, err := keys.ReadDir(dirName)

	if err != nil {
		// This is fine, it just means that the namespace doesn't have any keys yet.
		if os.IsNotExist(err) {
			return []types.GPGPublicKey{}, nil
		}

		// This is not fine, it means that we failed to read the directory for some
		// other reason.
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	publicKeys := make([]types.GPGPublicKey, 0, len(entries))
	var buildErrors []error

	for _, entry := range entries {
		path := filepath.Join(dirName, entry.Name())

		publicKey, err := buildKey(path)
		if err != nil {
			buildErrors = append(buildErrors, fmt.Errorf("could not build public key at %s: %w", path, err))
		} else {
			publicKeys = append(publicKeys, *publicKey)
		}
	}

	return publicKeys, errors.Join(buildErrors...)
}

// NamespacesWithKeys returns the namespaces that have keys.
func NamespacesWithKeys() ([]string, error) {
	entries, err := keys.ReadDir("keys")
	if err != nil {
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	var namespaces []string

	for _, entry := range entries {
		namespaces = append(namespaces, entry.Name())
	}

	return namespaces, nil
}

func buildKey(path string) (*types.GPGPublicKey, error) {
	file, err := keys.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read key file: %w", err)
	}

	asciiArmor := string(data)

	key, err := crypto.NewKeyFromArmored(asciiArmor)
	if err != nil {
		return nil, fmt.Errorf("could not build public key from ascii armor: %w", err)
	}

	return &types.GPGPublicKey{
		ASCIIArmor: asciiArmor,
		KeyID:      strings.ToUpper(key.GetHexKeyID()),
	}, nil
}
