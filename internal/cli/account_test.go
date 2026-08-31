package cli

import (
	"os"
	"os/exec"
	"testing"
)

func TestAccountForPathUsesGitIdentityAndAgeIdentity(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.name", "Mina Kim"},
		{"-C", dir, "config", "user.email", "mina@example.com"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}

	got := accountForPath(dir, os.Getenv)
	if got.Name != "Mina Kim" || got.Email != "mina@example.com" || !got.HasIdentity {
		t.Fatalf("account=%+v", got)
	}
}

func TestConfigureGitIdentityDoesNotOverrideExistingIdentity(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.name", "Mina Kim"},
		{"-C", dir, "config", "user.email", "mina@example.com"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}

	if err := configureGitIdentity(dir, "Someone Else", "other@example.com"); err == nil {
		t.Fatal("expected existing Git identity to remain unchanged")
	}
	name, _ := exec.Command("git", "-C", dir, "config", "--get", "user.name").Output()
	if string(name) != "Mina Kim\n" {
		t.Fatalf("Git identity changed to %q", name)
	}
}
