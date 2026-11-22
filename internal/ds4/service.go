package ds4

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/blazar/ds4-tui/internal/color"
)

// Status describes the controller, the LED files and the helper scripts.
type Status struct {
	Found      bool
	Sony       bool // the LED paths belong to a Sony device (VID 054C)
	Writable   bool
	Current    color.RGBA
	CurrentOK  bool
	KeeperPID  int
	RedPath    string
	GreenPath  string
	BluePath   string
	GlobalPath string

	LEDBinary     string
	RainbowBinary string
	Err           string
	CheckedAt     time.Time
}

// Readable returns whether a controller was found.
func (s Status) Readable() bool { return s.Found }

// Rainbow describes a running rainbow process.
type Rainbow interface {
	PID() int
	Stop() error
	Kill() error
	Wait() error
	Output() string
}

// Service is the bridge between the TUI and the DualShock 4 helpers. It is an
// interface so the UI can run in a safe demo mode without touching hardware.
type Service interface {
	Detect() Status
	Apply(ctx context.Context, c color.RGBA) (string, error)
	ColorCommand(c color.RGBA) *exec.Cmd
	AuthCommand() *exec.Cmd
	StartRainbow() (Rainbow, error)
}

// Demo is a virtual controller used by --demo and by UI tests.
type Demo struct {
	mu      sync.Mutex
	current color.RGBA
	rainbow *demoRainbow
}

// NewDemo returns a demo service with a connected virtual controller.
func NewDemo() *Demo {
	return &Demo{current: color.New(139, 92, 246)}
}

func (d *Demo) Detect() Status {
	d.mu.Lock()
	defer d.mu.Unlock()
	return Status{
		Found:         true,
		Sony:          true,
		Writable:      true,
		Current:       d.current,
		CurrentOK:     true,
		LEDBinary:     "demo",
		RainbowBinary: "demo",
		CheckedAt:     time.Now(),
	}
}

func (d *Demo) Apply(_ context.Context, c color.RGBA) (string, error) {
	d.mu.Lock()
	d.current = c
	d.mu.Unlock()
	return "Demo mode: color " + c.Hex() + " previewed without hardware", nil
}

func (d *Demo) ColorCommand(color.RGBA) *exec.Cmd {
	return exec.Command("true")
}

func (d *Demo) AuthCommand() *exec.Cmd { return nil }

func (d *Demo) StartRainbow() (Rainbow, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.rainbow != nil && !d.rainbow.closed() {
		return d.rainbow, nil
	}
	d.rainbow = newDemoRainbow()
	return d.rainbow, nil
}
