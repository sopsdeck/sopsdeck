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
	if err := os.WriteFile(file, []byte(plain), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"init", root, "--file", "eas.json", "--keys", "build.env.EXPO_NO_DOTENV"}, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("init exit %d: %s", code, stderr.String())
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `"EXPO_NO_DOTENV": "ENC[`) {
		t.Fatalf("selected path was not encrypted: %s", text)
	}
	if !strings.Contains(text, `"EXPO_PUBLIC_FOO": "FOO"`) || !strings.Contains(text, `"version": "20.5.1"`) {
		t.Fatalf("unselected values were changed: %s", text)
	}
	var getOut, getErr bytes.Buffer
	if code := cmdGet([]string{"-f", file, "--output", "json"}, &getOut, &getErr); code != 0 {
		t.Fatalf("get exit %d: %s", code, getErr.String())
	}
	var pairs map[string]string
	if err := json.Unmarshal(getOut.Bytes(), &pairs); err != nil {
		t.Fatal(err)
	}
	if pairs["build.env.EXPO_NO_DOTENV"] != "1" || pairs["build.env.EXPO_PUBLIC_FOO"] != "FOO" {
		t.Fatalf("pairs=%v", pairs)
	}
	var lockOut, lockErr bytes.Buffer
	if code := cmdUnlock([]string{"-f", file}, &lockOut, &lockErr); code != 0 {
		t.Fatalf("unlock exit %d: %s", code, lockErr.String())
	}
	plainRaw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plainRaw), "ENC[") {
		t.Fatalf("unlock left ciphertext on disk: %s", plainRaw)
	}
	if code := cmdSet([]string{"build.env.EXPO_NO_DOTENV", "2", "-f", file}, strings.NewReader(""), &lockOut, &lockErr, os.Getenv); code != 0 {
		t.Fatalf("set while unlocked exit %d: %s", code, lockErr.String())
	}
	if code := cmdLock([]string{"-f", file}, &lockOut, &lockErr, os.Getenv); code != 0 {
		t.Fatalf("lock exit %d: %s", code, lockErr.String())
	}
	lockedRaw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lockedRaw), `"EXPO_NO_DOTENV": "ENC[`) || !strings.Contains(string(lockedRaw), `"EXPO_PUBLIC_FOO": "FOO"`) {
		t.Fatalf("lock did not restore selective encryption: %s", lockedRaw)
	}
	getOut.Reset()
	getErr.Reset()
	if code := cmdGet([]string{"build.env.EXPO_NO_DOTENV", "-f", file}, &getOut, &getErr); code != 0 || strings.TrimSpace(getOut.String()) != "2" {
		t.Fatalf("updated value=%q err=%s", getOut.String(), getErr.String())
	}
}
