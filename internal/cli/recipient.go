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
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck recipient add|remove AGE1... -f FILE")
		return 1
	}
	switch args[0] {
	case "add":
		return recipientAdd(args[1:], stderr, getenv)
	case "remove":
		return recipientRemove(args[1:], stderr, getenv)
	default:
		fmt.Fprintln(stderr, "usage: sopsdeck recipient add|remove AGE1... -f FILE")
		return 1
	}
}

func recipientAdd(args []string, stderr io.Writer, getenv func(string) string) int {
	if getenv != nil {
		restore := applyProcessEnv(getenv)
		defer restore()
	}
	file, pub, errMsg := parseRecipientArgs("add", args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
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

func recipientRemove(args []string, stderr io.Writer, getenv func(string) string) int {
	if getenv != nil {
		restore := applyProcessEnv(getenv)
		defer restore()
	}
	file, pub, errMsg := parseRecipientArgs("remove", args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 1
	}
	format := fileFormat(file)
	store := common.StoreForFormat(format, config.NewStoresConfig())
	tree, err := common.LoadEncryptedFile(store, file)
	if err != nil {
		fmt.Fprintf(stderr, "recipient remove: %v\n", err)
		return 1
	}
	if !hasRecipient(*tree, pub) {
		return 0
	}
	if recipientCount(*tree) < 2 {
		fmt.Fprintln(stderr, "recipient remove: refusing to drop the last Recipient")
		return 1
	}
	svcs := []keyservice.KeyServiceClient{keyservice.NewLocalClient()}
	if _, err := common.DecryptTree(common.DecryptTreeOpts{
		Tree:        tree,
		Cipher:      aes.NewCipher(),
		KeyServices: svcs,
	}); err != nil {
		fmt.Fprintf(stderr, "recipient remove: %v\n", err)
		return 1
	}
	dropRecipient(tree, pub)
	dataKey, errs := tree.GenerateDataKeyWithKeyServices(svcs)
	if len(errs) > 0 {
		fmt.Fprintf(stderr, "recipient remove: %v\n", errs)
		return 1
	}
	if err := common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    tree,
		Cipher:  aes.NewCipher(),
	}); err != nil {
		fmt.Fprintf(stderr, "recipient remove: %v\n", err)
		return 1
	}
	out, err := store.EmitEncryptedFile(*tree)
	if err != nil {
		fmt.Fprintf(stderr, "recipient remove: %v\n", err)
		return 1
	}
	if err := writeAtomic(file, out); err != nil {
		fmt.Fprintf(stderr, "recipient remove: %v\n", err)
		return 1
	}
	fmt.Fprintln(stderr, "recipient remove: Access dropped. Git history and copies they already have still decrypt.")
	return 0
}

func parseRecipientArgs(verb string, args []string) (file, pub, errMsg string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				return "", "", "recipient " + verb + ": -f requires a file"
			}
			file = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", "recipient " + verb + ": unknown flag " + args[i]
			}
			if pub != "" {
				return "", "", "recipient " + verb + ": extra argument"
			}
			pub = args[i]
		}
	}
	if file == "" || pub == "" {
		return "", "", "usage: sopsdeck recipient " + verb + " AGE1... -f FILE"
	}
	return file, pub, ""
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

func recipientCount(tree sops.Tree) int {
	n := 0
	for _, group := range tree.Metadata.KeyGroups {
		n += len(group)
	}
	return n
}

func dropRecipient(tree *sops.Tree, pub string) {
	want := strings.ToLower(strings.TrimSpace(pub))
	for i, group := range tree.Metadata.KeyGroups {
		kept := make(sops.KeyGroup, 0, len(group))
		for _, key := range group {
			got := strings.ToLower(key.ToString())
			if got == want || strings.Contains(got, want) {
				continue
			}
			kept = append(kept, key)
		}
		tree.Metadata.KeyGroups[i] = kept
	}
}
