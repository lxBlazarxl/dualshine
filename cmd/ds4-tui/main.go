package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/blazar/ds4-tui/internal/ds4"
	"github.com/blazar/ds4-tui/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

var version = "1.0.6"

func main() {
	var (
		demo    bool
		doctor  bool
		opaque  bool
		showVer bool
	)
	flag.BoolVar(&demo, "demo", false, "run with a virtual controller; nothing is written to hardware")
	flag.BoolVar(&doctor, "doctor", false, "check dependencies, permissions and hardware, then exit")
	flag.BoolVar(&opaque, "opaque", false, "paint a solid dark background instead of using the terminal background")
	flag.BoolVar(&showVer, "version", false, "print the version and exit")
	flag.Parse()

	if showVer {
		fmt.Printf("ds4-tui %s\n", version)
		return
	}
	if doctor {
		os.Exit(runDoctor(os.Stdout, version))
	}
	if os.Getenv("DS4_TUI_DEMO") == "1" {
		demo = true
	}
	tui.SetTransparent(!opaque)

	var svc ds4.Service
	if demo {
		svc = ds4.NewDemo()
	} else {
		svc = ds4.NewSystem()
	}

	p := tea.NewProgram(
		tui.New(svc),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ds4-tui: %v\n", err)
		os.Exit(1)
	}
}
