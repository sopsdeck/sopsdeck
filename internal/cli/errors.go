package cli

import (
	"strings"
)

func explainReview(err error) string {
	msg := err.Error()
	if noAccess(msg) {
		return "review: no Access to this Managed File"
	}
	return "review: " + firstLine(msg)
}

func explainGet(err error) string {
	msg := err.Error()
	switch {
	case noAccess(msg):
		return "get: no Access to this Managed File (create or import an identity with `sopsdeck identity`)"
	case notSOPS(msg):
		return "get: not a SOPS-encrypted file (use Add file to encrypt it)"
	default:
		return "get: " + firstLine(msg)
	}
}

func notSOPS(msg string) bool {
	return strings.Contains(msg, "invalid dotenv input line") ||
		strings.Contains(msg, `cannot parse ""`)
}

func explainSyncSecrets(err error) string {
	_ = err
	return "sync: Secret Sync did not finish. Retry sync."
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
