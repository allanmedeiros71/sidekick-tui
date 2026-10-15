// Command sidekick is a pop-up toolbox for developers, inspired by Borland SideKick.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/allanmedeiros71/sidekick-tui/internal/app"
	"github.com/allanmedeiros71/sidekick-tui/internal/config"
	"github.com/allanmedeiros71/sidekick-tui/internal/module/calc"
	"github.com/allanmedeiros71/sidekick-tui/internal/module/inspect"
)

// Set by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	configPath := flag.String("config", "", "path to config file (default: ~/.config/sidekick/config.toml)")
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `usage: sidekick [flags] [module] [input...]

With no module, sidekick opens the menu. With a module name (e.g. calc) it
opens that module directly. "sidekick calc <input>" and "sidekick inspect
<input>" print the result and exit; they also read the input from a pipe.

flags:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("sidekick %s (%s, %s)\n", version, commit, date)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sidekick: %v\n", err)
		os.Exit(1)
	}

	root := app.New(cfg)
	if args := flag.Args(); len(args) > 0 {
		id := args[0]
		if !app.Implemented(id) {
			fmt.Fprintf(os.Stderr, "sidekick: unknown module %q\n", id)
			os.Exit(2)
		}
		if run, ok := printers[id]; ok {
			if input, ok := cliInput(args[1:]); ok {
				os.Exit(run(input))
			}
		}
		root = root.Open(id)
	}

	p := tea.NewProgram(root, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "sidekick: %v\n", err)
		os.Exit(1)
	}
}

// printers run a module non-interactively on the given input.
var printers = map[string]func(string) int{
	"calc":    printCalc,
	"inspect": printInspect,
}

// cliInput returns the input given on the command line or, when there is
// none and stdin is not a terminal, read from stdin without its final line
// break. ok is false when the module should open interactively.
func cliInput(args []string) (string, bool) {
	if len(args) > 0 {
		return strings.Join(args, " "), true
	}
	if fi, err := os.Stdin.Stat(); err != nil || fi.Mode()&os.ModeCharDevice != 0 {
		return "", false
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sidekick: %v\n", err)
		os.Exit(1)
	}
	s := strings.TrimSuffix(string(b), "\n")
	return strings.TrimSuffix(s, "\r"), true
}

// printInspect describes input non-interactively, one section at a time.
func printInspect(input string) int {
	r := inspect.Inspect(input, time.Now())
	if len(r.Sections) == 0 {
		fmt.Fprintln(os.Stderr, "sidekick: nothing to inspect")
		return 1
	}
	width := 0
	for _, l := range r.Lines() {
		width = max(width, len(l.Label))
	}
	indent := strings.Repeat(" ", width+4)
	for i, s := range r.Sections {
		if i > 0 {
			fmt.Println()
		}
		fmt.Println(s.Title)
		for _, l := range s.Lines {
			value := strings.ReplaceAll(l.Value, "\n", "\n"+indent)
			fmt.Printf("  %-*s  %s\n", width, l.Label, value)
		}
	}
	return 0
}

// printCalc evaluates input non-interactively, for scripts and quick checks.
func printCalc(input string) int {
	r, err := calc.Eval(input, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "sidekick: %v\n", err)
		return 1
	}
	width := 0
	for _, l := range r.Lines {
		width = max(width, len(l.Label))
	}
	for _, l := range r.Lines {
		fmt.Printf("%-*s  %s\n", width, l.Label, l.Value)
	}
	return 0
}
