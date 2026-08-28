package changelog

import (
	"strings"
	"testing"
)

const sample = `# Changelog

## Unreleased

- wip item

## 1.0.0

- First public release.
- Grant Access from the inspector.

## 0.1.0

- Dev preview.
`

func TestNotesForTagReturnsSection(t *testing.T) {
	got, err := Notes(sample, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "First public release") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "wip item") || strings.Contains(got, "Dev preview") {
		t.Fatalf("leaked other sections: %q", got)
	}
}

func TestNotesForTagFailsWhenMissing(t *testing.T) {
	_, err := Notes(sample, "2.0.0")
	if err == nil {
		t.Fatal("expected missing section")
	}
	if !strings.Contains(err.Error(), "2.0.0") {
		t.Fatalf("err=%v", err)
	}
}

func TestNotesForTagFailsWhenOnlyUnreleased(t *testing.T) {
	_, err := Notes("## Unreleased\n\n- still cooking\n", "1.0.0")
	if err == nil {
		t.Fatal("expected missing 1.0.0")
	}
}

func TestBulletsFromUnreleased(t *testing.T) {
	got := Bullets(sample, "Unreleased")
	if len(got) != 1 || got[0] != "wip item" {
		t.Fatalf("got %#v", got)
	}
}

func TestBulletsUnderAddedHeading(t *testing.T) {
	md := "## Unreleased\n\n### Added\n\n- Nested folders\n\n### Fixed\n\n- Folder picker hang\n"
	got := Bullets(md, "Unreleased")
	if len(got) != 2 || got[0] != "Nested folders" || got[1] != "Folder picker hang" {
		t.Fatalf("got %#v", got)
	}
}
