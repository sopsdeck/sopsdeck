package changelog

import (
	"fmt"
	"strings"
)

func Notes(md, version string) (string, error) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return "", fmt.Errorf("changelog: version is required")
	}
	body, ok := section(md, version)
	if !ok {
		return "", fmt.Errorf("CHANGELOG.md has no %s section", version)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("CHANGELOG.md %s section is empty", version)
	}
	return body + "\n", nil
}

func Bullets(md, heading string) []string {
	body, ok := section(md, heading)
	if !ok {
		return nil
	}
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			out = append(out, strings.TrimSpace(line[2:]))
		}
	}
	return out
}

func section(md, heading string) (string, bool) {
	lines := strings.Split(md, "\n")
	start := -1
	for i, line := range lines {
		if headingMatch(line, heading) {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), true
}

func headingMatch(line, heading string) bool {
	line = strings.TrimSpace(line)
	switch {
	case line == "## "+heading:
		return true
	case strings.HasPrefix(line, "## "+heading+" "):
		return true
	case strings.HasPrefix(line, "## ["+heading+"]"):
		return true
	default:
		return false
	}
}
