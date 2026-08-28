package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/keyservice"
)

func TestRecipientAddLetsSecondIdentityDecrypt(t *testing.T) {
	aliceDir := t.TempDir()
	bobDir := t.TempDir()
	aliceKey := filepath.Join(aliceDir, "identity")
	bobKey := filepath.Join(bobDir, "identity")

	aliceEnv := func(key string) string {
		switch key {
		case "SOPSDECK_STATE_DIR":
			return aliceDir
		case "SOPSDECK_KEYCHAIN_DIR":
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
		case "SOPSDECK_KEYCHAIN_DIR":
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

func TestRecipientRemoveRevokesAccessAndRotatesDataKey(t *testing.T) {
	aliceDir := t.TempDir()
	bobDir := t.TempDir()
	aliceKey := filepath.Join(aliceDir, "identity")
	bobKey := filepath.Join(bobDir, "identity")
	aliceEnv := identityEnv(aliceDir, aliceKey)
	bobEnv := identityEnv(bobDir, bobKey)

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

	envFile := filepath.Join(t.TempDir(), ".env.production")
	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "world", "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "add", bobPub, "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("recipient add exit %d stderr=%q", code, stderr.String())
	}

	before := fileDataKey(t, envFile, aliceKey)
	oldCopy := filepath.Join(t.TempDir(), ".env.production")
	raw, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldCopy, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "remove", bobPub, "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("recipient remove exit %d stderr=%q", code, stderr.String())
	}
	warn := stderr.String()
	if !strings.Contains(warn, "Git history") || !strings.Contains(warn, "still decrypt") {
		t.Fatalf("missing history warning: %q", warn)
	}

	after := fileDataKey(t, envFile, aliceKey)
	if bytes.Equal(before, after) {
		t.Fatal("data key was not rotated")
	}

	t.Setenv("SOPS_AGE_KEY_FILE", bobKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, bobEnv); code == 0 {
		t.Fatal("bob still has Access at HEAD")
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", oldCopy}, os.Stdin, &stdout, &stderr, bobEnv); code != 0 {
		t.Fatalf("bob get of kept copy exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "world" {
		t.Fatalf("bob kept copy stdout=%q", stdout.String())
	}

	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("alice get after remove exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "world" {
		t.Fatalf("alice get stdout=%q", stdout.String())
	}
}

func TestRecipientRemoveRefusesLastRecipient(t *testing.T) {
	aliceDir := t.TempDir()
	aliceKey := filepath.Join(aliceDir, "identity")
	aliceEnv := identityEnv(aliceDir, aliceKey)

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("alice identity exit %d stderr=%q", code, stderr.String())
	}
	alicePub := strings.TrimSpace(stdout.String())

	envFile := filepath.Join(t.TempDir(), ".env.production")
	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "world", "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	before := fileDataKey(t, envFile, aliceKey)

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "remove", alicePub, "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code == 0 {
		t.Fatal("removed the last Recipient")
	}
	if !strings.Contains(stderr.String(), "last Recipient") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	after := fileDataKey(t, envFile, aliceKey)
	if !bytes.Equal(before, after) {
		t.Fatal("data key changed after refused remove")
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("alice get exit %d stderr=%q", code, stderr.String())
	}
}

func TestRecipientRemoveUnknownIsNoOp(t *testing.T) {
	aliceDir := t.TempDir()
	bobDir := t.TempDir()
	aliceKey := filepath.Join(aliceDir, "identity")
	bobKey := filepath.Join(bobDir, "identity")
	aliceEnv := identityEnv(aliceDir, aliceKey)
	bobEnv := identityEnv(bobDir, bobKey)

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

	envFile := filepath.Join(t.TempDir(), ".env.production")
	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "world", "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	before := fileDataKey(t, envFile, aliceKey)

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "remove", bobPub, "-f", envFile}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("remove unknown exit %d stderr=%q", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "Access dropped") {
		t.Fatalf("warned on no-op: %q", stderr.String())
	}
	after := fileDataKey(t, envFile, aliceKey)
	if !bytes.Equal(before, after) {
		t.Fatal("data key changed for unknown Recipient")
	}
}

func identityEnv(stateDir, ageKey string) func(string) string {
	return func(key string) string {
		switch key {
		case "SOPSDECK_STATE_DIR":
			return stateDir
		case "SOPSDECK_KEYCHAIN_DIR":
			return stateDir
		case "SOPS_AGE_KEY_FILE":
			return ageKey
		default:
			return ""
		}
	}
}

func fileDataKey(t *testing.T, file, ageKey string) []byte {
	t.Helper()
	t.Setenv("SOPS_AGE_KEY_FILE", ageKey)
	store := common.StoreForFormat(fileFormat(file), config.NewStoresConfig())
	tree, err := common.LoadEncryptedFile(store, file)
	if err != nil {
		t.Fatal(err)
	}
	key, err := common.DecryptTree(common.DecryptTreeOpts{
		Tree:        tree,
		Cipher:      aes.NewCipher(),
		KeyServices: []keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	})
	if err != nil {
		t.Fatal(err)
	}
	return key
}
