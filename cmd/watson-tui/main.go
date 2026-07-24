package main

import (
	"flag"
	"fmt"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("watson-tui", version)
		return
	}
	fmt.Println("watson-tui: TUI kommt in Task 8")
}
