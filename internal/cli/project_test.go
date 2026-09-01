package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectInitEncryptsSelectedFilesAndWritesManifest(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, ".env.production")
	if err := os.WriteFile(file, []byte("API_URL=https://example.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"init", root, "--file", ".env.production"}, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("init exit %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".sopsdeck.toml")); err != nil {
		t.Fatal(err)
	}

	state, err := inspectProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Initialized || len(state.Managed) != 1 || state.Managed[0].Rel != ".env.production" {
		t.Fatalf("state=%+v", state)
	}
	var got map[string]string
	var getOut, getErr bytes.Buffer
	if code := cmdGet([]string{"-f", file, "--output", "json"}, &getOut, &getErr); code != 0 {
		t.Fatalf("get exit %d: %s", code, getErr.String())
	}
	if err := json.Unmarshal(getOut.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["API_URL"] != "https://example.test" {
		t.Fatalf("got=%v", got)
	}
}

func TestProjectInitAcceptsQuotedMultilineDotenv(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, "apps", "admin", ".env")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, file, `FIREBASE_DOT_JSON='{
"hosting": {
"site": "loyalty-platform-admin-dev",
"source": ".",
"ignore": [
"firebase.json",
"**/.*",
"**/node_modules/**",
"**/*.stories.*"
],
"frameworksBackend": {
"region": "australia-southeast1",
"maxInstances": 10
}
}
}'
`)

	state, err := inspectProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Candidates) != 1 || strings.Join(state.Candidates[0].Keys, ",") != "FIREBASE_DOT_JSON" {
		t.Fatalf("candidates=%+v", state.Candidates)
	}

	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", "apps/admin/.env"}, &stdout, &stderr, os.Getenv), &stderr, "init")
	var getOut, getErr bytes.Buffer
	mustCLI(t, cmdGet([]string{"FIREBASE_DOT_JSON", "-f", file}, &getOut, &getErr), &getErr, "get")
	if !strings.Contains(getOut.String(), `"hosting"`) {
		t.Fatalf("multiline dotenv value=%q", getOut.String())
	}
}

func TestInspectProjectListsEncryptablePaths(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "eas.json")
	plain := `{"build":{"env":{"EXPO_NO_DOTENV":"1","EXPO_PUBLIC_FOO":"FOO"}},"submit":{"production":{"ios":{"appleTeamId":"1234"}}}}`
	if err := os.WriteFile(file, []byte(plain), 0o600); err != nil {
		t.Fatal(err)
	}

	state, err := inspectProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Candidates) != 1 {
		t.Fatalf("candidates=%+v", state.Candidates)
	}
	want := []string{
		"build.env.EXPO_NO_DOTENV",
		"build.env.EXPO_PUBLIC_FOO",
		"submit.production.ios.appleTeamId",
	}
	if strings.Join(state.Candidates[0].Keys, ",") != strings.Join(want, ",") {
		t.Fatalf("keys=%v want %v", state.Candidates[0].Keys, want)
	}
}

func TestProjectAddPreservesSelectedPathsWhenInitializing(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, "eas.json")
	if err := os.WriteFile(file, []byte(`{"build":{"env":{"SECRET":"value","PUBLIC":"safe"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"add", root, "--file", "eas.json", "--keys", "build.env.SECRET"}, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("add exit %d: %s", code, stderr.String())
	}
	manifest, err := loadManifest(filepath.Join(root, ".sopsdeck.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := manifest.ManagedFile[0].EncryptedKeys; strings.Join(got, ",") != "build.env.SECRET" {
		t.Fatalf("encrypted keys=%v", got)
	}
}

func TestProjectInitEncryptsOnlySelectedJSONLeaf(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, "eas.json")
	plain := `{"cli":{"version":"20.5.1"},"build":{"env":{"EXPO_NO_DOTENV":"1","EXPO_PUBLIC_FOO":"FOO"}}}`
	mustWriteFile(t, file, plain)

	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", "eas.json", "--keys", "build.env.EXPO_NO_DOTENV"}, &stdout, &stderr, os.Getenv), &stderr, "init")
	text := mustReadFile(t, file)
	mustContain(t, text, `"EXPO_NO_DOTENV": "ENC[`, "selected path was not encrypted")
	mustContain(t, text, `"EXPO_PUBLIC_FOO": "FOO"`, "unselected values were changed")
	mustContain(t, text, `"version": "20.5.1"`, "unselected values were changed")

	var getOut, getErr bytes.Buffer
	mustCLI(t, cmdGet([]string{"-f", file, "--output", "json"}, &getOut, &getErr), &getErr, "get")
	var pairs map[string]string
	if err := json.Unmarshal(getOut.Bytes(), &pairs); err != nil {
		t.Fatal(err)
	}
	if pairs["build.env.EXPO_NO_DOTENV"] != "1" || pairs["build.env.EXPO_PUBLIC_FOO"] != "FOO" {
		t.Fatalf("pairs=%v", pairs)
	}

	var lockOut, lockErr bytes.Buffer
	mustCLI(t, cmdUnlock([]string{"-f", file}, &lockOut, &lockErr), &lockErr, "unlock")
	if strings.Contains(mustReadFile(t, file), "ENC[") {
		t.Fatalf("unlock left ciphertext on disk: %s", mustReadFile(t, file))
	}
	mustCLI(t, cmdSet([]string{"build.env.EXPO_NO_DOTENV", "2", "-f", file}, strings.NewReader(""), &lockOut, &lockErr, os.Getenv), &lockErr, "set while unlocked")
	mustCLI(t, cmdLock([]string{"-f", file}, &lockOut, &lockErr, os.Getenv), &lockErr, "lock")
	locked := mustReadFile(t, file)
	mustContain(t, locked, `"EXPO_NO_DOTENV": "ENC[`, "lock did not restore selective encryption")
	mustContain(t, locked, `"EXPO_PUBLIC_FOO": "FOO"`, "lock did not restore selective encryption")
	getOut.Reset()
	getErr.Reset()
	if code := cmdGet([]string{"build.env.EXPO_NO_DOTENV", "-f", file}, &getOut, &getErr); code != 0 || strings.TrimSpace(getOut.String()) != "2" {
		t.Fatalf("updated value=%q err=%s", getOut.String(), getErr.String())
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func mustCLI(t *testing.T, code int, errBuf *bytes.Buffer, name string) {
	t.Helper()
	if code != 0 {
		t.Fatalf("%s exit %d: %s", name, code, errBuf.String())
	}
}

func mustContain(t *testing.T, text, want, msg string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("%s: %s", msg, text)
	}
}

func TestProjectInitJSONWithoutKeysLeavesLeavesPlaintext(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, "eas.json")
	plain := `{"cli":{"version":"20.5.1"},"build":{"env":{"SECRET":"value"}}}`
	mustWriteFile(t, file, plain)

	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", "eas.json"}, &stdout, &stderr, os.Getenv), &stderr, "init")
	text := mustReadFile(t, file)
	mustContain(t, text, `"version": "20.5.1"`, "unselected values were encrypted")
	mustContain(t, text, `"SECRET": "value"`, "unselected values were encrypted")
	if strings.Contains(text, `"SECRET": "ENC[`) {
		t.Fatalf("json without selected keys encrypted fields: %s", text)
	}
}

func TestProjectEncryptUpdatesSelectedJSONLeaves(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, "eas.json")
	plain := `{"cli":{"version":"20.5.1"},"build":{"env":{"SECRET":"value","PUBLIC":"safe"}}}`
	mustWriteFile(t, file, plain)

	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", "eas.json", "--keys", "build.env.PUBLIC"}, &stdout, &stderr, os.Getenv), &stderr, "init")
	mustCLI(t, cmdProject([]string{"encrypt", file, "--keys", "build.env.SECRET"}, &stdout, &stderr, os.Getenv), &stderr, "encrypt")
	text := mustReadFile(t, file)
	mustContain(t, text, `"SECRET": "ENC[`, "new path was not encrypted")
	mustContain(t, text, `"PUBLIC": "safe"`, "removed path stayed encrypted")
	mustContain(t, text, `"version": "20.5.1"`, "unselected values were changed")
	manifest, err := loadManifest(filepath.Join(root, ".sopsdeck.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(manifest.ManagedFile[0].EncryptedKeys, ","); got != "build.env.SECRET" {
		t.Fatalf("encrypted keys=%q", got)
	}
}

func TestProjectInitRecordsOwner(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, ".env.production")
	if err := os.WriteFile(file, []byte("API_URL=https://example.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"init", root, "--file", ".env.production"}, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("init exit %d: %s", code, stderr.String())
	}
	manifest, err := loadManifest(filepath.Join(root, ".sopsdeck.toml"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := ageRecipientFromEnv(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Owner) != 1 || manifest.Owner[0].Key != key {
		t.Fatalf("owners=%+v want %s", manifest.Owner, key)
	}
	cfg := projectConfigFor(root, os.Getenv)
	if !cfg.CanGrant || cfg.Name == "" {
		t.Fatalf("config=%+v", cfg)
	}
}
