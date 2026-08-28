package cli

import (
	"errors"
	"os/exec"
	"strings"
)

func explainSync(err error) string {
	msg := err.Error()
	switch {
	case noUpstream(msg):
		return "sync: this branch has no upstream. Push to origin once, then Sync again."
	case diverged(msg):
		return "sync: this branch has diverged from origin. Sync never force-pushes."
	default:
		return "sync: " + firstLine(msg)
	}
}

func explainReview(err error) string {
	msg := err.Error()
	if noAccess(msg) {
		return "review: no Access to this Managed File"
	}
	return "review: " + firstLine(msg)
}

func explainGet(err error) string {
	msg := err.Error()
	if noAccess(msg) {
		return "get: no Access to this Managed File"
	}
	return "get: " + firstLine(msg)
}

func explainPublish(err error) string {
	_ = err
	return "publish: Publish did not finish. Retry Publish."
}

func noUpstream(msg string) bool {
	return strings.Contains(msg, "no tracking information") ||
		strings.Contains(msg, "no upstream") ||
		strings.Contains(msg, "has no upstream branch")
}

func diverged(msg string) bool {
	return strings.Contains(msg, "Not possible to fast-forward") ||
		strings.Contains(msg, "cannot fast-forward") ||
		strings.Contains(strings.ToLower(msg), "diverged")
}

func noAccess(msg string) bool {
	return strings.Contains(msg, "no identity matched") ||
		strings.Contains(msg, "Failed to get the data key") ||
		strings.Contains(msg, "Error getting data key") ||
		strings.Contains(msg, "successful groups required") ||
		strings.Contains(msg, "could not decrypt")
}

func firstLine(msg string) string {
	line, _, _ := strings.Cut(msg, "\n")
	return strings.TrimSpace(line)
}

func gitWorktreeDirtyAt(dir string) (bool, error) {
	cmd := exec.Command("git", "diff-index", "--quiet", "HEAD")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	var status *exec.ExitError
	if errors.As(err, &status) && status.ExitCode() == 1 {
		return true, nil
	}
	return false, err
}
