package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"
)

func cmdFileStatus(args []string, stdout, stderr io.Writer) int {
	file, usage, code := parseFileFlag(args, "status")
	if usage != "" {
		fmt.Fprintln(stderr, usage)
		return code
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(stderr, "status: %v\n", err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(map[string]bool{"locked": isEncryptedBytes(data)}); err != nil {
		fmt.Fprintf(stderr, "status: %v\n", err)
		return 1
	}
	return 0
}

func cmdUnlock(args []string, stdout, stderr io.Writer) int {
	file, usage, code := parseFileFlag(args, "unlock")
	if usage != "" {
		fmt.Fprintln(stderr, usage)
		return code
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(stderr, "unlock: %v\n", err)
		return 1
	}
	if !isEncryptedBytes(data) {
		fmt.Fprintln(stderr, "unlock: file is already unlocked")
		return 1
	}
	plain, err := decrypt.File(file, formatName(fileFormat(file)))
	if err != nil {
		fmt.Fprintf(stderr, "unlock: %v\n", err)
		return 1
	}
	// Plaintext has no SOPS metadata; keep its public recipients for relocking.
	tree, err := common.LoadEncryptedFile(common.StoreForFormat(fileFormat(file), config.NewStoresConfig()), file)
	if err != nil {
		fmt.Fprintf(stderr, "unlock: %v\n", err)
		return 1
	}
	var recipients []string
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			if _, ok := key.(*sopsage.MasterKey); !ok || len(tree.Metadata.KeyGroups) != 1 {
				fmt.Fprintln(stderr, "unlock: keep this file encrypted to preserve its non-Age or grouped Access")
				return 1
			}
			recipients = append(recipients, key.ToString())
		}
	}
	mapping, _, manifestPath := mappingFor(file)
	if mapping.Path != "" {
		m, err := loadManifest(manifestPath)
		if err != nil {
			fmt.Fprintf(stderr, "unlock: %v\n", err)
			return 1
		}
		for i := range m.ManagedFile {
			if filepath.ToSlash(m.ManagedFile[i].Path) == filepath.ToSlash(mapping.Path) {
				m.ManagedFile[i].Recipients = recipients
			}
		}
		if err := writeManifest(manifestPath, m); err != nil {
			fmt.Fprintf(stderr, "unlock: %v\n", err)
			return 1
		}
	} else if len(recipients) > 1 {
		fmt.Fprintln(stderr, "unlock: initialize a Project first to preserve all Recipients when relocking")
		return 1
	}
	if err := writeAtomic(file, plain); err != nil {
		fmt.Fprintf(stderr, "unlock: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "file unlocked")
	return 0
}

func cmdLock(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	file, usage, code := parseFileFlag(args, "lock")
	if usage != "" {
		fmt.Fprintln(stderr, usage)
		return code
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(stderr, "lock: %v\n", err)
		return 1
	}
	if isEncryptedBytes(data) {
		fmt.Fprintln(stderr, "lock: file is already locked")
		return 1
	}
	mapping, _, _ := mappingFor(file)
	if err := encryptPlainFile(file, data, getenv, mapping.EncryptedKeys); err != nil {
		fmt.Fprintf(stderr, "lock: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "file locked")
	return 0
}
