package providers

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
)

//go:embed keys/*
var keys embed.FS

// KeysForNamespace returns the GPG public keys for the given namespace.
func KeysForNamespace(namespace string) ([]GPGPublicKey, error) {
	k := keys

	dirName := filepath.Join("keys", namespace)

	entries, err := k.ReadDir(dirName)

	// This is fine, it just means that the namespace doesn't have any keys yet.
	if os.IsNotExist(err) {
		return []GPGPublicKey{}, nil
	}

	// This is not fine, it means that we failed to read the directory for some
	// other reason.
	if err != nil {
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	var publicKeys []GPGPublicKey

	for _, entry := range entries {
		path := filepath.Join(dirName, entry.Name())

		publicKey, err := buildKey(path)
		if err != nil {
			return nil, fmt.Errorf("could not build public key at %s: %w", path, err)
		}

		publicKeys = append(publicKeys, *publicKey)
	}

	return publicKeys, nil
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

func buildKey(path string) (*GPGPublicKey, error) {
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
		return nil, fmt.Errorf("could not build public key from ascii armor: %v", err)
	}

	return &GPGPublicKey{
		AsciiArmor: asciiArmor,
		KeyID:      strings.ToUpper(key.GetHexKeyID()),
	}, nil
}
