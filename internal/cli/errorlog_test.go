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
	stderr.WriteString("failed to decrypt fake-adapter-secret\n")
	err := cliErr(1, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "fake-adapter-secret") || err.Error() != "command failed (exit 1)" {
		t.Fatalf("adapter error was not safely summarized (length=%d)", len(err.Error()))
	}
	records := readErrorLog(t, state)
	if len(records) != 1 {
		t.Fatalf("records=%d", len(records))
	}
	if records[0].Message != "command failed (exit 1)" {
		t.Fatalf("message=%q", records[0].Message)
	}
	if records[0].Count != 1 {
		t.Fatalf("count=%d", records[0].Count)
	}
}

func TestErrorLogOmitsRawDiagnostics(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)

	var stderr strings.Builder
	stderr.WriteString("parse error AGE-SECRET-KEY-1ABCDEF ENC[AES256_GCM,data:secret] fake-parser-secret")
	err := cliErr(1, &stderr)
	if err == nil || err.Error() != "command failed (exit 1)" {
		t.Fatal("parser diagnostic was not summarized")
	}
	records := readErrorLog(t, state)
	if len(records) != 1 {
		t.Fatalf("records=%d", len(records))
	}
	if records[0].Message != "command failed (exit 1)" {
		t.Fatalf("raw diagnostic was persisted (length=%d)", len(records[0].Message))
	}
}

func TestFailedNoRedactChildStderrIsNotPersisted(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	secret := "fake-child-secret-7f3"
	t.Setenv("SOPSDECK_TEST_FAKE_SECRET", secret)
	child := fakeTestExecutable(t, "error-child", "failed-stderr")

	var stdout, stderr strings.Builder
	code := Main([]string{"run", "--no-redact", "--", child}, strings.NewReader(""), &stdout, &stderr, os.Getenv)
	if code != 23 {
		t.Fatalf("run exit=%d want 23", code)
	}
	if !strings.Contains(stderr.String(), "fake child diagnostic: "+secret) || !strings.Contains(stderr.String(), "child diagnostic stream end") {
		t.Fatalf("child stderr was not streamed to the caller (length=%d)", stderr.Len())
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatal("child stderr secret was copied to stdout")
	}

	records := readErrorLog(t, state)
	if len(records) != 1 || records[0].Message != "command failed (exit 23)" {
		messageLength := 0
		if len(records) > 0 {
			messageLength = len(records[0].Message)
		}
		t.Fatalf("diagnostic log was not safely summarized (records=%d message length=%d)", len(records), messageLength)
	}
	if strings.Contains(records[0].Message, secret) || strings.Contains(records[0].Message, "AGE-SECRET-KEY") || strings.Contains(records[0].Message, "ENC[") {
		t.Fatal("raw child diagnostic appeared in the error log")
	}
	info, err := os.Stat(filepath.Join(state, errorLogName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 256 {
		t.Fatalf("diagnostic log grew with child output: %d bytes", info.Size())
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
