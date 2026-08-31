package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

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
