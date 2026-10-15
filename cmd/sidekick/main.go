// Command sidekick is a pop-up toolbox for developers, inspired by Borland SideKick.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/allanmedeiros71/sidekick-tui/internal/app"
	"github.com/allanmedeiros71/sidekick-tui/internal/config"
	"github.com/allanmedeiros71/sidekick-tui/internal/module/calc"
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
opens that module directly. "sidekick calc <input>" prints the result and exits.

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
		if id == "calc" && len(args) > 1 {
			os.Exit(printCalc(strings.Join(args[1:], " ")))
		}
		root = root.Open(id)
	}

	p := tea.NewProgram(root, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "sidekick: %v\n", err)
		os.Exit(1)
	}
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
