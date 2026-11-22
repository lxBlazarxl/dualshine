package ds4

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/blazar/ds4-tui/internal/color"
	"golang.org/x/sys/unix"
)

const (
	pidFile = "/tmp/ds4-led.pid"
)

var ledDir = "/sys/class/leds"

// System talks to the real controller through the ds4-led and ds4-rb scripts.
type System struct {
	ledPath     string
	rainbowPath string
}

// NewSystem locates the helper scripts. Missing scripts are not fatal: the UI
// reports them and the rest of the application still works.
func NewSystem() *System {
	return &System{
		ledPath:     findBinary("ds4-led"),
		rainbowPath: findBinary("ds4-rb"),
	}
}

func findBinary(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	fallbacks := []string{
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/usr/bin", name),
	}
	if home, err := os.UserHomeDir(); err == nil {
		fallbacks = append(fallbacks, filepath.Join(home, ".local", "bin", name))
	}
	for _, path := range fallbacks {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path
		}
	}
	return ""
}

// Detect scans sysfs and reports the first suitable lightbar.
func (s *System) Detect() Status {
	st := Status{
		LEDBinary:     s.ledPath,
		RainbowBinary: s.rainbowPath,
		CheckedAt:     time.Now(),
	}
	if st.LEDBinary == "" {
		st.LEDBinary = "built-in"
	}
	if st.RainbowBinary == "" {
		st.RainbowBinary = "built-in"
	}
	if info, err := os.Stat(ledDir); err == nil && info.IsDir() {
		st.LEDClass = true
	}

	st.ControllerSeen = sonyDS4Present()

	reds := ledFiles("red")
	greens := ledFiles("green")
	blues := ledFiles("blue")
	globals := ledFiles("global")

	// Prefer Sony devices, exactly like ds4-led. If none are found we still
	// keep the discovered entries so a device with an unusual sysfs path can
	// be used, but flag it as unverified.
	reds, redSony := preferSony(reds)
	greens, greenSony := preferSony(greens)
	blues, blueSony := preferSony(blues)
	globals, _ = preferSony(globals)
	st.Sony = redSony && greenSony && blueSony && len(reds) > 0 && len(greens) > 0 && len(blues) > 0

	if len(reds) == 0 || len(greens) == 0 || len(blues) == 0 {
		switch {
		case !st.LEDClass:
			st.Err = "kernel LED class /sys/class/leds is not available; CONFIG_LEDS_CLASS may be disabled"
		case st.ControllerSeen:
			st.Err = "DS4 is connected, but no LED entries were exposed; hid-sony is missing or not bound"
		default:
			st.Err = "no DualShock 4 detected; connect it over USB or Bluetooth"
		}
		st.KeeperPID = keeperPID()
		return st
	}

	st.Found = true
	st.RedPath = reds[0]
	st.GreenPath = greens[0]
	st.BluePath = blues[0]
	if len(globals) > 0 {
		st.GlobalPath = globals[0]
	}
	st.Writable = writable(st.RedPath) && writable(st.GreenPath) && writable(st.BluePath)
	st.KeeperPID = keeperPID()

	if r, ok := readLED(st.RedPath); ok {
		if g, ok := readLED(st.GreenPath); ok {
			if b, ok := readLED(st.BluePath); ok {
				st.Current = color.New(r, g, b)
				st.CurrentOK = true
			}
		}
	}
	return st
}

// Apply applies the chosen color to the controller. If ds4-led is installed it is used
// in persistent mode; otherwise the color is written directly to sysfs.
func (s *System) Apply(ctx context.Context, c color.RGBA) (string, error) {
	if s.ledPath != "" {
		cmd := s.ColorCommand(c)
		if cmd == nil {
			return "", errors.New("ds4-led was not found in PATH")
		}
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			if text == "" {
				text = err.Error()
			}
			return text, errors.New(text)
		}
		return text, nil
	}
	return s.applyNative(c)
}

func (s *System) applyNative(c color.RGBA) (string, error) {
	st := s.Detect()
	if !st.Found {
		return "", errors.New("no DualShock 4 lightbar found")
	}
	r, g, b := c.Effective()
	if st.GlobalPath != "" {
		_ = os.WriteFile(st.GlobalPath, []byte("1\n"), 0o644)
	}
	if err := os.WriteFile(st.RedPath, []byte(strconv.Itoa(int(r))+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("failed to write red LED: %w", err)
	}
	if err := os.WriteFile(st.GreenPath, []byte(strconv.Itoa(int(g))+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("failed to write green LED: %w", err)
	}
	if err := os.WriteFile(st.BluePath, []byte(strconv.Itoa(int(b))+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("failed to write blue LED: %w", err)
	}
	if keeperPID() > 0 {
		_ = s.stopKeeper(st.Writable)
	}
	return fmt.Sprintf("DualShock 4 LED set: %s (R:%d G:%d B:%d)", c.Hex(), r, g, b), nil
}

// ColorCommand returns the command used for an interactive apply.
// If ds4-led is found, it is invoked so it can prompt for sudo if necessary.
// If not found, a shell or sudo command is synthesized to write directly to sysfs.
func (s *System) ColorCommand(c color.RGBA) *exec.Cmd {
	if s.ledPath != "" {
		return exec.Command(s.ledPath,
			strconv.Itoa(int(c.R)),
			strconv.Itoa(int(c.G)),
			strconv.Itoa(int(c.B)),
			ledBrightnessArg(c.A),
		)
	}
	st := s.Detect()
	if !st.Found {
		return nil
	}
	r, g, b := c.Effective()
	script := fmt.Sprintf("if [ -n %q ] && [ -e %q ]; then printf '1\\n' > %q 2>/dev/null || true; fi; printf '%%s\\n' %d > %q && printf '%%s\\n' %d > %q && printf '%%s\\n' %d > %q",
		st.GlobalPath, st.GlobalPath, st.GlobalPath,
		r, st.RedPath,
		g, st.GreenPath,
		b, st.BluePath)

	if st.Writable {
		return exec.Command("/bin/sh", "-c", script)
	}
	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		return nil
	}
	return exec.Command(sudoPath, "/bin/sh", "-c", script)
}

// AuthCommand validates (and caches) sudo credentials interactively. The
// caller runs this through tea.ExecProcess so sudo can prompt on the real tty.
func (s *System) AuthCommand() *exec.Cmd {
	if _, err := exec.LookPath("sudo"); err != nil {
		return nil
	}
	return exec.Command("sudo", "-v")
}

// StartRainbow starts the rainbow animation. Writable LED files use the
// user's ds4-rb script directly if available, or the built-in native rainbow.
// Root-owned files use a small fixed helper run through one sudo authentication;
// that helper watches this process and a stop file, so it cannot outlive the TUI.
func (s *System) StartRainbow() (Rainbow, error) {
	st := s.Detect()
	if !st.Found {
		return nil, errors.New("no DualShock 4 lightbar found")
	}

	if keeperPID() > 0 {
		if err := s.stopKeeper(st.Writable); err != nil {
			return nil, err
		}
	}

	if st.Writable {
		if s.rainbowPath != "" {
			return s.startScriptRainbow()
		}
		return s.startNativeRainbow(st)
	}
	return s.startPrivilegedRainbow(st)
}

func (s *System) startScriptRainbow() (Rainbow, error) {
	cmd := exec.Command(s.rainbowPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	buf := &lockedBuffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start ds4-rb: %w", err)
	}

	p := &systemRainbow{cmd: cmd, out: buf, done: make(chan struct{})}
	go func() {
		p.waitErr = cmd.Wait()
		close(p.done)
	}()
	return p, nil
}

func (s *System) stopKeeper(writable bool) error {
	if s.ledPath != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var cmd *exec.Cmd
		if writable {
			cmd = exec.CommandContext(ctx, s.ledPath, "--stop")
		} else {
			sudoPath, err := exec.LookPath("sudo")
			if err != nil {
				return errors.New("sudo is required to stop the running color keeper")
			}
			cmd = exec.CommandContext(ctx, sudoPath, "-n", "--", s.ledPath, "--stop")
		}
		out, err := cmd.CombinedOutput()
		if keeperPID() == 0 {
			return nil
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if msg == "" {
			msg = "the keeper is still running"
		}
		return fmt.Errorf("could not stop the running ds4-led color keeper: %s", msg)
	}

	pid := keeperPID()
	if pid == 0 {
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err == nil {
		_ = os.Remove(pidFile)
		return nil
	}
	if !writable {
		if sudoPath, err := exec.LookPath("sudo"); err == nil {
			_ = exec.Command(sudoPath, "-n", "--", "kill", strconv.Itoa(pid)).Run()
			_ = exec.Command(sudoPath, "-n", "--", "rm", "-f", pidFile).Run()
		}
	}
	if keeperPID() == 0 {
		return nil
	}
	return errors.New("could not stop the running color keeper")
}

const privilegedRainbowScript = `
set -u
ppid=$1
stopfile=$2
red=$3
green=$4
blue=$5
global=$6

trap 'rm -f "$stopfile"' EXIT

alive() {
  [ ! -e "$stopfile" ] || exit 0
  kill -0 "$ppid" 2>/dev/null || exit 0
}

apply() {
  printf '%s\n' "$r" > "$red" || { printf 'failed to write %s\n' "$red" >&2; exit 1; }
  printf '%s\n' "$g" > "$green" || { printf 'failed to write %s\n' "$green" >&2; exit 1; }
  printf '%s\n' "$b" > "$blue" || { printf 'failed to write %s\n' "$blue" >&2; exit 1; }
  alive
}

tick() {
  sleep 0.03
  alive
}

alive
if [ -n "$global" ]; then
  printf '1\n' > "$global" 2>/dev/null || true
fi
r=255; g=0; b=0
apply
while :; do
  while [ "$g" -lt 255 ]; do g=$((g+5)); apply; tick; done
  while [ "$r" -gt 0 ]; do r=$((r-5)); apply; tick; done
  while [ "$b" -lt 255 ]; do b=$((b+5)); apply; tick; done
  while [ "$g" -gt 0 ]; do g=$((g-5)); apply; tick; done
  while [ "$r" -lt 255 ]; do r=$((r+5)); apply; tick; done
  while [ "$b" -gt 0 ]; do b=$((b-5)); apply; tick; done
done
`

func (s *System) startPrivilegedRainbow(st Status) (Rainbow, error) {
	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		return nil, errors.New("sudo was not found; rainbow needs writable LEDs or sudo")
	}
	bashPath := "/bin/bash"
	if _, err := os.Stat(bashPath); err != nil {
		found, lookErr := exec.LookPath("bash")
		if lookErr != nil {
			return nil, errors.New("bash was not found; cannot start the privileged rainbow helper")
		}
		bashPath = found
	}

	tmp, err := os.CreateTemp("", "ds4-tui-rainbow-*.stop")
	if err != nil {
		return nil, fmt.Errorf("could not allocate a rainbow control file: %w", err)
	}
	stopPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(stopPath)

	cmd := exec.Command(sudoPath, "-n", "--", bashPath, "-c", privilegedRainbowScript, "ds4-rainbow-helper",
		strconv.Itoa(os.Getpid()), stopPath,
		st.RedPath, st.GreenPath, st.BluePath, st.GlobalPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin = os.Stdin

	buf := &lockedBuffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start the privileged rainbow helper: %w", err)
	}

	p := &privilegedRainbow{
		cmd:      cmd,
		out:      buf,
		done:     make(chan struct{}),
		stopPath: stopPath,
	}
	go func() {
		p.waitErr = cmd.Wait()
		close(p.done)
	}()
	return p, nil
}

// ledBrightnessArg translates the picker's 0-255 alpha into ds4-led's slightly
// ambiguous brightness syntax. ds4-led treats values 0-100 as percentages and
// values 101-255 as raw 8-bit values, so alpha <= 100 must be sent as a
// percentage to keep brightness linear across the whole slider.
func ledBrightnessArg(a uint8) string {
	if a > 100 {
		return strconv.Itoa(int(a))
	}
	pct := int((uint32(a)*100 + 127) / 255)
	return strconv.Itoa(pct) + "%"
}

// sonyDS4Present checks the HID and USB device trees without requiring
// hid-sony or LED class entries. This lets the UI distinguish a disconnected
// controller from one that is present but has no LED interface.
func sonyDS4Present() bool {
	hidDevices, _ := filepath.Glob(filepath.Join("/sys/bus/hid/devices", "*"))
	for _, device := range hidDevices {
		name := strings.ToUpper(filepath.Base(device))
		if !strings.Contains(name, ":054C:") {
			continue
		}
		parts := strings.SplitN(name, ":", 3)
		if len(parts) != 3 {
			continue
		}
		product := strings.SplitN(parts[2], ".", 2)[0]
		switch product {
		case "05C4", "09CC": // known DualShock 4 product IDs
			return true
		}
	}

	// Some kernels expose the USB parent even if HID enumeration is delayed.
	usbDevices, _ := filepath.Glob(filepath.Join("/sys/bus/usb/devices", "*"))
	for _, device := range usbDevices {
		vendor, err := os.ReadFile(filepath.Join(device, "idVendor"))
		if err != nil || !strings.EqualFold(strings.TrimSpace(string(vendor)), "054c") {
			continue
		}
		product, err := os.ReadFile(filepath.Join(device, "idProduct"))
		if err != nil {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(string(product))) {
		case "05C4", "09CC":
			return true
		}
	}
	return false
}

func ledFiles(kind string) []string {
	matches, _ := filepath.Glob(filepath.Join(ledDir, "*:"+kind, "brightness"))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if st, err := os.Stat(m); err == nil && !st.IsDir() {
			out = append(out, m)
		}
	}
	return out
}

func preferSony(files []string) ([]string, bool) {
	var sony []string
	for _, f := range files {
		target, err := filepath.EvalSymlinks(f)
		if err != nil {
			target = f
		}
		if strings.Contains(strings.ToUpper(target), "054C") {
			sony = append(sony, f)
		}
	}
	if len(sony) > 0 {
		return sony, true
	}
	return files, false
}

func writable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

func readLED(path string) (uint8, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || v < 0 || v > 255 {
		return 0, false
	}
	return uint8(v), true
}

func keeperPID() int {
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return 0
	}
	if err := syscall.Kill(pid, 0); err == nil || errors.Is(err, syscall.EPERM) {
		return pid
	}
	return 0
}

type systemRainbow struct {
	cmd     *exec.Cmd
	out     *lockedBuffer
	done    chan struct{}
	waitErr error
}

func (p *systemRainbow) PID() int { return p.cmd.Process.Pid }

func (p *systemRainbow) Wait() error {
	<-p.done
	return p.waitErr
}

func (p *systemRainbow) Stop() error {
	return signalGroup(p.cmd, syscall.SIGINT)
}

func (p *systemRainbow) Kill() error {
	return signalGroup(p.cmd, syscall.SIGKILL)
}

func (p *systemRainbow) Output() string {
	return strings.TrimSpace(p.out.String())
}

type privilegedRainbow struct {
	cmd      *exec.Cmd
	out      *lockedBuffer
	done     chan struct{}
	waitErr  error
	stopPath string
	stopOnce sync.Once
}

func (p *privilegedRainbow) PID() int { return p.cmd.Process.Pid }

func (p *privilegedRainbow) Wait() error {
	<-p.done
	_ = os.Remove(p.stopPath)
	return p.waitErr
}

func (p *privilegedRainbow) Stop() error {
	p.stopOnce.Do(func() {
		_ = os.WriteFile(p.stopPath, []byte("stop\n"), 0o600)
		// If the process is already gone, the file is harmless; clean it up.
		go func() {
			time.Sleep(2 * time.Second)
			_ = os.Remove(p.stopPath)
		}()
	})
	return nil
}

func (p *privilegedRainbow) Kill() error { return p.Stop() }

func (p *privilegedRainbow) Output() string {
	return strings.TrimSpace(p.out.String())
}

type nativeRainbow struct {
	done     chan struct{}
	stop     chan struct{}
	stopOnce sync.Once
}

func (p *nativeRainbow) PID() int { return os.Getpid() }

func (p *nativeRainbow) Wait() error {
	<-p.done
	return nil
}

func (p *nativeRainbow) Stop() error {
	p.stopOnce.Do(func() { close(p.stop) })
	return nil
}

func (p *nativeRainbow) Kill() error { return p.Stop() }

func (p *nativeRainbow) Output() string { return "" }

func (s *System) startNativeRainbow(st Status) (Rainbow, error) {
	p := &nativeRainbow{
		done: make(chan struct{}),
		stop: make(chan struct{}),
	}
	go func() {
		defer close(p.done)
		if st.GlobalPath != "" {
			_ = os.WriteFile(st.GlobalPath, []byte("1\n"), 0o644)
		}
		apply := func(r, g, b int) bool {
			_ = os.WriteFile(st.RedPath, []byte(strconv.Itoa(r)+"\n"), 0o644)
			_ = os.WriteFile(st.GreenPath, []byte(strconv.Itoa(g)+"\n"), 0o644)
			_ = os.WriteFile(st.BluePath, []byte(strconv.Itoa(b)+"\n"), 0o644)
			select {
			case <-p.stop:
				return false
			default:
				return true
			}
		}
		tick := func() bool {
			select {
			case <-p.stop:
				return false
			case <-time.After(30 * time.Millisecond):
				return true
			}
		}

		r, g, b := 255, 0, 0
		if !apply(r, g, b) {
			return
		}
		for {
			for g < 255 {
				g += 5
				if !apply(r, g, b) || !tick() {
					return
				}
			}
			for r > 0 {
				r -= 5
				if !apply(r, g, b) || !tick() {
					return
				}
			}
			for b < 255 {
				b += 5
				if !apply(r, g, b) || !tick() {
					return
				}
			}
			for g > 0 {
				g -= 5
				if !apply(r, g, b) || !tick() {
					return
				}
			}
			for r < 255 {
				r += 5
				if !apply(r, g, b) || !tick() {
					return
				}
			}
			for b > 0 {
				b -= 5
				if !apply(r, g, b) || !tick() {
					return
				}
			}
		}
	}()
	return p, nil
}

func signalGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	err := syscall.Kill(-cmd.Process.Pid, sig)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	if err != nil {
		return cmd.Process.Signal(sig)
	}
	return nil
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}
