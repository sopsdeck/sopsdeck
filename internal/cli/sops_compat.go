package cli

import (
	"bytes"
	"os"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/cmd/sops/formats"
	"github.com/getsops/sops/v3/decrypt"
)

// dotenvSOPSLoader removes Windows line endings before SOPS parses dotenv
// metadata. SOPS' dotenv timestamp parser treats the carriage return as part
// of the timestamp on Windows.
type dotenvSOPSLoader struct {
	sops.EncryptedFileLoader
	format formats.Format
}

func (l dotenvSOPSLoader) LoadEncryptedFile(data []byte) (sops.Tree, error) {
	if l.format == formats.Dotenv {
		data = normalizeSOPSDotenv(data)
	}
	return l.EncryptedFileLoader.LoadEncryptedFile(data)
}

func loadEncryptedSOPSFile(loader sops.EncryptedFileLoader, file string) (*sops.Tree, error) {
	return common.LoadEncryptedFile(dotenvSOPSLoader{
		EncryptedFileLoader: loader,
		format:              fileFormat(file),
	}, file)
}

func decryptSOPSFile(file string, format formats.Format) ([]byte, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return decryptSOPSData(data, format)
}

func decryptSOPSData(data []byte, format formats.Format) ([]byte, error) {
	if format == formats.Dotenv {
		data = normalizeSOPSDotenv(data)
	}
	return decrypt.Data(data, formatName(format))
}

func normalizeSOPSDotenv(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}
