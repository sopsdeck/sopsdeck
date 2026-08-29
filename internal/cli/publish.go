package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strings"

	"golang.org/x/crypto/nacl/box"
)

func cmdPublish(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	file, prefix, yes, prune, printMapping, errMsg := parsePublishFlags(args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 1
	}
	mapping, prefix, repo, manifestPath := resolvePublishMapping(file, prefix, getenv)
	if printMapping {
		if err := json.NewEncoder(stdout).Encode(map[string]any{
			"repo":        repo,
			"environment": mapping.Environment,
			"prefix":      prefix,
			"keys":        mapping.Keys,
		}); err != nil {
			fmt.Fprintf(stderr, "publish: %v\n", err)
			return 1
		}
		return 0
	}
	base := getenv("SOPSDECK_GITHUB_API")
	if base == "" {
		fmt.Fprintln(stderr, "publish: SOPSDECK_GITHUB_API is required (local fake or GitHub api root)")
		return 1
	}
	pairs, errMsg := decryptPublishPairs(file)
	if errMsg != "" {
		fmt.Fprintf(stderr, "publish: %s", errMsg)
		return 1
	}
	secrets := prefixedPairs(selectKeys(pairs, mapping.Keys), prefix)
	names := sortedKeys(secrets)
	if !yes {
		fmt.Fprintf(stdout, "dry-run %d secrets for %s\n", len(names), repo)
		for _, n := range names {
			fmt.Fprintln(stdout, n)
		}
		return 0
	}
	client := &http.Client{}
	token := githubToken(getenv)
	if err := putSecrets(client, base, repo, mapping.Environment, token, secrets); err != nil {
		fmt.Fprintln(stderr, explainPublish(err))
		return 1
	}
	if prune {
		if err := pruneSecrets(client, base, repo, mapping.Environment, prefix, token, names, mapping.Published); err != nil {
			fmt.Fprintf(stderr, "publish: %v\n", err)
			return 1
		}
	}
	if err := recordPublished(mapping, manifestPath, names); err != nil {
		fmt.Fprintf(stderr, "publish: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "published %d secrets to %s\n", len(names), repo)
	return 0
}

func resolvePublishMapping(file, prefix string, getenv func(string) string) (manifestFile, string, string, string) {
	mapping, _, manifestPath := mappingFor(file)
	if prefix == "" {
		prefix = mapping.Prefix
	}
	repo := mapping.Repo
	if repo == "" {
		repo = getenv("SOPSDECK_GITHUB_REPO")
	}
	if repo == "" {
		repo = "studio/demo"
	}
	return mapping, prefix, repo, manifestPath
}

func decryptPublishPairs(file string) (map[string]string, string) {
	var dump bytes.Buffer
	var errBuf bytes.Buffer
	if code := cmdGet([]string{"-f", file, "--output", "json"}, &dump, &errBuf); code != 0 {
		return nil, errBuf.String()
	}
	var pairs map[string]string
	if err := json.Unmarshal(dump.Bytes(), &pairs); err != nil {
		return nil, err.Error() + "\n"
	}
	return pairs, ""
}

func selectKeys(pairs map[string]string, keys []string) map[string]string {
	if len(keys) == 0 {
		return pairs
	}
	want := map[string]struct{}{}
	for _, k := range keys {
		want[k] = struct{}{}
	}
	selected := map[string]string{}
	for k, v := range pairs {
		if _, ok := want[k]; ok {
			selected[k] = v
		}
	}
	return selected
}

func prefixedPairs(pairs map[string]string, prefix string) map[string]string {
	secrets := make(map[string]string, len(pairs))
	for key, value := range pairs {
		secrets[prefix+key] = value
	}
	return secrets
}

func recordPublished(mapping manifestFile, manifestPath string, names []string) error {
	if mapping.Path == "" || manifestPath == "" {
		return nil
	}
	recorded := append([]string(nil), names...)
	sort.Strings(recorded)
	return setPublished(manifestPath, mapping.Path, recorded)
}

func parsePublishFlags(args []string) (file, prefix string, yes, prune, printMapping bool, errMsg string) {
	usage := "usage: sopsdeck publish -f FILE [--prefix P] [--yes] [--prune] [--mapping]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f":
			i++
			if i >= len(args) {
				return "", "", false, false, false, "publish: -f requires a file"
			}
			file = args[i]
		case "--prefix":
			i++
			if i >= len(args) {
				return "", "", false, false, false, "publish: --prefix requires a value"
			}
			prefix = args[i]
		case "--yes":
			yes = true
		case "--prune":
			prune = true
		case "--mapping":
			printMapping = true
		default:
			return "", "", false, false, false, usage
		}
	}
	if file == "" {
		return "", "", false, false, false, usage
	}
	return file, prefix, yes, prune, printMapping, ""
}

func githubToken(getenv func(string) string) string {
	if v := getenv("GH_TOKEN"); v != "" {
		return v
	}
	if v := getenv("GITHUB_TOKEN"); v != "" {
		return v
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func setAuth(req *http.Request, token string) {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func putSecrets(client *http.Client, base, repo, environment, token string, secrets map[string]string) error {
	keyID, publicKey, err := githubPublicKey(client, base, repo, environment, token)
	if err != nil {
		return err
	}
	for _, name := range sortedKeys(secrets) {
		endpoint, err := secretURL(base, repo, environment, name)
		if err != nil {
			return err
		}
		sealed, err := box.SealAnonymous(nil, []byte(secrets[name]), publicKey, rand.Reader)
		if err != nil {
			return err
		}
		body, err := json.Marshal(map[string]string{
			"encrypted_value": base64.StdEncoding.EncodeToString(sealed),
			"key_id":          keyID,
		})
		if err != nil {
			return err
		}
		req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("content-type", "application/json")
		setAuth(req, token)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("PUT %s → %s", name, resp.Status)
		}
	}
	return nil
}

func githubPublicKey(client *http.Client, base, repo, environment, token string) (string, *[32]byte, error) {
	endpoint, err := publicKeyURL(base, repo, environment)
	if err != nil {
		return "", nil, err
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", nil, err
	}
	setAuth(req, token)
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", nil, fmt.Errorf("GET public key → %s", resp.Status)
	}
	var payload struct {
		KeyID string `json:"key_id"`
		Key   string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.Key)
	if err != nil {
		return "", nil, fmt.Errorf("decode GitHub public key: %w", err)
	}
	if len(decoded) != 32 {
		return "", nil, fmt.Errorf("GitHub public key has %d bytes, want 32", len(decoded))
	}
	var publicKey [32]byte
	copy(publicKey[:], decoded)
	return payload.KeyID, &publicKey, nil
}

func pruneSecrets(client *http.Client, base, repo, environment, prefix, token string, keep, previously []string) error {
	want := map[string]struct{}{}
	for _, n := range keep {
		want[n] = struct{}{}
	}
	for _, name := range previously {
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		if _, ok := want[name]; ok {
			continue
		}
		delURL, err := secretURL(base, repo, environment, name)
		if err != nil {
			return err
		}
		req, err := http.NewRequest(http.MethodDelete, delURL, nil)
		if err != nil {
			return err
		}
		setAuth(req, token)
		delResp, err := client.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, delResp.Body)
		_ = delResp.Body.Close()
	}
	return nil
}

func secretURL(base, repo, environment, name string) (string, error) {
	if environment != "" {
		return url.JoinPath(base, "repos", repo, "environments", environment, "secrets", name)
	}
	return url.JoinPath(base, "repos", repo, "actions", "secrets", name)
}

func publicKeyURL(base, repo, environment string) (string, error) {
	if environment != "" {
		return url.JoinPath(base, "repos", repo, "environments", environment, "secrets", "public-key")
	}
	return url.JoinPath(base, "repos", repo, "actions", "secrets", "public-key")
}
