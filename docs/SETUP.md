# Setup and run

DUALSHINE is a Linux terminal UI for the DualShock 4 lightbar. It drives the
controller through the existing `ds4-led` and `ds4-rb` helper scripts rather
than writing to sysfs itself.

This guide assumes you are starting from a fresh shell.

## 1. Prerequisites

- **Linux** with a DualShock 4 connected over USB or Bluetooth.
  This project is Linux-only: it uses `/sys/class/leds`, `sudo`, and the
  `ds4-led`/`ds4-rb` shell scripts. Windows and macOS are not supported
  natively. WSL2 may work only with USB passthrough and is not officially
  supported.
  The kernel must have `CONFIG_LEDS_CLASS` and `CONFIG_HID_SONY` (module or
  built-in), and the controller must be bound to `hid-sony`. Standard desktop
  distros usually have these; minimal/embedded/container kernels may not.
- **Go 1.24 or newer** (`go version`).
- **GNU make** (optional, but used by the examples below).
- **`ds4-led` and `ds4-rb`** available on `PATH`.
  These helper scripts are not bundled with DUALSHINE. You need the versions
  already used on this machine, or equivalent scripts with the same interface:

  ```text
  ds4-led <red> <green> <blue> [brightness]
  ds4-led --stop
  ds4-rb
  ```

  `ds4-led` may elevate itself with `sudo`; `ds4-rb` must be run by a user that
  can write the LED files.
- **`sudo`** if the LED files are root-owned and you have not installed the
  optional udev rule yet.
- A **truecolor terminal** is strongly recommended. The UI degrades to a
  readable monochrome layout when `NO_COLOR` or `TERM=dumb` is set.
- If you use the default transparent chrome, a dark terminal background is
  recommended. Use `--opaque` on a light or very busy background.

Verify the tools:

```bash
go version
make --version
command -v ds4-led
command -v ds4-rb
```

If a helper is missing, put it in `~/.local/bin` (and make sure that directory
is on `PATH`) or in `/usr/local/bin`.

## 2. Dependencies at a glance

The full inventory is in the [README](../README.md#dependencies). In short:

- **Go modules:** Bubble Tea, Lip Gloss, `x/ansi`, `termenv`, and `x/sys`,
  plus their 17 transitive modules. Versions are pinned in `go.mod` and
  `go.sum`.
- **External commands:** `ds4-led` (required), `ds4-rb` (writable-LED rainbow),
  `sudo` + `bash` (read-only fallback), and optionally `udevadm`.
- **System:** Linux LED sysfs, a DualShock 4, and a terminal. No database,
  network service, Python, or Node.js is involved.

Go downloads the module dependencies automatically on the first build. To
pre-fetch or verify them without building:

```bash
go mod download
go mod verify
```

## 3. Build

From the project directory:

```bash
cd /path/to/dualshine
make build
./bin/ds4-tui --version
```

Without `make`:

```bash
mkdir -p bin
go build -trimpath -ldflags "-s -w" -o bin/ds4-tui ./cmd/ds4-tui
```

Go downloads the Bubble Tea/Lip Gloss dependencies during the first build.

## 4. Check your setup

Run the dependency and hardware preflight:

```bash
./bin/ds4-tui --doctor
# or, from source:
make doctor
```

It checks the platform, kernel LED class, `hid-sony`, Sony hidraw devices,
Go/make (for developers), `ds4-led`, `ds4-rb`, the controller, LED
permissions, `sudo`, `bash`, `udevadm`, the udev rule, TTY support, and color
capability.

- Exit code `0`: no hard blockers; warnings are allowed.
- Exit code `1`: a required dependency or platform requirement is missing.

The doctor never prompts for a password and never writes to the controller.
It is safe to run before the first launch.

## 5. Try demo mode first

Demo mode needs no controller and writes nothing to hardware:

```bash
./bin/ds4-tui --demo
```

Move around with `c`, `p`, `w`, the arrow keys, `tab`, and `?`. Quit with `q`.

## 6. Run with the controller

Connect the controller, then:

```bash
./bin/ds4-tui
```

What to check:

1. The top-right badge should say `DS4 READY` when the LED files are writable,
   or `DS4 · NEEDS SUDO` when they are root-owned.
2. If you connected the controller after launching, press `r` to rescan.
3. Move the picker and press `a` or `enter` to apply the live preview.
4. Press `w`, then `space`, for rainbow mode.

### Read-only LEDs

When the badge says `NEEDS SUDO`, applying a static color temporarily hands the
terminal to `ds4-led`, which can ask for your password. Rainbow mode asks for
`sudo` once and then runs a privileged helper in the background.

You can keep using that, or remove the prompts permanently in the next step.

### One-time udev setup (recommended)

This grants your account's primary group write access to Sony (`054c`) LED
brightness files:

```bash
sudo ./scripts/install-udev.sh
# or
make setup-led
```

The script installs
`/etc/udev/rules.d/99-dualshock4-led.rules`, fixes the currently connected
controller immediately, and reloads udev. Press `r` in the TUI afterwards; the
badge should become `DS4 READY`.

To undo the rule:

```bash
sudo rm /etc/udev/rules.d/99-dualshock4-led.rules
```

> Do not retry a wrong `sudo` password repeatedly. Many systems lock the
> account after 3 failures for several minutes. `sudo` does not echo
> passwords. If you get locked out, inspect with
> `faillock --user "$USER"` and clear it with
> `faillock --user "$USER" --reset` (or wait for the lockout to expire).

## 7. Install the binary

Install into `~/.local/bin`:

```bash
make install
```

Make sure `~/.local/bin` is on `PATH`, then run:

```bash
ds4-tui
```

To install elsewhere:

```bash
sudo make install PREFIX=/usr/local
```

Both `make install` and `make uninstall` respect `PREFIX` and `DESTDIR`.
The prebuilt `bin/ds4-tui` targets Linux x86-64/glibc; build from source for
another architecture or libc. The executable embeds its Go dependencies, so no
Go toolchain is needed to run it.

## 8. Options

```text
--demo      Virtual controller; no hardware writes.
--doctor    Dependency/hardware preflight, then exit.
--opaque    Solid dark background instead of transparent terminal background.
--version   Print the version.
--help      Print the flag summary.
```

Environment:

- `DS4_TUI_DEMO=1` is equivalent to `--demo`.
- `NO_COLOR` / `TERM=dumb` selects the monochrome layout.

## 9. Behavior notes

- Applying a static color uses `ds4-led` in its persistent mode. The color
  keeps being reasserted after you quit. Stop it with:

  ```bash
  ds4-led --stop
  ```

- The TUI maps its 0–255 alpha/brightness value to `ds4-led`'s ambiguous
  brightness syntax (`0–100` means percent, `101–255` means raw). For example,
  `A 50` is sent as `20%`.

- Rainbow mode stops a competing `ds4-led` keeper before starting. Quitting the
  TUI stops the rainbow as well.

- Transparency only affects the chrome: the base, cards and padding. Color
  swatches, the picker, sliders and the preview must remain colored.

## 10. Troubleshooting

| Symptom | Fix |
| --- | --- |
| Not sure what is missing | Run `./bin/ds4-tui --doctor` (or `make doctor`). |
| `NO DS4` badge | Connect the controller and press `r`. Check `ls /sys/class/leds/`. |
| `/sys/class/leds` missing | Kernel lacks `CONFIG_LEDS_CLASS` or sysfs is not mounted; this backend cannot work, but preview/demo still runs. |
| No DS4 `*:red`/`*:green`/`*:blue` entries | `hid-sony` is missing or not bound. Check `lsmod`, try `sudo modprobe hid-sony`, and inspect `dmesg`. |
| Sony hidraw exists, no LED class | The current sysfs backend cannot use hidraw. Enable `hid-sony` or run `--demo`; direct HID output is not implemented. |
| `ds4-led was not found in PATH` | Install it in `~/.local/bin` or `/usr/local/bin`. |
| `ds4-rb was not found in PATH` | Install it there too, or install the udev rule and use the writable path. |
| `Read-only LEDs` / password on every apply | Run `sudo ./scripts/install-udev.sh`, then press `r`. |
| `sudo authentication failed or was cancelled` | Try once more and type the password carefully (it is not echoed), or use the udev setup. |
| `could not stop the running ds4-led color keeper` | Run `sudo ds4-led --stop`, then start the rainbow again. |
| Account locked after sudo attempts | Run `faillock --user "$USER"` then `faillock --user "$USER" --reset`, or wait for the lockout timeout. |
| Colors look flat | Check `NO_COLOR`; use a truecolor terminal. Monochrome mode is intentional otherwise. |
| Windows / macOS | Not supported natively; the hardware backend is Linux-only. |
| Text hard to read | Use `--opaque`, or a darker less busy terminal background. |
| Terminal too small | The minimum full layout is 36×13; below that a size notice is shown. |
| Stuck TUI | From another terminal: `pkill -u "$USER" -f './bin/ds4-tui'` then `stty sane`. |

## 11. Uninstall

```bash
make uninstall
# if you installed the udev rule:
sudo rm /etc/udev/rules.d/99-dualshock4-led.rules
```

No other state is written by DUALSHINE beyond the udev rule and the
controller's current LED color.
