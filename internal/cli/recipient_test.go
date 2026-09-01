package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
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

func TestRecipientLabelsRoundTrip(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, ".env")
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"set", "HELLO", "world", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	manifest := "[[managed_file]]\npath = \".env\"\n"
	if err := os.WriteFile(filepath.Join(root, ".sopsdeck.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	robot, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "add", robot.Recipient().String(), "-f", file, "--name", "Deploy bot", "--kind", "robot"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("add exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "list", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("list exit %d stderr=%q", code, stderr.String())
	}
	var recipients []accessRecipient
	if err := json.Unmarshal(stdout.Bytes(), &recipients); err != nil {
		t.Fatal(err)
	}
	if len(recipients) != 2 || recipients[1].Name != "Deploy bot" || recipients[1].Kind != "robot" {
		t.Fatalf("recipients=%+v", recipients)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "remove", robot.Recipient().String(), "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("remove exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	if code := Main([]string{"recipient", "list", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("list after remove exit %d stderr=%q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), robot.Recipient().String()) {
		t.Fatalf("removed recipient still listed: %s", stdout.String())
	}
}

func TestRecipientListShowsInitOwnerAndAssignedIdentity(t *testing.T) {
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
	alicePub := strings.TrimSpace(stdout.String())
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, bobEnv); code != 0 {
		t.Fatalf("bob identity exit %d stderr=%q", code, stderr.String())
	}
	bobPub := strings.TrimSpace(stdout.String())

	root := t.TempDir()
	file := filepath.Join(root, ".env")
	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "world", "-f", file}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	manifest := "[[managed_file]]\npath = \".env\"\n\n[[owner]]\nkey = \"" + alicePub + "\"\nname = \"Alice Example\"\nemail = \"alice@example.com\"\n"
	if err := os.WriteFile(filepath.Join(root, ".sopsdeck.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "add", bobPub, "-f", file, "--name", "Bob Builder <bob@example.com>"}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("add exit %d stderr=%q", code, stderr.String())
	}

	aliceList := listFileAccess(t, file, aliceEnv)
	bobRow := accessByKey(aliceList, bobPub)
	if bobRow.Name != "Bob Builder" || bobRow.Email != "bob@example.com" || bobRow.Self {
		t.Fatalf("alice view of bob=%+v", bobRow)
	}

	t.Setenv("SOPS_AGE_KEY_FILE", bobKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", file}, os.Stdin, &stdout, &stderr, bobEnv); code != 0 {
		t.Fatalf("bob get exit %d stderr=%q", code, stderr.String())
	}

	bobList := listFileAccess(t, file, bobEnv)
	aliceRow := accessByKey(bobList, alicePub)
	if aliceRow.Name != "Alice Example" || aliceRow.Email != "alice@example.com" || aliceRow.Self {
		t.Fatalf("bob view of alice=%+v (want Alice’s init identity)", aliceRow)
	}
	if accessByKey(bobList, bobPub).Name != "Bob Builder" {
		t.Fatalf("bob should still see the name Alice assigned: %+v", bobList)
	}
}

func TestRecipientAddRefusesWhenNotOwner(t *testing.T) {
	aliceDir := t.TempDir()
	aliceKey := filepath.Join(aliceDir, "identity")
	aliceEnv := identityEnv(aliceDir, aliceKey)
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("alice identity exit %d stderr=%q", code, stderr.String())
	}
	root := t.TempDir()
	file := filepath.Join(root, ".env")
	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "world", "-f", file}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	extra, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "add", extra.Recipient().String(), "-f", file}, os.Stdin, &stdout, &stderr, aliceEnv); code != 0 {
		t.Fatalf("recipient add exit %d stderr=%q", code, stderr.String())
	}
	outsider, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	manifest := "[[managed_file]]\npath = \".env\"\n\n[[owner]]\nkey = \"" + outsider.Recipient().String() + "\"\nname = \"Lead\"\n"
	if err := os.WriteFile(filepath.Join(root, ".sopsdeck.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "add", extra.Recipient().String(), "-f", file, "--name", "Bob"}, os.Stdin, &stdout, &stderr, aliceEnv); code == 0 {
		t.Fatal("non-owner recipient add succeeded")
	}
	if !strings.Contains(stderr.String(), "only a Project owner") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "remove", extra.Recipient().String(), "-f", file}, os.Stdin, &stdout, &stderr, aliceEnv); code == 0 {
		t.Fatal("non-owner recipient remove succeeded")
	}
	if !strings.Contains(stderr.String(), "only a Project owner") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRecipientAddAllowedWhenOwner(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, ".env")
	if err := os.WriteFile(file, []byte("HELLO=world\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"init", root, "--file", ".env"}, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("init exit %d: %s", code, stderr.String())
	}
	extra, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"recipient", "add", extra.Recipient().String(), "-f", file, "--name", "Bot"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("owner add exit %d stderr=%q", code, stderr.String())
	}
}

func listFileAccess(t *testing.T, file string, getenv func(string) string) []accessRecipient {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"recipient", "list", "-f", file}, os.Stdin, &stdout, &stderr, getenv); code != 0 {
		t.Fatalf("recipient list exit %d stderr=%q", code, stderr.String())
	}
	var list []accessRecipient
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	return list
}

func accessByKey(list []accessRecipient, key string) accessRecipient {
	for _, item := range list {
		if strings.EqualFold(item.Key, key) {
			return item
		}
	}
	return accessRecipient{}
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
