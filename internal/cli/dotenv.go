package cli

import (
	"fmt"
	"strings"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/cmd/sops/formats"
	"github.com/getsops/sops/v3/config"
)

func loadPlainBranches(format formats.Format, plain []byte) (sops.TreeBranches, error) {
	if format == formats.Dotenv {
		return parseDotenv(plain)
	}
	return common.StoreForFormat(format, config.NewStoresConfig()).LoadPlainFile(plain)
}

func parseDotenv(plain []byte) (sops.TreeBranches, error) {
	branches, _, err := parseDotenvQuoted(plain)
	return branches, err
}

// parseDotenvQuoted also reports which keys had single-quoted values; dotenvx
// treats those as literals and skips expansion for them.
func parseDotenvQuoted(plain []byte) (sops.TreeBranches, map[string]bool, error) {
	lines := strings.Split(strings.ReplaceAll(string(plain), "\r\n", "\n"), "\n")
	var branch sops.TreeBranch
	single := map[string]bool{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}
		if line[0] == '#' {
			branch = append(branch, sops.TreeItem{Key: sops.Comment{Value: line[1:]}, Value: nil})
			continue
		}
		pos := strings.IndexByte(line, '=')
		if pos < 1 {
			return nil, nil, fmt.Errorf("invalid dotenv input line: %s", line)
		}
		key, value := line[:pos], line[pos+1:]
		quoted := byte(0)
		if len(value) > 0 && (value[0] == '\'' || value[0] == '"') {
			quote := value[0]
			quoted = quote
			firstLine := line
			for {
				end, ok := dotenvQuoteEnd(value, quote)
				if ok {
					value = value[1:end]
					break
				}
				if i+1 == len(lines) {
					return nil, nil, fmt.Errorf("invalid dotenv input line: %s", firstLine)
				}
				i++
				value += "\n" + lines[i]
			}
		}
		if quoted == '\'' {
			single[key] = true
		}
		branch = append(branch, sops.TreeItem{Key: key, Value: strings.ReplaceAll(value, `\n`, "\n")})
	}
	return sops.TreeBranches{branch}, single, nil
}

func dotenvQuoteEnd(value string, quote byte) (int, bool) {
	end := len(value) - 1
	for end >= 0 && (value[end] == ' ' || value[end] == '\t' || value[end] == '\r') {
		end--
	}
	if end <= 0 || value[end] != quote {
		return 0, false
	}
	slashes := 0
	for i := end - 1; i >= 0 && value[i] == '\\'; i-- {
		slashes++
	}
	return end, slashes%2 == 0
}

func parseDotenvMap(plain []byte) (map[string]string, error) {
	branches, err := parseDotenv(plain)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, item := range branches[0] {
		key, ok := item.Key.(string)
		if !ok {
			continue
		}
		out[key] = fmt.Sprint(item.Value)
	}
	return out, nil
}

func dotenvMap(plain []byte) map[string]string {
	out, _ := parseDotenvMap(plain)
	return out
}
