package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestErrorLogIncrementsCountForSameMessage(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", state)
	mustUnsetenv(t, "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("identity exit %d stderr=%q", code, stderr.String())
	}

	for range 2 {
		stdout.Reset()
		stderr.Reset()
		code := Main([]string{"get", "HELLO", "-f", testdata(t, "hello.env")}, os.Stdin, &stdout, &stderr, os.Getenv)
		if code == 0 {
			t.Fatal("expected get to fail without Access")
		}
	}

	records := readErrorLog(t, state)
	if len(records) != 1 {
		t.Fatalf("records=%d %+v, want 1", len(records), records)
	}
	if records[0].Count != 2 {
		t.Fatalf("count=%d, want 2", records[0].Count)
	}
	if records[0].Message == "" {
		t.Fatal("empty message")
	}
}

func TestErrorLogRecordsDriveStderr(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)

	var stderr strings.Builder
	stderr.WriteString("failed to decrypt\n")
	if err := cliErr(1, &stderr); err == nil {
		t.Fatal("expected error")
	}
	records := readErrorLog(t, state)
	if len(records) != 1 {
		t.Fatalf("records=%d", len(records))
	}
	if records[0].Message != "failed to decrypt" {
		t.Fatalf("message=%q", records[0].Message)
	}
	if records[0].Count != 1 {
		t.Fatalf("count=%d", records[0].Count)
	}
}

func TestErrorLogOmitsSecretValues(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)

	recordError(os.Getenv, "decrypt failed AGE-SECRET-KEY-1ABCDEF ENC[AES256_GCM,data:secret]")
	records := readErrorLog(t, state)
	if len(records) != 1 {
		t.Fatalf("records=%d", len(records))
	}
	got := records[0].Message
	if strings.Contains(got, "AGE-SECRET-KEY-1ABCDEF") {
		t.Fatalf("leaked private key: %q", got)
	}
	if strings.Contains(got, "data:secret") {
		t.Fatalf("leaked ciphertext: %q", got)
	}
}

func readErrorLog(t *testing.T, state string) []errorRecord {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(state, "errors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var records []errorRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		t.Fatal(err)
	}
	return records
}
