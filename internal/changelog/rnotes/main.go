package main

import (
	"fmt"
	"os"

	"sopsdeck/internal/changelog"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: release-notes vX.Y.Z")
		os.Exit(2)
	}
	md, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	notes, err := changelog.Notes(string(md), os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprint(os.Stdout, notes)
}
