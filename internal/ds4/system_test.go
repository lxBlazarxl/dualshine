package ds4

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/blazar/ds4-tui/internal/color"
)

func TestLEDBrightnessArg(t *testing.T) {
	tests := []struct {
		alpha uint8
		want  string
	}{
		{0, "0%"},
		{1, "0%"},
		{50, "20%"},
		{100, "39%"},
		{101, "101"},
		{128, "128"},
		{200, "200"},
		{255, "255"},
	}
	for _, tt := range tests {
		if got := ledBrightnessArg(tt.alpha); got != tt.want {
			t.Errorf("ledBrightnessArg(%d) = %q, want %q", tt.alpha, got, tt.want)
		}
	}
}

func TestColorCommandArgs(t *testing.T) {
	s := &System{ledPath: "/usr/bin/true"}
	cmd := s.ColorCommand(color.RGBA{R: 1, G: 2, B: 3, A: 50})
	if cmd == nil {
		t.Fatal("nil command")
	}
	want := []string{"/usr/bin/true", "1", "2", "3", "20%"}
	if len(cmd.Args) != len(want) {
		t.Fatalf("args = %v", cmd.Args)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, cmd.Args[i], want[i])
		}
	}
	if (&System{}).ColorCommand(color.New(1, 2, 3)) != nil {
		t.Fatal("expected nil command without ds4-led")
	}
}

func TestApplyRunsScript(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := filepath.Join(dir, "ds4-led")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\necho 'DualShock 4 LED set: ok'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	s := &System{ledPath: script}
	out, err := s.Apply(context.Background(), color.RGBA{R: 10, G: 20, B: 30, A: 200})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "DualShock 4 LED set") {
		t.Fatalf("output = %q", out)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "10\n20\n30\n200\n"
	if string(got) != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestApplyReportsFailure(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "ds4-led")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'no controller' >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := &System{ledPath: script}
	out, err := s.Apply(context.Background(), color.New(1, 2, 3))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(out, "no controller") || !strings.Contains(err.Error(), "no controller") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestPreferSony(t *testing.T) {
	root := t.TempDir()
	mkLED := func(name string, sony bool) string {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(dir, "target")
		if err := os.WriteFile(target, []byte("0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "brightness")
		src := target
		if sony {
			sonyDir := filepath.Join(root, "0005:054C:05C4.0003:"+name)
			if err := os.MkdirAll(sonyDir, 0o755); err != nil {
				t.Fatal(err)
			}
			src = filepath.Join(sonyDir, "brightness")
			if err := os.WriteFile(src, []byte("0\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Symlink(src, link); err != nil {
			t.Fatal(err)
		}
		return link
	}

	other := mkLED("other:red", false)
	got, sony := preferSony([]string{other})
	if sony || len(got) != 1 || got[0] != other {
		t.Fatalf("non-sony fallback = %v, %v", got, sony)
	}

	sonyLED := mkLED("sony:red", true)
	got, sony = preferSony([]string{other, sonyLED})
	if !sony || len(got) != 1 || got[0] != sonyLED {
		t.Fatalf("sony preference = %v, %v", got, sony)
	}
}

func TestPrivilegedRainbowHelperStops(t *testing.T) {
	dir := t.TempDir()
	red := filepath.Join(dir, "red")
	green := filepath.Join(dir, "green")
	blue := filepath.Join(dir, "blue")
	global := filepath.Join(dir, "global")
	stop := filepath.Join(dir, "stop")
	for _, path := range []string{red, green, blue, global} {
		if err := os.WriteFile(path, []byte("0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("/bin/bash", "-c", privilegedRainbowScript, "ds4-rainbow-test",
		strconv.Itoa(os.Getpid()), stop, red, green, blue, global)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// The first frame is written immediately.
	deadline := time.Now().Add(2 * time.Second)
	var got string
	for time.Now().Before(deadline) {
		b, _ := os.ReadFile(red)
		got = strings.TrimSpace(string(b))
		if got != "" && got != "0" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got == "" || got == "0" {
		_ = cmd.Process.Kill()
		t.Fatalf("helper never wrote a frame; output=%q", output.String())
	}
	if b, _ := os.ReadFile(global); strings.TrimSpace(string(b)) != "1" {
		_ = cmd.Process.Kill()
		t.Fatalf("global was not enabled: %q", b)
	}

	if err := os.WriteFile(stop, []byte("stop\n"), 0o600); err != nil {
		_ = cmd.Process.Kill()
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("helper exit: %v; output=%q", err, output.String())
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("helper did not exit after the stop file was created")
	}
}

func TestStartPrivilegedRainbowWithFakeSudo(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeSudo := filepath.Join(bin, "sudo")
	if err := os.WriteFile(fakeSudo, []byte("#!/bin/sh\nif [ \"$1\" = \"-n\" ]; then shift; fi\nif [ \"$1\" = \"--\" ]; then shift; fi\nexec \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))

	red := filepath.Join(dir, "red")
	green := filepath.Join(dir, "green")
	blue := filepath.Join(dir, "blue")
	global := filepath.Join(dir, "global")
	for _, path := range []string{red, green, blue, global} {
		if err := os.WriteFile(path, []byte("0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := &System{}
	p, err := s.startPrivilegedRainbow(Status{
		Found: true, RedPath: red, GreenPath: green, BluePath: blue, GlobalPath: global,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.PID() <= 0 {
		t.Fatal("invalid PID")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		b, _ := os.ReadFile(red)
		if strings.TrimSpace(string(b)) != "" && strings.TrimSpace(string(b)) != "0" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if b, _ := os.ReadFile(red); strings.TrimSpace(string(b)) == "0" {
		t.Fatalf("helper did not write: %q", b)
	}
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- p.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("wait: %v; output=%q", err, p.Output())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("privileged rainbow did not stop")
	}
}

func TestNativeApplyDirectWrites(t *testing.T) {
	origLedDir := ledDir
	defer func() { ledDir = origLedDir }()

	tempDir := t.TempDir()
	ledDir = tempDir

	mkMockLED := func(name string) string {
		p := filepath.Join(tempDir, "mock:"+name)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		f := filepath.Join(p, "brightness")
		if err := os.WriteFile(f, []byte("0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return f
	}

	red := mkMockLED("red")
	green := mkMockLED("green")
	blue := mkMockLED("blue")
	global := mkMockLED("global")

	s := &System{} // No ledPath set
	out, err := s.Apply(context.Background(), color.RGBA{R: 200, G: 100, B: 50, A: 255})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if !strings.Contains(out, "DualShock 4 LED set") {
		t.Fatalf("unexpected output: %q", out)
	}

	readVal := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(b))
	}

	if got := readVal(red); got != "200" {
		t.Errorf("red = %q, want 200", got)
	}
	if got := readVal(green); got != "100" {
		t.Errorf("green = %q, want 100", got)
	}
	if got := readVal(blue); got != "50" {
		t.Errorf("blue = %q, want 50", got)
	}
	if got := readVal(global); got != "1" {
		t.Errorf("global = %q, want 1", got)
	}
}

func TestStartNativeRainbow(t *testing.T) {
	dir := t.TempDir()
	red := filepath.Join(dir, "red")
	green := filepath.Join(dir, "green")
	blue := filepath.Join(dir, "blue")
	global := filepath.Join(dir, "global")
	for _, p := range []string{red, green, blue, global} {
		if err := os.WriteFile(p, []byte("0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := &System{}
	st := Status{
		Found:      true,
		Writable:   true,
		RedPath:    red,
		GreenPath:  green,
		BluePath:   blue,
		GlobalPath: global,
	}

	rb, err := s.startNativeRainbow(st)
	if err != nil {
		t.Fatalf("startNativeRainbow error: %v", err)
	}
	if rb.PID() <= 0 {
		t.Fatalf("invalid PID: %d", rb.PID())
	}

	// Verify it wrote the initial frame
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		b, _ := os.ReadFile(red)
		if strings.TrimSpace(string(b)) == "255" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if b, _ := os.ReadFile(red); strings.TrimSpace(string(b)) != "255" {
		t.Fatalf("initial red frame not written: %s", string(b))
	}

	if err := rb.Stop(); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- rb.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("wait error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("native rainbow did not stop in time")
	}
}
