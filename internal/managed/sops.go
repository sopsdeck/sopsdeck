package managed

import (
	"os"
	"strings"
)

// looksSOPSDotenv reports whether a dotenv file carries SOPS metadata.
// SOPS dotenv files always emit sops_-prefixed lines (sops_mac,
// sops_age__list..., sops_lastmodified, ...); plain dotenv files do not.
func looksSOPSDotenv(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if len(data) > 16_384 {
		data = data[:16_384]
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "sops_") {
			return true
		}
	}
	return false
}
