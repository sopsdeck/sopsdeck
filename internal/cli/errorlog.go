package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const errorLogName = "errors.json"

type errorRecord struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
	Last    string `json:"last"`
}

var (
	errorLogMu      sync.Mutex
	ageSecretRE     = regexp.MustCompile(`AGE-SECRET-KEY-[A-Z0-9-]+`)
	encCiphertextRE = regexp.MustCompile(`ENC\[[^\]]*\]`)
)

func recordError(getenv func(string) string, raw string) {
	if getenv == nil {
		return
	}
	dir := getenv("SOPSDECK_STATE_DIR")
	if dir == "" {
		return
	}
	msg := redactError(strings.TrimSpace(raw))
	if msg == "" || strings.HasPrefix(msg, "usage:") {
		return
	}

	errorLogMu.Lock()
	defer errorLogMu.Unlock()

	path := filepath.Join(dir, errorLogName)
	records, _ := readErrorRecords(path)
	now := time.Now().UTC().Format(time.RFC3339)
	found := false
	for i := range records {
		if records[i].Message == msg {
			records[i].Count++
			records[i].Last = now
			found = true
			break
		}
	}
	if !found {
		records = append(records, errorRecord{Message: msg, Count: 1, Last: now})
	}
	body, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(dir, 0o700)
	_ = writeAtomic(path, append(body, '\n'))
}

func readErrorRecords(path string) ([]errorRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []errorRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func redactError(msg string) string {
	msg = ageSecretRE.ReplaceAllString(msg, "AGE-SECRET-KEY-[redacted]")
	msg = encCiphertextRE.ReplaceAllString(msg, "ENC[redacted]")
	return msg
}
