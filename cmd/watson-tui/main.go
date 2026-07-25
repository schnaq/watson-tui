package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/tui"
	"github.com/schnaq/watson-tui/internal/watson"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	dirFlag := flag.String("dir", "", "watson data directory (default: $WATSON_DIR or OS app dir)")
	flag.Parse()
	if *showVersion {
		fmt.Println("watson-tui", version)
		return
	}
	dir, err := watson.Dir(*dirFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	app := tui.NewApp(watson.NewStore(dir), version)
	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
