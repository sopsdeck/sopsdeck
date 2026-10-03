package cli

import (
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/getsops/sops/v3/cmd/sops/formats"
)

type redactionPair struct {
	key   string
	value string
}

type redactor struct {
	pairs    []redactionPair
	replacer *strings.Replacer
}

func newRedactor(pairs map[string]string) *redactor {
	values := make([]redactionPair, 0, len(pairs))
	for key, value := range pairs {
		values = append(values, redactionPair{key: key, value: value})
	}
	return newRedactorValues(values)
}

func newRedactorValues(values []redactionPair) *redactor {
	r := &redactor{}
	for _, pair := range values {
		if pair.value != "" {
			r.pairs = append(r.pairs, pair)
		}
	}
	sort.Slice(r.pairs, func(i, j int) bool {
		if len(r.pairs[i].value) != len(r.pairs[j].value) {
			return len(r.pairs[i].value) > len(r.pairs[j].value)
		}
		if r.pairs[i].value != r.pairs[j].value {
			return r.pairs[i].value < r.pairs[j].value
		}
		return r.pairs[i].key < r.pairs[j].key
	})
	if len(r.pairs) == 0 {
		return r
	}

	args := make([]string, 0, len(r.pairs)*2)
	unique := r.pairs[:0]
	for _, pair := range r.pairs {
		if len(unique) > 0 && unique[len(unique)-1].value == pair.value {
			continue
		}
		unique = append(unique, pair)
		args = append(args, pair.value, "[sopsdeck:"+pair.key+"]")
	}
	r.pairs = unique
	r.replacer = strings.NewReplacer(args...)
	return r
}

func redactionPairs(file string, pairs map[string]string, includeInherited bool, encryptedKeys []string, encryptsAll bool, regex string) []redactionPair {
	mapping, _, _ := mappingFor(file)
	format := fileFormat(file)
	policyKnown := encryptsAll || len(encryptedKeys) > 0 || regex != ""
	values := make([]redactionPair, 0, len(pairs)*2)
	for key, value := range pairs {
		if format != formats.Dotenv && (key == "sops" || strings.HasPrefix(key, "sops.") || strings.HasPrefix(key, "sops[")) {
			continue
		}
		if publicRedactionPath(key, mapping.PublicKeys, format) {
			continue
		}
		if format != formats.Dotenv && policyKnown && !pairEncrypted(key, encryptedKeys, encryptsAll, regex) {
			continue
		}
		if value != "" {
			values = append(values, redactionPair{key: key, value: value})
		}
		if format == formats.Dotenv && includeInherited {
			if inherited, ok := os.LookupEnv(key); ok && inherited != "" && inherited != value {
				values = append(values, redactionPair{key: key, value: inherited})
			}
		}
	}
	return values
}

func publicRedactionPath(key string, public []string, format formats.Format) bool {
	for _, path := range public {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if key == path || (format != formats.Dotenv && (strings.HasPrefix(key, path+".") || strings.HasPrefix(key, path+"["))) {
			return true
		}
	}
	return false
}

func (r *redactor) redact(s string) string {
	if r.replacer == nil {
		return s
	}
	return r.replacer.Replace(s)
}

// safePrefixLen keeps only a suffix that is still a prefix of a secret.
// ponytail: scan candidates directly; build a prefix trie if large sets make this hot.
func (r *redactor) safePrefixLen(s string) int {
	for i := 0; i < len(s); {
		matched := false
		for _, pair := range r.pairs {
			if strings.HasPrefix(s[i:], pair.value) {
				i += len(pair.value)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		for _, pair := range r.pairs {
			if len(s)-i < len(pair.value) && strings.HasPrefix(pair.value, s[i:]) {
				return i
			}
		}
		i++
	}
	return len(s)
}

// outputWriter wraps a child output stream with streaming redaction.
func (r childRunner) outputWriter(w io.Writer) io.Writer {
	if r.redact == nil || len(r.redact.pairs) == 0 {
		return w
	}
	return &redactWriter{redactor: r.redact, out: w}
}

type redactWriter struct {
	redactor *redactor
	out      io.Writer
	tail     string
	err      error
}

func (w *redactWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	buf := w.tail + string(p)
	flushLen := w.redactor.safePrefixLen(buf)
	for flushLen > 0 && !utf8.ValidString(buf[:flushLen]) {
		flushLen--
	}
	chunk := w.redactor.redact(buf[:flushLen])
	if chunk != "" {
		n, err := io.WriteString(w.out, chunk)
		if err == nil && n < len(chunk) {
			err = io.ErrShortWrite
		}
		if err != nil {
			w.tail = ""
			w.err = err
			return 0, err
		}
	}
	w.tail = buf[flushLen:]
	return len(p), nil
}

func (w *redactWriter) Flush() error {
	if w.err != nil {
		return w.err
	}
	if w.tail == "" {
		return nil
	}
	chunk := w.redactor.redact(w.tail)
	n, err := io.WriteString(w.out, chunk)
	if err == nil && n < len(chunk) {
		err = io.ErrShortWrite
	}
	w.tail = ""
	w.err = err
	return err
}
