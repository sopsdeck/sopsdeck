package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/cmd/sops/formats"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/keyservice"
	"github.com/getsops/sops/v3/version"
)

// encryptDotenvFixture encrypts plaintext at path for the age identity in
// ageFile, mirroring the SOPS tree setup used by the identity tests.
func encryptDotenvFixture(t *testing.T, path, ageFile, plain string) {
	t.Helper()
	store := common.StoreForFormat(formats.Dotenv, config.NewStoresConfig())
	branches, err := store.LoadPlainFile([]byte(plain))
	if err != nil {
		t.Fatal(err)
	}
	pub, err := os.ReadFile(ageFile)
	if err != nil {
		t.Fatal(err)
	}
	// age.txt is a keypair file (comments + secret); extract the public key
	// line for the recipient.
	var recipient string
	for _, line := range strings.Split(string(pub), "\n") {
		if strings.HasPrefix(line, "# public key: ") {
			recipient = strings.TrimPrefix(line, "# public key: ")
			break
		}
	}
	if recipient == "" {
		t.Fatal("no public key line in age fixture")
	}
	mk, err := sopsage.MasterKeyFromRecipient(recipient)
	if err != nil {
		t.Fatal(err)
	}
	tree := sops.Tree{
		FilePath: path,
		Metadata: sops.Metadata{
			Version:           version.Version,
			UnencryptedSuffix: sops.DefaultUnencryptedSuffix,
			KeyGroups:         []sops.KeyGroup{{mk}},
		},
		Branches: branches,
	}
	dataKey, errs := tree.GenerateDataKeyWithKeyServices([]keyservice.KeyServiceClient{keyservice.NewLocalClient()})
	if len(errs) > 0 {
		t.Fatal(errs[0])
	}
	if err := common.EncryptTree(common.EncryptTreeOpts{DataKey: dataKey, Tree: &tree, Cipher: aes.NewCipher()}); err != nil {
		t.Fatal(err)
	}
	out, err := store.EmitEncryptedFile(tree)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestGetInterpolatesFileLocalRef(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	encryptDotenvFixture(t, file, age, "HOST=db.internal\nPORT=5432\nURL=postgres://${HOST}:${PORT}/app\n")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"get", "URL", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "postgres://db.internal:5432/app" {
		t.Fatalf("URL=%q want interpolated value", got)
	}
}

func TestGetFallsBackToRealEnvThenEmpty(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	t.Setenv("DB_USER", "realuser")
	mustUnsetenv(t, "DB_MISSING")

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	// HOST exists in the file, so it wins over any env value; DB_USER exists
	// only in the env; DB_MISSING nowhere, expanding empty.
	encryptDotenvFixture(t, file, age, "HOST=filehost\nUSER=${DB_USER}\nURL=postgres://${DB_USER}@${HOST}\nGONE=${DB_MISSING}x\n")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"get", "URL", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "postgres://realuser@filehost" {
		t.Fatalf("URL=%q want file-first then env resolution", got)
	}

	stdout.Reset()
	if code := Main([]string{"get", "GONE", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "x" {
		t.Fatalf("GONE=%q want undefined ref expanded empty", got)
	}
}

func TestGetEscapedDollarStaysLiteral(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	encryptDotenvFixture(t, file, age, "PRICE=\\$9.99\n")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"get", "PRICE", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "$9.99" {
		t.Fatalf("PRICE=%q want literal dollar with backslash dropped", got)
	}
}

func TestGetSingleQuotedValueIsLiteral(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	t.Setenv("NOT_EXPANDED", "envval")

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	encryptDotenvFixture(t, file, age, "LITERAL='${NOT_EXPANDED}'\n")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"get", "LITERAL", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "${NOT_EXPANDED}" {
		t.Fatalf("LITERAL=%q want unexpanded single-quoted value", got)
	}
}

func TestGetCommandSubstitutionResolves(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	encryptDotenvFixture(t, file, age, "VERSION=$(echo 1.2.3)\nAPI=https://api.test/v$VERSION\n")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"get", "VERSION", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "1.2.3" {
		t.Fatalf("VERSION=%q want command substitution output", got)
	}

	stdout.Reset()
	if code := Main([]string{"get", "API", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "https://api.test/v1.2.3" {
		t.Fatalf("API=%q want substituted value composed into later key", got)
	}
}

func TestGetCommandSubstitutionRunsInFileKeyEnv(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	// The substitution shell must see the file's own keys as its environment.
	encryptDotenvFixture(t, file, age, "REGION=eu-west-1\nENDPOINT=$(echo host is ${REGION})\n")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"get", "ENDPOINT", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "host is eu-west-1" {
		t.Fatalf("ENDPOINT=%q want file env visible to substitution", got)
	}
}

func TestGetEscapedCommandSubstitutionStaysLiteral(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	encryptDotenvFixture(t, file, age, "NOTE=\\$(not a command)\n")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"get", "NOTE", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "$(not a command)" {
		t.Fatalf("NOTE=%q want literal escaped command", got)
	}
}

func TestGetCyclesCollapseInFileOrder(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	// dotenvx has no cycle detection: resolution is single-pass in file
	// order. A references B before B exists in the running map, so it
	// expands empty; B then sees A's already-collapsed value; C composes
	// both. The same expression one layer deep does resolve (ONE).
	encryptDotenvFixture(t, file, age, "A=${B}tail\nB=${A}more\nC=${A}-${B}\nSELF=${SELF}x\nONE=${SELF}\n")

	var stdout, stderr bytes.Buffer
	for key, want := range map[string]string{"A": "tail", "B": "tailmore", "C": "tail-tailmore", "SELF": "x"} {
		stdout.Reset()
		if code := Main([]string{"get", key, "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
			t.Fatalf("get %s exit %d stderr=%q", key, code, stderr.String())
		}
		if got := strings.TrimSpace(stdout.String()); got != want {
			t.Fatalf("%s=%q want %q", key, got, want)
		}
	}

	// ONE=${SELF} resolves to SELF's final collapsed value because only one
	// expansion layer is applied per key: "x", not "xx".
	stdout.Reset()
	if code := Main([]string{"get", "ONE", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get ONE exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "x" {
		t.Fatalf("ONE=%q want one-layer resolution of SELF", got)
	}
}

func TestDotenvComputedValuesAreSinglePass(t *testing.T) {
	got, err := dotenvPairs([]byte("EARLY=${LATER}tail\nLATER=ready\nSELF=${SELF}x\n"+
		"NESTED=$(printf '%s' \"$(printf 'nested ) value')\")\n"+
		"QUOTED=$(printf '%s' 'a ) b')\nGENERATED=$(printf '%s' '$(false)')\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"EARLY":     "tail",
		"LATER":     "ready",
		"SELF":      "x",
		"NESTED":    "nested ) value",
		"QUOTED":    "a ) b",
		"GENERATED": "$(false)",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s=%q want %q", key, got[key], value)
		}
	}
}

func TestDotenvCommandFailureDoesNotReturnOutput(t *testing.T) {
	_, err := dotenvPairs([]byte("FAIL=$(printf fake_output_value; false)\n"), nil)
	if err == nil {
		t.Fatal("expected command substitution failure")
	}
	if got := err.Error(); !strings.HasPrefix(got, "command substitution failed:") || strings.Contains(got, "fake_output_value") || strings.Contains(got, "printf") {
		t.Fatalf("unsafe or unexpected error: %q", got)
	}
}

func TestRunSubstitutionHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := runSubstitution(ctx, "sleep 5", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v want context deadline", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("substitution ignored cancellation for %s", elapsed)
	}
}
