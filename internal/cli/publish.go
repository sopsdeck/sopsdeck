package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strings"
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
	names := prefixedNames(selectKeys(pairs, mapping.Keys), prefix)
	if !yes {
		fmt.Fprintf(stdout, "dry-run %d secrets for %s\n", len(names), repo)
		for _, n := range names {
			fmt.Fprintln(stdout, n)
		}
		return 0
	}
	client := &http.Client{}
	token := githubToken(getenv)
	if err := putSecrets(client, base, repo, mapping.Environment, token, names); err != nil {
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

func prefixedNames(pairs map[string]string, prefix string) []string {
	var names []string
	for key := range pairs {
		names = append(names, prefix+key)
	}
	return names
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

func putSecrets(client *http.Client, base, repo, environment, token string, names []string) error {
	for _, name := range names {
		endpoint, err := secretURL(base, repo, environment, name)
		if err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]string{
			"encrypted_value": "studio",
			"key_id":          "studio",
		})
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
