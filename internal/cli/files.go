package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"sopsdeck/internal/managed"
)

func cmdFiles(args []string, stdout, stderr io.Writer) int {
	root := "."
	if len(args) == 1 {
		root = args[0]
	} else if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck files [FOLDER]")
		return 1
	}
	files, err := managed.List(root)
	if err != nil {
		if errors.Is(err, fs.ErrInvalid) {
			fmt.Fprintln(stderr, "files: not a folder")
			return 1
		}
		fmt.Fprintf(stderr, "files: %v\n", err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(files); err != nil {
		fmt.Fprintf(stderr, "files: %v\n", err)
		return 1
	}
	return 0
}
