package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blazar/ds4-tui/internal/ds4"
	"golang.org/x/sys/unix"
)

type checkLevel int

const (
	checkOK checkLevel = iota
	checkWarn
	checkFail
	checkInfo
)

func (l checkLevel) label() string {
	switch l {
	case checkOK:
		return "OK"
	case checkWarn:
		return "WARN"
	case checkFail:
		return "FAIL"
	default:
		return "INFO"
	}
}

type doctorCheck struct {
	level  checkLevel
	name   string
	detail string
	hint   string
}

type doctorReport struct {
	checks []doctorCheck
}

func (r *doctorReport) add(level checkLevel, name, detail, hint string) {
	r.checks = append(r.checks, doctorCheck{
		level: level, name: name, detail: detail, hint: hint,
	})
}

func (r doctorReport) exitCode() int {
	for _, c := range r.checks {
		if c.level == checkFail {
			return 1
		}
	}
	return 0
}

func (r doctorReport) render(w io.Writer, version string) {
	fmt.Fprintf(w, "DUALSHINE doctor %s\n", version)
	fmt.Fprintf(w, "%s/%s · tools=%s\n\n", runtime.GOOS, runtime.GOARCH, runtime.Version())

	ok, warnings, failures := 0, 0, 0
	for _, c := range r.checks {
		switch c.level {
		case checkOK:
			ok++
		case checkWarn:
			warnings++
		case checkFail:
			failures++
		}
		fmt.Fprintf(w, "  %-4s %-18s %s\n", c.level.label(), c.name, c.detail)
		if c.hint != "" {
			fmt.Fprintf(w, "       %s\n", c.hint)
		}
	}

	fmt.Fprintf(w, "\nSummary: %d OK, %d warning(s), %d error(s)\n", ok, warnings, failures)
	switch {
	case failures > 0:
		fmt.Fprintln(w, "Result: missing required dependencies or an unsupported platform.")
	case warnings > 0:
		fmt.Fprintln(w, "Result: usable, with warnings. Demo mode works even without a controller.")
	default:
		fmt.Fprintln(w, "Result: all checks passed.")
	}
}

func runDoctor(w io.Writer, version string) int {
	r := doctorReport{}

	// Platform.
	if runtime.GOOS == "linux" {
		r.add(checkOK, "platform", "Linux", "")
	} else {
		r.add(checkFail, "platform", runtime.GOOS,
			"DUALSHINE is Linux-only because it uses /sys/class/leds.")
	}

	// Kernel support for the sysfs LED backend.
	if _, err := os.Stat("/sys/class/leds"); err != nil {
		r.add(checkFail, "kernel LED class", "/sys/class/leds is missing",
			"Check the sysfs mount and CONFIG_LEDS_CLASS, or boot a distro kernel. See docs/LED-SUPPORT.md.")
	} else {
		r.add(checkOK, "kernel LED class", "/sys/class/leds", "")
	}
	hidLoaded := false
	for _, p := range []string{"/sys/module/hid_sony", "/sys/bus/hid/drivers/hid-sony"} {
		if _, err := os.Stat(p); err == nil {
			hidLoaded = true
			break
		}
	}
	switch {
	case hidLoaded:
		r.add(checkOK, "hid-sony", "driver loaded/registered", "")
	default:
		if _, err := exec.LookPath("modinfo"); err != nil {
			r.add(checkInfo, "hid-sony", "state unknown (modinfo not found)",
				"Check 'lsmod | grep hid_sony' or load the module manually.")
		} else if exec.Command("modinfo", "hid_sony").Run() == nil {
			r.add(checkInfo, "hid-sony", "module available but not loaded",
				"It normally loads when the controller connects. If not: sudo modprobe hid-sony.")
		} else {
			r.add(checkWarn, "hid-sony", "not available in this kernel",
				"Kernel may lack CONFIG_HID_SONY; the DS4 may not expose LED class entries.")
		}
	}

	// Build tools are optional at runtime but useful for developers.
	if p, err := exec.LookPath("go"); err == nil {
		r.add(checkOK, "go toolchain", p, "")
	} else {
		r.add(checkWarn, "go toolchain", "not found",
			"Build-only dependency. Install Go 1.24+ to build or develop.")
	}
	if p, err := exec.LookPath("make"); err == nil {
		r.add(checkOK, "make", p, "")
	} else {
		r.add(checkInfo, "make", "not found",
			"Optional. Use 'go build'/'go test' directly if you prefer.")
	}

	svc := ds4.NewSystem()
	st := svc.Detect()

	// External helpers.
	if st.LEDBinary != "" {
		r.add(checkOK, "ds4-led", st.LEDBinary, "")
	} else {
		r.add(checkFail, "ds4-led", "not found on PATH",
			"Install it in ~/.local/bin or /usr/local/bin; static color apply needs it.")
	}
	if st.RainbowBinary != "" {
		r.add(checkOK, "ds4-rb", st.RainbowBinary, "")
	} else {
		r.add(checkWarn, "ds4-rb", "not found on PATH",
			"Needed for the rainbow when the LED files are writable. Get it in PATH.")
	}

	// Controller and LED access.
	if !st.Found {
		detail := "no DualShock 4 lightbar found"
		if st.Err != "" {
			detail = st.Err
		}
		r.add(checkWarn, "controller", detail,
			"Connect the DS4, check hid-sony, and press r. See docs/LED-SUPPORT.md; demo mode still works.")
	} else {
		detail := "DS4 lightbar found"
		if !st.Sony {
			detail += " (not verified as Sony VID 054c)"
		}
		r.add(checkOK, "controller", detail, "")
		if st.CurrentOK {
			r.add(checkInfo, "current color", st.Current.HexA(), "")
		}
		if st.Writable {
			r.add(checkOK, "LED access", "writable", "")
		} else {
			r.add(checkWarn, "LED access", "read-only (root-owned)",
				"Apply will ask for sudo. Run 'sudo ./scripts/install-udev.sh' once, then press r.")
		}
		if st.KeeperPID > 0 {
			r.add(checkInfo, "color keeper", fmt.Sprintf("running as PID %d", st.KeeperPID), "")
		}
	}

	// If the kernel does not expose LED class devices, a Sony HID interface
	// may still be present as /dev/hidrawN. The current backend cannot use it,
	// but knowing that distinguishes a missing controller from a missing
	// kernel driver.
	if !st.Found {
		hidraws := sonyHIDRawDevices()
		if len(hidraws) > 0 {
			detail := "Sony HID device present: " + strings.Join(hidraws, ", ")
			r.add(checkWarn, "hidraw", detail,
				"The kernel is not exposing it through the LED class. Enable hid-sony or use --demo; see docs/LED-SUPPORT.md.")
		} else {
			r.add(checkInfo, "hidraw", "no Sony HID device detected", "")
		}
	}

	// Privilege and helper requirements for the read-only path.
	sudoPath, sudoErr := exec.LookPath("sudo")
	if sudoErr == nil {
		r.add(checkOK, "sudo", sudoPath, "")
	} else if st.Found && !st.Writable {
		r.add(checkFail, "sudo", "not found",
			"Root-owned LEDs cannot be elevated. Install sudo or make the LEDs writable.")
	} else {
		r.add(checkInfo, "sudo", "not found", "Only needed for read-only LEDs.")
	}

	if p, err := exec.LookPath("bash"); err == nil {
		r.add(checkOK, "bash", p, "")
	} else {
		r.add(checkWarn, "bash", "not found",
			"Needed for the privileged rainbow fallback when the LEDs are read-only.")
	}

	if p, err := exec.LookPath("udevadm"); err == nil {
		r.add(checkOK, "udevadm", p, "")
	} else {
		r.add(checkInfo, "udevadm", "not found",
			"Optional. Needed only for the one-time LED permission setup.")
	}

	const udevRule = "/etc/udev/rules.d/99-dualshock4-led.rules"
	if _, err := os.Stat(udevRule); err == nil {
		r.add(checkOK, "udev rule", udevRule, "")
	} else if st.Found && !st.Writable {
		r.add(checkWarn, "udev rule", "not installed",
			"Run 'sudo ./scripts/install-udev.sh' to remove the sudo prompt.")
	} else {
		r.add(checkInfo, "udev rule", "not installed", "Optional.")
	}

	// Terminal and color support.
	stdinTTY := isTerminal(os.Stdin)
	stdoutTTY := isTerminal(os.Stdout)
	switch {
	case stdinTTY && stdoutTTY:
		r.add(checkOK, "terminal", "stdin and stdout are TTYs", "")
	case !stdinTTY && !stdoutTTY:
		r.add(checkWarn, "terminal", "stdin and stdout are not TTYs",
			"The TUI needs an interactive terminal. This doctor output works when piped.")
	case !stdinTTY:
		r.add(checkWarn, "terminal", "stdin is not a TTY",
			"The TUI needs interactive stdin. This doctor output works when piped.")
	default:
		r.add(checkWarn, "terminal", "stdout is not a TTY",
			"The TUI needs interactive stdout. This doctor output works when piped.")
	}

	switch {
	case os.Getenv("NO_COLOR") != "":
		r.add(checkInfo, "color", "NO_COLOR is set; monochrome mode", "")
	case strings.EqualFold(os.Getenv("TERM"), "dumb"):
		r.add(checkInfo, "color", "TERM=dumb; monochrome mode", "")
	case strings.Contains(strings.ToLower(os.Getenv("COLORTERM")), "truecolor"),
		strings.Contains(strings.ToLower(os.Getenv("COLORTERM")), "24bit"):
		r.add(checkOK, "color", "truecolor terminal", "")
	default:
		r.add(checkWarn, "color", "truecolor not detected",
			"Gradients and the preview may look flat. A truecolor terminal is recommended.")
	}

	r.render(w, version)
	return r.exitCode()
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TCGETS)
	return err == nil
}

func sonyHIDRawDevices() []string {
	dirs, _ := filepath.Glob("/sys/class/hidraw/hidraw*")
	var out []string
	for _, dir := range dirs {
		uevent, err := os.ReadFile(filepath.Join(dir, "device", "uevent"))
		if err != nil {
			continue
		}
		if !strings.Contains(strings.ToUpper(string(uevent)), "0000054C") {
			continue
		}
		dev := "/dev/" + filepath.Base(dir)
		if _, err := os.Stat(dev); err != nil {
			continue
		}
		out = append(out, dev)
	}
	return out
}
