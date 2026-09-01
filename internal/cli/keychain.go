package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	keychainService  = "sopsdeck"
	keychainAccount  = "age"
	keychainFileName = "identity"
)

func putIdentity(getenv func(string) string, body string) error {
	if dir := getenv("SOPSDECK_KEYCHAIN_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, keychainFileName), []byte(body), 0o600)
	}
	return keyring.Set(keychainService, keychainAccount, body)
}

func getIdentity(getenv func(string) string) (string, error) {
	if dir := getenv("SOPSDECK_KEYCHAIN_DIR"); dir != "" {
		data, err := os.ReadFile(filepath.Join(dir, keychainFileName))
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return keyring.Get(keychainService, keychainAccount)
}

func deleteIdentity(getenv func(string) string) error {
	if dir := getenv("SOPSDECK_KEYCHAIN_DIR"); dir != "" {
		err := os.Remove(filepath.Join(dir, keychainFileName))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := keyring.Delete(keychainService, keychainAccount); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}
