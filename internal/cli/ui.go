package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

const usageLine = "usage: sopsdeck <get|set|del|lock|unlock|status|copy|run|identity|account|robot|commit|sync|review|history|restore|recipient|publish|files|project|references|unused|rename|drive|team|scan|mcp> ..."

var (
	brandStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{
		Light: "#7C3AED",
		Dark:  "#C4B5FD",
	})
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#6B7280",
		Dark:  "#94A3B8",
	})
	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{
		Light: "#111827",
		Dark:  "#F8FAFC",
	})
	commandStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#2563EB",
		Dark:  "#7DD3FC",
	})
)

func printUsage(w io.Writer) {
	if !isTerminal(w) {
		fmt.Fprintln(w, usageLine)
		return
	}

	fmt.Fprintln(w, brandStyle.Render("sopsdeck"), mutedStyle.Render("local-first secrets, without the sprawl"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, headingStyle.Render("Usage"))
	fmt.Fprintf(w, "  %s %s\n\n", commandStyle.Render("sopsdeck"), commandStyle.Render("<command> [args]"))
	fmt.Fprintln(w, headingStyle.Render("Commands"))
	fmt.Fprintln(w, "  "+commandStyle.Render("get set del lock unlock status copy run"), mutedStyle.Render("read, write, lock, or inject secrets"))
	fmt.Fprintln(w, "  "+commandStyle.Render("commit sync review history restore"), mutedStyle.Render("move through Secret History"))
	fmt.Fprintln(w, "  "+commandStyle.Render("identity account robot recipient"), mutedStyle.Render("manage Users and Access"))
	fmt.Fprintln(w, "  "+commandStyle.Render("files project references unused rename"), mutedStyle.Render("organize a Project"))
	fmt.Fprintln(w, "  "+commandStyle.Render("publish scan drive team mcp"), mutedStyle.Render("connect, protect, and automate"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, mutedStyle.Render("Run 'sopsdeck <command>' for command-specific usage."))
}

func isTerminal(w io.Writer) bool {
	if os.Getenv("SOPSDECK_COLOR") == "always" {
		lipgloss.SetColorProfile(termenv.TrueColor)
		return true
	}
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	fdWriter, ok := w.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(int(fdWriter.Fd()))
}
