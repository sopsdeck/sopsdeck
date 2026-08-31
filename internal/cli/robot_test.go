package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestRobotCreateEmitsExportableNamedIdentity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"robot", "create", "deploy bot"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("robot create exit %d stderr=%q", code, stderr.String())
	}
	var robot robotIdentity
	if err := json.Unmarshal(stdout.Bytes(), &robot); err != nil {
		t.Fatal(err)
	}
	if robot.Name != "deploy bot" || !strings.HasPrefix(robot.PublicKey, "age1") {
		t.Fatalf("robot=%+v", robot)
	}
	ids, err := age.ParseIdentities(strings.NewReader(robot.PrivateKey))
	if err != nil || len(ids) != 1 {
		t.Fatalf("private key is not importable: %v", err)
	}
}
