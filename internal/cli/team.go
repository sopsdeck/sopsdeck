package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"sopsdeck/internal/studio"
)

func cmdTeam(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck team init DIR | project DIR NAME")
		return 1
	}
	switch args[0] {
	case "init":
		if len(args) != 2 || args[1] == "" {
			fmt.Fprintln(stderr, "usage: sopsdeck team init DIR")
			return 1
		}
		return teamInit(args[1], stdout, stderr)
	case "project":
		if len(args) != 3 || args[1] == "" || args[2] == "" {
			fmt.Fprintln(stderr, "usage: sopsdeck team project DIR NAME")
			return 1
		}
		return teamProject(args[1], args[2], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: sopsdeck team init DIR | project DIR NAME")
		return 1
	}
}

func teamInit(dir string, stdout, stderr io.Writer) int {
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(stderr, "team init: %v\n", err)
		return 1
	}
	st, alice, bob, err := studio.Prepare(abs)
	if err != nil {
		fmt.Fprintf(stderr, "team init: %v\n", err)
		return 1
	}
	st.Close()
	fmt.Fprintf(stdout, "local teammates\n  studio  %s\n\n", abs)
	printTeamUser(stdout, "alice", alice)
	fmt.Fprintln(stdout)
	printTeamUser(stdout, "bob", bob)
	fmt.Fprintf(stdout, "\ncheckout is cloned into both homes. Do not open the same folder in both windows.\n")
	fmt.Fprintf(stdout, "Another shared Project: sopsdeck team project %s NAME\n", abs)
	return 0
}

func printTeamUser(stdout io.Writer, label string, u *studio.User) {
	fmt.Fprintf(stdout, "  %s   %s\n", label, u.Home)
	fmt.Fprintf(stdout, "        home %s\n", u.ConfigHome)
	fmt.Fprintf(stdout, "        . %s\n", filepath.Join(filepath.Dir(u.ConfigHome), u.Name+".env"))
	fmt.Fprintf(stdout, "        %s\n", u.PublicKey)
}

func teamProject(dir, name string, stdout, stderr io.Writer) int {
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(stderr, "team project: %v\n", err)
		return 1
	}
	st, _, _, err := studio.Prepare(abs)
	if err != nil {
		fmt.Fprintf(stderr, "team project: %v\n", err)
		return 1
	}
	defer st.Close()
	aliceDir, bobDir, err := st.SharedProject(name)
	if err != nil {
		fmt.Fprintf(stderr, "team project: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "alice  %s\nbob    %s\nEach is a clone in that person's home. Add that path only in that window.\n", aliceDir, bobDir)
	return 0
}
