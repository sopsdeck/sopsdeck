package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func cmdPublish(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	file, prefix, yes, prune, errMsg := parsePublishFlags(args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 1
	}
	base := getenv("SOPSDECK_GITHUB_API")
	if base == "" {
		fmt.Fprintln(stderr, "publish: SOPSDECK_GITHUB_API is required (local fake or GitHub api root)")
		return 1
	}
	repo := getenv("SOPSDECK_GITHUB_REPO")
	if repo == "" {
		repo = "studio/demo"
	}
	var dump bytes.Buffer
	var errBuf bytes.Buffer
	if code := cmdGet([]string{"-f", file, "--output", "json"}, &dump, &errBuf); code != 0 {
		fmt.Fprintf(stderr, "publish: %s", errBuf.String())
		return 1
	}
	var pairs map[string]string
	if err := json.Unmarshal(dump.Bytes(), &pairs); err != nil {
		fmt.Fprintf(stderr, "publish: %v\n", err)
		return 1
	}
	var names []string
	for key := range pairs {
		names = append(names, prefix+key)
	}
	if !yes {
		fmt.Fprintf(stdout, "dry-run %d secrets for %s\n", len(names), repo)
		for _, n := range names {
			fmt.Fprintln(stdout, n)
		}
		return 0
	}
	client := &http.Client{}
	if err := putSecrets(client, base, repo, names); err != nil {
		fmt.Fprintln(stderr, explainPublish(err))
		return 1
	}
	if prune {
		if err := pruneSecrets(client, base, repo, prefix, names, stderr); err != nil {
			fmt.Fprintf(stderr, "publish: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "published %d secrets to %s\n", len(names), repo)
	return 0
}

func parsePublishFlags(args []string) (file, prefix string, yes, prune bool, errMsg string) {
	usage := "usage: sopsdeck publish -f FILE [--prefix P] [--yes] [--prune]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f":
			i++
			if i >= len(args) {
				return "", "", false, false, "publish: -f requires a file"
			}
			file = args[i]
		case "--prefix":
			i++
			if i >= len(args) {
				return "", "", false, false, "publish: --prefix requires a value"
			}
			prefix = args[i]
		case "--yes":
			yes = true
		case "--prune":
			prune = true
		default:
			return "", "", false, false, usage
		}
	}
	if file == "" {
		return "", "", false, false, usage
	}
	return file, prefix, yes, prune, ""
}

func putSecrets(client *http.Client, base, repo string, names []string) error {
	for _, name := range names {
		endpoint, err := url.JoinPath(base, "repos", repo, "actions", "secrets", name)
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

func pruneSecrets(client *http.Client, base, repo, prefix string, keep []string, stderr io.Writer) error {
	_ = stderr
	want := map[string]struct{}{}
	for _, n := range keep {
		want[n] = struct{}{}
	}
	listURL, err := url.JoinPath(base, "repos", repo, "actions", "secrets")
	if err != nil {
		return err
	}
	resp, err := client.Get(listURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	var payload struct {
		Secrets []struct {
			Name string `json:"name"`
		} `json:"secrets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	for _, sec := range payload.Secrets {
		if prefix != "" && !strings.HasPrefix(sec.Name, prefix) {
			continue
		}
		if _, ok := want[sec.Name]; ok {
			continue
		}
		delURL, err := url.JoinPath(base, "repos", repo, "actions", "secrets", sec.Name)
		if err != nil {
			return err
		}
		req, err := http.NewRequest(http.MethodDelete, delURL, nil)
		if err != nil {
			return err
		}
		delResp, err := client.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, delResp.Body)
		_ = delResp.Body.Close()
	}
	return nil
}
