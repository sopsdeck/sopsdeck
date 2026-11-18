package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactWriterRedactsAcrossFlushBoundary(t *testing.T) {
	secret := "sk_live_abcdef123456"
	var out bytes.Buffer
	w := &redactWriter{redactor: newRedactor(map[string]string{"TOKEN": secret}), out: &out}
	input := strings.Repeat("x", 600) + secret + strings.Repeat("y", 500)
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), secret) || !strings.Contains(out.String(), "[sopsdeck:TOKEN]") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRedactWriterRedactsEveryChunkSplit(t *testing.T) {
	secret := "sk_live_abcdef123456"
	input := "token=" + secret + " end"
	for split := 0; split <= len(input); split++ {
		var out bytes.Buffer
		w := &redactWriter{redactor: newRedactor(map[string]string{"TOKEN": secret}), out: &out}
		for _, part := range [][]byte{[]byte(input[:split]), []byte(input[split:])} {
			if _, err := w.Write(part); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Flush(); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), secret) || !strings.Contains(out.String(), "[sopsdeck:TOKEN]") {
			t.Fatalf("split %d leaked or missed the secret: %q", split, out.String())
		}
	}
}

func TestRedactorHandlesShortEqualAndOverlappingValues(t *testing.T) {
	r := newRedactor(map[string]string{
		"A":        "same",
		"B":        "same",
		"SHORT":    "abc",
		"LONG":     "abcdef",
		"ORIGINAL": "fake_secret",
		"MARKER":   "ORIGINAL",
	})
	got := r.redact("same abcdef fake_secret")
	want := "[sopsdeck:A] [sopsdeck:LONG] [sopsdeck:ORIGINAL]"
	if got != want {
		t.Fatalf("redact=%q want %q", got, want)
	}
}

func TestRedactWriterHandlesLongUnicodeAndMultilineValues(t *testing.T) {
	values := map[string]string{
		"LONG":      strings.Repeat("z", 600),
		"UNICODE":   "秘密🔑",
		"MULTILINE": "line one\nline two",
	}
	input := "start " + values["LONG"] + " | " + values["UNICODE"] + " | " + values["MULTILINE"] + " end"
	want := "start [sopsdeck:LONG] | [sopsdeck:UNICODE] | [sopsdeck:MULTILINE] end"
	for split := 0; split <= len(input); split++ {
		var out bytes.Buffer
		w := &redactWriter{redactor: newRedactor(values), out: &out}
		if _, err := w.Write([]byte(input[:split])); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(input[split:])); err != nil {
			t.Fatal(err)
		}
		if err := w.Flush(); err != nil {
			t.Fatal(err)
		}
		if got := out.String(); got != want {
			t.Fatalf("split %d output mismatch: %q", split, got)
		}
	}
}

func TestRedactWriterFlushesOrdinaryOutputPromptly(t *testing.T) {
	longSecret := strings.Repeat("secret-prefix-", 1000)
	var out bytes.Buffer
	w := &redactWriter{redactor: newRedactor(map[string]string{"LONG": longSecret}), out: &out}
	if _, err := w.Write([]byte("ready\n")); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "ready\n" {
		t.Fatalf("ordinary output was held back: %q", got)
	}
}

func TestRedactionPairsUsePublicPathsAndInheritedValues(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	if err := os.WriteFile(filepath.Join(dir, ".sopsdeck.toml"), []byte("[[managed_file]]\npath = \".env\"\npublic_keys = [\"PUBLIC\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("API_TOKEN", "fake_parent_token")
	t.Setenv("EMPTY_FILE_TOKEN", "fake_inherited_empty_file_token")
	values := redactionPairs(file, map[string]string{
		"API_TOKEN":        "fake_file_token",
		"EMPTY_FILE_TOKEN": "",
		"PUBLIC":           "visible",
		"PUBLIC_ENDPOINT":  "still_secret",
		"TOKEN_PLAIN":      "also_secret",
	}, true, nil, false, "")
	r := newRedactorValues(values)
	got := r.redact("fake_file_token fake_parent_token fake_inherited_empty_file_token visible still_secret also_secret")
	for _, secret := range []string{"fake_file_token", "fake_parent_token", "fake_inherited_empty_file_token", "still_secret", "also_secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("candidate %q leaked: %q", secret, got)
		}
	}
	if !strings.Contains(got, "visible") || strings.Contains(got, "[sopsdeck:PUBLIC]") {
		t.Fatalf("explicit public value was redacted: %q", got)
	}
}

func TestRedactionPairsUseStructuredEncryptedPaths(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	if err := os.WriteFile(filepath.Join(dir, ".sopsdeck.toml"), []byte("[[managed_file]]\npath = \"config.json\"\nencrypted_keys = [\"token\", \"public\", \"label\"]\npublic_keys = [\"auth.public\", \"metadata\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	values := redactionPairs(file, map[string]string{
		"auth.token":     "fake_token",
		"auth.public":    "visible",
		"app.url":        "plaintext_value",
		"metadata.label": "public_group_value",
	}, false, []string{"token", "public", "label"}, false, encryptedKeyRegex([]string{"token", "public", "label"}))
	got := newRedactorValues(values).redact("fake_token visible plaintext_value public_group_value")
	if !strings.Contains(got, "[sopsdeck:auth.token]") || strings.Contains(got, "fake_token") {
		t.Fatalf("encrypted path was not redacted: %q", got)
	}
	if !strings.Contains(got, "visible") || !strings.Contains(got, "plaintext_value") || !strings.Contains(got, "public_group_value") {
		t.Fatalf("unencrypted or public values were redacted: %q", got)
	}
}

type failedWriter struct {
	buf bytes.Buffer
}

func (w *failedWriter) Write(p []byte) (int, error) {
	_, _ = w.buf.Write(p)
	return 0, errors.New("fixture writer failed")
}

func TestRedactWriterDoesNotBypassErrors(t *testing.T) {
	secret := "fake_secret_value"
	out := &failedWriter{}
	w := &redactWriter{redactor: newRedactor(map[string]string{"TOKEN": secret}), out: out}
	if _, err := w.Write([]byte(secret)); err == nil {
		t.Fatal("expected output error")
	}
	if strings.Contains(out.buf.String(), secret) {
		t.Fatalf("writer received unredacted bytes: %q", out.buf.String())
	}
	if err := w.Flush(); err == nil {
		t.Fatal("flush hid the output error")
	}
}
