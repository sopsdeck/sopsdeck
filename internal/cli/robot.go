package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

type robotIdentity struct {
	Name       string `json:"name"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

func cmdRobot(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "create" || strings.TrimSpace(args[1]) == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck robot create NAME")
		return 1
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		fmt.Fprintf(stderr, "robot create: %v\n", err)
		return 1
	}
	value := robotIdentity{
		Name:       strings.TrimSpace(args[1]),
		PublicKey:  id.Recipient().String(),
		PrivateKey: fmt.Sprintf("# public key: %s\n%s\n", id.Recipient(), id),
	}
	if err := json.NewEncoder(stdout).Encode(value); err != nil {
		fmt.Fprintf(stderr, "robot create: %v\n", err)
		return 1
	}
	return 0
}
