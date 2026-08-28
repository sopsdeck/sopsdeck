package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/keyservice"
)

func cmdRecipient(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	_ = stdout
	if len(args) == 0 || args[0] != "add" {
		fmt.Fprintln(stderr, "usage: sopsdeck recipient add AGE1... -f FILE")
		return 1
	}
	return recipientAdd(args[1:], stderr, getenv)
}

func recipientAdd(args []string, stderr io.Writer, getenv func(string) string) int {
	if getenv != nil {
		restore := applyProcessEnv(getenv)
		defer restore()
	}
	var file, pub string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "recipient add: -f requires a file")
				return 1
			}
			file = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "recipient add: unknown flag %s\n", args[i])
				return 1
			}
			if pub != "" {
				fmt.Fprintln(stderr, "recipient add: extra argument")
				return 1
			}
			pub = args[i]
		}
	}
	if file == "" || pub == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck recipient add AGE1... -f FILE")
		return 1
	}
	format := fileFormat(file)
	store := common.StoreForFormat(format, config.NewStoresConfig())
	tree, err := common.LoadEncryptedFile(store, file)
	if err != nil {
		fmt.Fprintf(stderr, "recipient add: %v\n", err)
		return 1
	}
	if hasRecipient(*tree, pub) {
		return 0
	}
	svcs := []keyservice.KeyServiceClient{keyservice.NewLocalClient()}
	dataKey, err := common.DecryptTree(common.DecryptTreeOpts{
		Tree:        tree,
		Cipher:      aes.NewCipher(),
		KeyServices: svcs,
	})
	if err != nil {
		fmt.Fprintf(stderr, "recipient add: %v\n", err)
		return 1
	}
	mk, err := sopsage.MasterKeyFromRecipient(pub)
	if err != nil {
		fmt.Fprintf(stderr, "recipient add: %v\n", err)
		return 1
	}
	if len(tree.Metadata.KeyGroups) == 0 {
		tree.Metadata.KeyGroups = []sops.KeyGroup{{}}
	}
	tree.Metadata.KeyGroups[0] = append(tree.Metadata.KeyGroups[0], mk)
	if errs := tree.Metadata.UpdateMasterKeysWithKeyServices(dataKey, svcs); len(errs) > 0 {
		fmt.Fprintf(stderr, "recipient add: %v\n", errs)
		return 1
	}
	if err := common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    tree,
		Cipher:  aes.NewCipher(),
	}); err != nil {
		fmt.Fprintf(stderr, "recipient add: %v\n", err)
		return 1
	}
	out, err := store.EmitEncryptedFile(*tree)
	if err != nil {
		fmt.Fprintf(stderr, "recipient add: %v\n", err)
		return 1
	}
	if err := writeAtomic(file, out); err != nil {
		fmt.Fprintf(stderr, "recipient add: %v\n", err)
		return 1
	}
	return 0
}

func hasRecipient(tree sops.Tree, pub string) bool {
	want := strings.ToLower(strings.TrimSpace(pub))
	if want == "" {
		return false
	}
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			got := strings.ToLower(key.ToString())
			if got == want || strings.Contains(got, want) {
				return true
			}
		}
	}
	return false
}
