package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const errorLogName = "errors.json"

type errorRecord struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
	Last    string `json:"last"`
}

var errorLogMu sync.Mutex

func recordError(getenv func(string) string, code int) {
	if getenv == nil {
		return
	}
	dir := getenv("SOPSDECK_STATE_DIR")
	if dir == "" {
		return
	}
	msg := fmt.Sprintf("command failed (exit %d)", code)

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
