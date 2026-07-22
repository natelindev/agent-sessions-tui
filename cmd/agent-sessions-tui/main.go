package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/natelindev/agent-sessions-tui/internal/discovery"
	"github.com/natelindev/agent-sessions-tui/internal/ui"
)

var version = "0.1.2"

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fail("cannot locate the home directory")
	}

	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.StringVar(&home, "home", home, "home directory containing agent session stores")
	flag.Usage = func() {
		name := filepath.Base(os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Browse, search, and resume local coding-agent sessions.\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  %s [--home PATH]\n\nOptions:\n", name)
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nControls:\n  / search   arrows or j/k navigate   Enter resume   r refresh   q quit\n")
	}
	flag.Parse()
	if *showVersion {
		fmt.Printf("agent-sessions-tui %s\n", version)
		return
	}
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	program := tea.NewProgram(
		ui.New(discovery.New(home)),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := program.Run(); err != nil {
		fail(err.Error())
	}
}

func fail(message string) {
	fmt.Printf("error: %s\n", message)
	os.Exit(1)
}
