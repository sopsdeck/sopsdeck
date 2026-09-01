package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
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
		fmt.Fprintln(stderr, "usage: sopsdeck recipient add|remove|list|request|grant ...")
		return 1
	}
	switch args[0] {
	case "add":
		return recipientAdd(args[1:], stderr, getenv)
	case "remove":
		return recipientRemove(args[1:], stderr, getenv)
	case "list":
		return recipientList(args[1:], stdout, stderr, getenv)
	case "request":
		return recipientRequest(args[1:], stdout, stderr, getenv)
	case "grant":
		return recipientGrant(args[1:], stdout, stderr, getenv)
	default:
		fmt.Fprintln(stderr, "usage: sopsdeck recipient add|remove|list|request|grant ...")
		return 1
	}
}

func recipientAdd(args []string, stderr io.Writer, getenv func(string) string) int {
	if getenv != nil {
		restore := applyProcessEnv(getenv)
		defer restore()
	}
	file, pub, name, kind, email, errMsg := parseRecipientArgs("add", args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 1
	}
	name, email = displayIdentity(name, email)
	if err := denyUnlessOwner(file, getenv); err != nil {
		fmt.Fprintf(stderr, "recipient add: %v\n", err)
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
		if err := setRecipientLabel(file, pub, name, kind, email); err != nil {
			fmt.Fprintf(stderr, "recipient add: %v\n", err)
			return 1
		}
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
	if err := setRecipientLabel(file, pub, name, kind, email); err != nil {
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
	file, pub, _, _, _, errMsg := parseRecipientArgs("remove", args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 1
	}
	if err := denyUnlessOwner(file, getenv); err != nil {
		fmt.Fprintf(stderr, "recipient remove: %v\n", err)
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
	if err := removeRecipientLabel(file, pub); err != nil {
		fmt.Fprintf(stderr, "recipient remove: %v\n", err)
		return 1
	}
	fmt.Fprintln(stderr, "recipient remove: Access dropped. Git history and copies they already have still decrypt.")
	return 0
}

func removeRecipientLabel(file, key string) error {
	_, manifestPath := findManifest(file)
	if manifestPath == "" {
		return nil
	}
	m, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	kept := m.Recipient[:0]
	for _, item := range m.Recipient {
		if !strings.EqualFold(item.Key, key) {
			kept = append(kept, item)
		}
	}
	m.Recipient = kept
	return writeManifest(manifestPath, m)
}

func parseRecipientArgs(verb string, args []string) (file, pub, name, kind, email, errMsg string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				return "", "", "", "", "", "recipient " + verb + ": -f requires a file"
			}
			file = args[i]
		case "--name":
			i++
			if i >= len(args) {
				return "", "", "", "", "", "recipient " + verb + ": --name requires a value"
			}
			name = strings.TrimSpace(args[i])
		case "--email":
			i++
			if i >= len(args) {
				return "", "", "", "", "", "recipient " + verb + ": --email requires a value"
			}
			email = strings.TrimSpace(args[i])
		case "--kind":
			i++
			if i >= len(args) {
				return "", "", "", "", "", "recipient " + verb + ": --kind requires a value"
			}
			kind = strings.TrimSpace(args[i])
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", "", "", "", "recipient " + verb + ": unknown flag " + args[i]
			}
			if pub != "" {
				return "", "", "", "", "", "recipient " + verb + ": extra argument"
			}
			pub = args[i]
		}
	}
	if file == "" || pub == "" {
		return "", "", "", "", "", "usage: sopsdeck recipient " + verb + " AGE1... -f FILE [--name NAME] [--email EMAIL]"
	}
	return file, pub, name, kind, email, ""
}

func displayIdentity(name, email string) (string, string) {
	parsedName, parsedEmail := splitDisplayIdentity(name)
	email = strings.TrimSpace(email)
	if email == "" {
		email = parsedEmail
	}
	return parsedName, email
}

func splitDisplayIdentity(raw string) (name, email string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	start := strings.LastIndex(raw, "<")
	if start >= 0 && strings.HasSuffix(raw, ">") {
		return strings.TrimSpace(raw[:start]), strings.TrimSpace(raw[start+1 : len(raw)-1])
	}
	return raw, ""
}

type accessRecipient struct {
	Key   string `json:"key"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Kind  string `json:"kind,omitempty"`
	Self  bool   `json:"self,omitempty"`
}

func recipientList(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) != 2 || args[0] != "-f" {
		fmt.Fprintln(stderr, "usage: sopsdeck recipient list -f FILE")
		return 1
	}
	file := args[1]
	format := fileFormat(file)
	tree, err := common.LoadEncryptedFile(common.StoreForFormat(format, config.NewStoresConfig()), file)
	if err != nil {
		fmt.Fprintf(stderr, "recipient list: %v\n", err)
		return 1
	}
	_, root, _ := mappingFor(file)
	m, _ := loadManifest(filepath.Join(root, ".sopsdeck.toml"))
	labels := identityLabels(m)
	self, _ := ageRecipientFromEnv(getenv)
	list := make([]accessRecipient, 0)
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			pub := key.ToString()
			label := labels[strings.ToLower(pub)]
			list = append(list, accessRecipient{
				Key:   pub,
				Name:  label.Name,
				Email: label.Email,
				Kind:  label.Kind,
				Self:  self != "" && strings.EqualFold(pub, self),
			})
		}
	}
	if err := json.NewEncoder(stdout).Encode(list); err != nil {
		fmt.Fprintf(stderr, "recipient list: %v\n", err)
		return 1
	}
	return 0
}

func identityLabels(m projectManifest) map[string]manifestRecipient {
	labels := make(map[string]manifestRecipient, len(m.Owner)+len(m.Recipient))
	for _, item := range m.Owner {
		labels[strings.ToLower(item.Key)] = item
	}
	for _, item := range m.Recipient {
		key := strings.ToLower(item.Key)
		prev := labels[key]
		if item.Name == "" {
			item.Name = prev.Name
		}
		if item.Email == "" {
			item.Email = prev.Email
		}
		if item.Kind == "" {
			item.Kind = prev.Kind
		}
		labels[key] = item
	}
	return labels
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
