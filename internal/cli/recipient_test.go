package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecipientAddLetsSecondIdentityDecrypt(t *testing.T) {
	aliceDir := t.TempDir()
	bobDir := t.TempDir()
	aliceKey := filepath.Join(aliceDir, "age.txt")
	bobKey := filepath.Join(bobDir, "age.txt")

	aliceEnv := func(key string) string {
		switch key {
		case "SOPSDECK_STATE_DIR":
			return aliceDir
		case "SOPS_AGE_KEY_FILE":
			return aliceKey
		default:
			return ""
		}
	}
	bobEnv := func(key string) string {
		switch key {
		case "SOPSDECK_STATE_DIR":
			return bobDir
		case "SOPS_AGE_KEY_FILE":
			return bobKey
		default:
			return ""
		}
	}

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("alice identity exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, bobEnv); code != 0 {
		t.Fatalf("bob identity exit %d stderr=%q", code, stderr.String())
	}
	bobPub := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(bobPub, "age1") {
		t.Fatalf("bob pub=%q", bobPub)
	}

	envFile := filepath.Join(t.TempDir(), ".env.production")
	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "world", "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	t.Setenv("SOPS_AGE_KEY_FILE", bobKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, bobEnv); code == 0 {
		t.Fatal("bob decrypted before recipient add")
	}

	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "add", bobPub, "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("recipient add exit %d stderr=%q", code, stderr.String())
	}

	t.Setenv("SOPS_AGE_KEY_FILE", bobKey)

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, bobEnv); code != 0 {
		t.Fatalf("bob get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "world" {
		t.Fatalf("bob get stdout=%q", stdout.String())
	}

	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "add", bobPub, "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("second add exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("alice get after second add exit %d stderr=%q", code, stderr.String())
	}
}
