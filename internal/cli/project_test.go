package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectStaleManifestKeepsValidFilesUsable(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, ".sopsdeck.toml")
	mustWriteFile(t, manifest, "[[managed_file]]\npath = '.en'\n[[managed_file]]\npath = '.env'\n")
	mustWriteFile(t, filepath.Join(root, ".env"), "TOKEN=fixture\n")
	before := mustReadFile(t, manifest)
	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"files", root}, &stdout, &stderr, os.Getenv), &stderr, "inspect stale manifest")
	var state struct {
		Initialized bool          `json:"initialized"`
		Managed     []projectFile `json:"managed"`
		Warnings    []struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if !state.Initialized || len(state.Managed) != 1 || state.Managed[0].Rel != ".env" || len(state.Warnings) != 1 || state.Warnings[0].Path != ".en" || !strings.Contains(state.Warnings[0].Message, "missing") {
		t.Fatalf("stale manifest state: %+v", state)
	}
	if mustReadFile(t, manifest) != before {
		t.Fatal("inspection rewrote the manifest")
	}
	stdout.Reset()
	stderr.Reset()
	mustCLI(t, cmdProject([]string{"remove", root, "--file", ".en"}, &stdout, &stderr, os.Getenv), &stderr, "remove missing entry")
	if mustReadFile(t, filepath.Join(root, ".env")) != "TOKEN=fixture\n" {
		t.Fatal("recovery changed a valid file")
	}
}

func TestProjectAddAcceptsAnyFileName(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, ".en")
	mustWriteFile(t, file, "TOKEN=fixture\n")
	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"add", root, "--file", ".en"}, &stdout, &stderr, os.Getenv), &stderr, "add arbitrary file name")
	if !isEncryptedBytes([]byte(mustReadFile(t, file))) {
		t.Fatal("arbitrary file was not encrypted")
	}
	manifest, err := loadManifest(filepath.Join(root, ".sopsdeck.toml"))
	if err != nil || len(manifest.ManagedFile) != 1 || manifest.ManagedFile[0].Format != "dotenv" {
		t.Fatalf("manifest=%+v err=%v", manifest, err)
	}
	stdout.Reset()
	stderr.Reset()
	mustCLI(t, cmdGet([]string{"TOKEN", "-f", file}, &stdout, &stderr, os.Getenv), &stderr, "read arbitrary file name")
	if strings.TrimSpace(stdout.String()) != "fixture" {
		t.Fatalf("value=%q", stdout.String())
	}
}

func TestProjectInitAcceptsAnyFileName(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, ".en")
	mustWriteFile(t, file, "TOKEN=fixture\n")
	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", ".en"}, &stdout, &stderr, os.Getenv), &stderr, "init arbitrary file name")
	manifest, err := loadManifest(filepath.Join(root, ".sopsdeck.toml"))
	if err != nil || len(manifest.ManagedFile) != 1 || manifest.ManagedFile[0].Format != "dotenv" {
		t.Fatalf("manifest=%+v err=%v", manifest, err)
	}
	stdout.Reset()
	stderr.Reset()
	mustCLI(t, cmdGet([]string{"TOKEN", "-f", file}, &stdout, &stderr, os.Getenv), &stderr, "read initialized arbitrary file name")
	if strings.TrimSpace(stdout.String()) != "fixture" {
		t.Fatalf("value=%q", stdout.String())
	}
}

func TestProjectManifestIssuesAreIsolated(t *testing.T) {
	for _, rel := range []string{"../outside.env", ".", "linked.env", "nested/.env", "directory.env"} {
		t.Run(rel, func(t *testing.T) {
			root := t.TempDir()
			outside := filepath.Join(t.TempDir(), "outside.env")
			mustWriteFile(t, outside, "OUTSIDE=untouched\n")
			if err := os.Symlink(outside, filepath.Join(root, "linked.env")); err != nil {
				t.Fatal(err)
			}
			for _, dir := range []string{"nested/.git", "directory.env"} {
				if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			mustWriteFile(t, filepath.Join(root, "nested", ".env"), "NESTED=untouched\n")
			mustWriteFile(t, filepath.Join(root, ".env"), "TOKEN=fixture\n")
			mustWriteFile(t, filepath.Join(root, ".sopsdeck.toml"), "[[managed_file]]\npath = '.env'\n[[managed_file]]\npath = '"+rel+"'\n")
			state, err := inspectProject(root)
			if err != nil || len(state.Managed) != 1 || len(state.Warnings) != 1 {
				t.Fatalf("issue should not block safe files: %+v, %v", state, err)
			}
			var stdout, stderr bytes.Buffer
			mustCLI(t, cmdProject([]string{"remove", root, "--file", rel}, &stdout, &stderr, os.Getenv), &stderr, "remove invalid entry")
			if mustReadFile(t, outside) != "OUTSIDE=untouched\n" || mustReadFile(t, filepath.Join(root, "nested", ".env")) != "NESTED=untouched\n" {
				t.Fatal("recovery touched a file outside the Project")
			}
		})
	}
}

func TestProjectListsLegacyManagedNames(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".sopsdeck.toml"), "[[managed_file]]\npath = '.en'\n")
	mustWriteFile(t, filepath.Join(root, ".en"), "TOKEN=fixture\n")
	state, err := inspectProject(root)
	if err != nil || len(state.Managed) != 1 || state.Managed[0].Rel != ".en" {
		t.Fatalf("legacy entry silently hidden: %+v, %v", state, err)
	}
}

func TestProjectAddWithBrokenManifestDoesNotModifyFiles(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	manifest := filepath.Join(root, ".sopsdeck.toml")
	mustWriteFile(t, manifest, "[[managed_file\n")
	file := filepath.Join(root, ".env")
	mustWriteFile(t, file, "TOKEN=untouched\n")
	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"add", root, "--file", ".env"}, &stdout, &stderr, os.Getenv); code == 0 {
		t.Fatal("accepted broken manifest")
	}
	if mustReadFile(t, file) != "TOKEN=untouched\n" || mustReadFile(t, manifest) != "[[managed_file\n" {
		t.Fatal("failed add changed files")
	}
}

func TestProjectBoundariesAndCanonicalPaths(t *testing.T) {
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	app := filepath.Join(root, "apps", "web")
	nested := filepath.Join(root, "vendor-repo")
	for _, dir := range []string{app, filepath.Join(nested, ".git")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	mustWriteFile(t, filepath.Join(root, ".sopsdeck.toml"), "[[managed_file]]\npath = \"apps/web/.env\"\n")
	mustWriteFile(t, filepath.Join(app, ".env"), "HELLO=world\n")
	mustWriteFile(t, filepath.Join(nested, ".env"), "OTHER=private\n")
	state, err := inspectProject(app)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Initialized || len(state.Managed) != 1 || len(state.Candidates) != 0 {
		t.Fatalf("opening a monorepo subfolder: %+v", state)
	}
	if owner, _ := findManifest(filepath.Join(nested, ".env")); owner != "" {
		t.Fatalf("nested repository inherited parent Project %s", owner)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".", "../escape.env", "linked/new.env", "vendor-repo/new.env"} {
		var stdout, stderr bytes.Buffer
		if code := cmdProject([]string{"add", root, "--file", rel}, &stdout, &stderr, os.Getenv); code == 0 || !strings.Contains(stderr.String(), "Project") {
			t.Fatalf("unsafe add %s: exit %d: %s", rel, code, stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"init", app}, &stdout, &stderr, os.Getenv); code == 0 {
		t.Fatal("initialized an overlapping Project")
	}
	alias := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	mustCLI(t, cmdProject([]string{"files", alias}, &stdout, &stderr, os.Getenv), &stderr, "inspect alias")
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["path"] != root {
		t.Fatalf("canonical project path=%v want %s", got["path"], root)
	}
}

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
	if code := cmdGet([]string{"-f", file, "--output", "json"}, &getOut, &getErr, os.Getenv); code != 0 {
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
	mustCLI(t, cmdGet([]string{"FIREBASE_DOT_JSON", "-f", file}, &getOut, &getErr, os.Getenv), &getErr, "get")
	if !strings.Contains(getOut.String(), `"hosting"`) {
		t.Fatalf("multiline dotenv value=%q", getOut.String())
	}
}

func TestInspectProjectListsEncryptablePaths(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "app.config.json")
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
	file := filepath.Join(root, "app.config.json")
	if err := os.WriteFile(file, []byte(`{"build":{"env":{"SECRET":"value","PUBLIC":"safe"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"add", root, "--file", "app.config.json", "--keys", "build.env.SECRET"}, &stdout, &stderr, os.Getenv); code != 0 {
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

func TestProjectRemoveStopsManagingFileWithoutDeletingIt(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, ".env.production")
	mustWriteFile(t, file, "API_URL=https://example.test\n")

	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", ".env.production"}, &stdout, &stderr, os.Getenv), &stderr, "init")
	stdout.Reset()
	stderr.Reset()
	mustCLI(t, cmdProject([]string{"remove", root, "--file", ".env.production"}, &stdout, &stderr, os.Getenv), &stderr, "remove")

	state, err := inspectProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Managed) != 0 || len(state.Candidates) != 1 {
		t.Fatalf("state=%+v", state)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("removed file: %v", err)
	}
}

func TestProjectInitEncryptsOnlySelectedJSONLeaf(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, "app.config.json")
	plain := `{"cli":{"version":"20.5.1"},"build":{"env":{"EXPO_NO_DOTENV":"1","EXPO_PUBLIC_FOO":"FOO"}}}`
	mustWriteFile(t, file, plain)

	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", "app.config.json", "--keys", "build.env.EXPO_NO_DOTENV"}, &stdout, &stderr, os.Getenv), &stderr, "init")
	text := mustReadFile(t, file)
	mustContain(t, text, `"EXPO_NO_DOTENV": "ENC[`, "selected path was not encrypted")
	mustContain(t, text, `"EXPO_PUBLIC_FOO": "FOO"`, "unselected values were changed")
	mustContain(t, text, `"version": "20.5.1"`, "unselected values were changed")

	var getOut, getErr bytes.Buffer
	mustCLI(t, cmdGet([]string{"-f", file, "--output", "json"}, &getOut, &getErr, os.Getenv), &getErr, "get")
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
	if code := cmdGet([]string{"build.env.EXPO_NO_DOTENV", "-f", file}, &getOut, &getErr, os.Getenv); code != 0 || strings.TrimSpace(getOut.String()) != "2" {
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

func TestProjectAddAcceptsEASJSON(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "eas.json"), []byte(`{"build":{"env":{"SECRET":"value"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"add", root, "--file", "eas.json", "--keys", "build.env.SECRET"}, &stdout, &stderr, os.Getenv), &stderr, "add eas.json")
	manifest, err := loadManifest(filepath.Join(root, ".sopsdeck.toml"))
	if err != nil || len(manifest.ManagedFile) != 1 || manifest.ManagedFile[0].Path != "eas.json" {
		t.Fatalf("manifest=%+v err=%v", manifest, err)
	}
	if manifest.ManagedFile[0].Format != "json" {
		t.Fatalf("eas.json format=%q want json", manifest.ManagedFile[0].Format)
	}
}

func TestProjectInitJSONWithoutKeysLeavesLeavesPlaintext(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, "app.config.json")
	plain := `{"cli":{"version":"20.5.1"},"build":{"env":{"SECRET":"value"}}}`
	mustWriteFile(t, file, plain)

	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", "app.config.json"}, &stdout, &stderr, os.Getenv), &stderr, "init")
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
	file := filepath.Join(root, "app.config.json")
	plain := `{"cli":{"version":"20.5.1"},"build":{"env":{"SECRET":"value","PUBLIC":"safe"}}}`
	mustWriteFile(t, file, plain)

	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", "app.config.json", "--keys", "build.env.PUBLIC"}, &stdout, &stderr, os.Getenv), &stderr, "init")
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

func TestProjectInitRecordsRecipientIdentity(t *testing.T) {
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
	if len(manifest.Recipient) != 1 || manifest.Recipient[0].Key != key {
		t.Fatalf("recipients=%+v want %s", manifest.Recipient, key)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".sopsdeck.toml"))
	if err != nil || strings.Contains(string(raw), "[[owner]]") {
		t.Fatalf("new manifest contains owner roles: %s (%v)", raw, err)
	}
}
