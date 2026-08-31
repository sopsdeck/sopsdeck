package cli

import (
	"testing"
)

func TestTreePathToleratesEmptySegments(t *testing.T) {
	// package-lock.json stores the root package under an empty-string key, so
	// "packages" -> "" -> "name" flattens to "packages..name". Treat it as one
	// opaque path element instead of erroring.
	got, err := treePath("packages..name", true)
	if err != nil {
		t.Fatalf("treePath: %v", err)
	}
	if len(got) != 1 || got[0] != "packages..name" {
		t.Fatalf("got=%v want [packages..name]", got)
	}
}

func TestTreePathStillRejectsMalformedBrackets(t *testing.T) {
	if _, err := treePath("items[0", true); err == nil {
		t.Fatal("want error for unterminated bracket")
	}
}
