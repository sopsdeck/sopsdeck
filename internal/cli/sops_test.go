package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSOPSPassthrough(t *testing.T) {
	if _, err := exec.LookPath("sops"); err != nil {
		t.Skip("sops is not installed")
	}
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	file := filepath.Join(t.TempDir(), "config.json")
	var stdout, stderr bytes.Buffer
	key, err := ageRecipientFromEnv(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"encrypt", "--age", key, "--filename-override", file, "--input-type", "json", "--output-type", "json"}
	if code := Main(args, strings.NewReader(`{"TOKEN":"passthrough-fixture"}`), &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("encrypt exit %d: %s", code, &stderr)
	}
	mustWriteFile(t, file, stdout.String())
	for _, args := range [][]string{
		{"decrypt", file},
		{"--decrypt", file},
		{"sops", "decrypt", file},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := Main(args, strings.NewReader(""), &stdout, &stderr, os.Getenv); code != 0 {
			t.Fatalf("%v exit %d: %s", args, code, &stderr)
		}
		if !strings.Contains(stdout.String(), "passthrough-fixture") {
			t.Fatalf("%v did not decrypt fixture", args)
		}
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", file, `["TOKEN"]`, `"updated-fixture"`}, strings.NewReader(""), &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("native set exit %d: %s", code, &stderr)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"decrypt", file}, strings.NewReader(""), &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("decrypt updated file exit %d: %s", code, &stderr)
	}
	if !strings.Contains(stdout.String(), "updated-fixture") {
		t.Fatal("native set did not update the encrypted file")
	}
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", t.TempDir())
	if err := putIdentity(os.Getenv, mustReadFile(t, testdata(t, "age.txt"))); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"} {
		t.Setenv(key, "")
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"decrypt", file}, strings.NewReader(""), &stdout, &stderr, os.Getenv); code != 0 || !strings.Contains(stdout.String(), "updated-fixture") {
		t.Fatalf("keychain decrypt exit %d: %s", code, &stderr)
	}
}

func TestSOPSPreservesArgumentsStreamsAndExitCode(t *testing.T) {
	fakeTestExecutable(t, "sops", "sops")
	stateDir := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", stateDir)
	var stdout, stderr bytes.Buffer
	args := []string{"decrypt", "--extract", `["a b"][0]`, "file with spaces.json"}
	if code := Main(args, strings.NewReader("stdin-fixture\n"), &stdout, &stderr, os.Getenv); code != 23 {
		t.Fatalf("upstream exit %d, want 23: %s", code, &stderr)
	}
	if got := strings.Split(strings.TrimSpace(stdout.String()), "\n"); !reflect.DeepEqual(got, append(args, "stdin-fixture")) {
		t.Fatalf("arguments/stdin changed: %q", got)
	}
	if stderr.String() != "upstream error\n" {
		t.Fatalf("stderr changed: %q", &stderr)
	}
	if _, err := os.Stat(filepath.Join(stateDir, errorLogName)); !os.IsNotExist(err) {
		t.Fatal("native stderr was persisted in the error log")
	}
}
