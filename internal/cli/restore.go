package cli

import (
	"fmt"
	"io"

	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/getsops/sops/v3/keyservice"
)

func cmdRestore(args []string, stdout, stderr io.Writer) int {
	_ = stdout
	var file, rev string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "restore: -f requires a file")
				return 1
			}
			file = args[i]
		case "--at":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "restore: --at requires a revision")
				return 1
			}
			rev = args[i]
		default:
			fmt.Fprintln(stderr, "usage: sopsdeck restore -f FILE --at REV")
			return 1
		}
	}
	if file == "" || rev == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck restore -f FILE --at REV")
		return 1
	}
	if err := restoreAt(file, rev); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func restoreAt(file, rev string) error {
	format := fileFormat(file)
	raw, err := gitShowAt(file, rev)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	plain, err := decrypt.Data(raw, formatName(format))
	if err != nil {
		return fmt.Errorf("%s", explainRestore(err))
	}
	want, err := secretPairs(plain, format)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	store := common.StoreForFormat(format, config.NewStoresConfig())
	tree, err := common.LoadEncryptedFile(store, file)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	cipher := aes.NewCipher()
	dataKey, err := common.DecryptTree(common.DecryptTreeOpts{
		Tree:        tree,
		Cipher:      cipher,
		KeyServices: []keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	})
	if err != nil {
		return fmt.Errorf("%s", explainRestore(err))
	}
	currentPlain, err := decrypt.File(file, formatName(format))
	if err != nil {
		return fmt.Errorf("%s", explainRestore(err))
	}
	current, err := secretPairs(currentPlain, format)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	for k := range current {
		if _, ok := want[k]; ok {
			continue
		}
		branch, unsetErr := tree.Branches[0].Unset([]interface{}{k})
		if unsetErr != nil {
			return fmt.Errorf("restore: %w", unsetErr)
		}
		tree.Branches[0] = branch
	}
	for k, v := range want {
		tree.Branches[0], _ = tree.Branches[0].Set([]interface{}{k}, v)
	}
	if err := common.EncryptTree(common.EncryptTreeOpts{DataKey: dataKey, Tree: tree, Cipher: cipher}); err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	out, err := store.EmitEncryptedFile(*tree)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	if err := writeAtomic(file, out); err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	return nil
}

func explainRestore(err error) string {
	msg := err.Error()
	if noAccess(msg) {
		return "restore: no Access to this Managed File"
	}
	return "restore: " + firstLine(msg)
}
