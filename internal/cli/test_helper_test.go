package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fakeTestExecutable(t *testing.T, name, helper string) string {
	t.Helper()

	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(binDir, name)
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOPSDECK_TEST_HELPER", helper)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return path
}

func TestMain(m *testing.M) {
	switch os.Getenv("SOPSDECK_TEST_HELPER") {
	case "gh-args":
		if err := os.WriteFile(os.Getenv("SOPSDECK_GH_ARGS"), []byte(strings.Join(os.Args[1:], "\n")+"\n"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	case "gh-token":
		if len(os.Args) == 3 && os.Args[1] == "auth" && os.Args[2] == "token" {
			fmt.Fprintln(os.Stdout, "gho_from_gh")
			os.Exit(0)
		}
		os.Exit(1)
	case "sops":
		for _, arg := range os.Args[1:] {
			fmt.Fprintln(os.Stdout, arg)
		}
		_, _ = io.Copy(os.Stdout, os.Stdin)
		fmt.Fprintln(os.Stderr, "upstream error")
		os.Exit(23)
	}
	os.Exit(m.Run())
}
