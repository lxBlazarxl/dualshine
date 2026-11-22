# LED support troubleshooting

DUALSHINE's current backend talks to the kernel's LED class through
`/sys/class/leds`. That interface is created by the kernel, not by DUALSHINE.
This guide helps classify the failure and points to the right fix.

## Quick diagnosis

```bash
ls -ld /sys/class/leds
ls /sys/class/leds/*:red 2>/dev/null
lsusb | grep -i sony
```

| Result | Meaning | Go to |
| --- | --- | --- |
| `/sys/class/leds` missing | The kernel has no LED class (or sysfs is not mounted) | [Section A](#a-sysclassleds-is-missing) |
| Class exists, no DS4 `*:red` entries, no Sony USB/HID device | Controller is not connected | Connect it, then recheck |
| Class exists, no DS4 `*:red` entries, Sony USB/HID device present | LED class is present, but `hid-sony` is missing or not bound | [Section B](#b-led-class-exists-but-there-are-no-ds4-entries) |
| DS4 entries exist, writing fails | Permissions issue | [Section C](#c-led-entries-exist-but-writing-is-denied) |
| Sony `/dev/hidrawN` exists, no LED entries | HID device is present but kernel LED interface is not | [Section D](#d-sony-hidraw-exists-but-no-led-class) |

You can also run:

```bash
./bin/ds4-tui --doctor
```

It distinguishes a missing LED class, a missing controller, and a missing
`hid-sony` driver.

## A. `/sys/class/leds` is missing

Loading `hid-sony` will not help here: the LED class itself is absent. Check the
kernel and sysfs first.

### 1. Is sysfs mounted?

```bash
mount | grep ' on /sys '
ls /sys/class
```

If `/sys` is missing or not a sysfs mount, mount it:

```bash
sudo mount -t sysfs sysfs /sys
```

In a container, you normally cannot mount or expose the host's LEDs from
inside. Run DUALSHINE on the host, or give the container access to the host
sysfs and devices. A stock container without host device access will not work.

### 2. Is `CONFIG_LEDS_CLASS` enabled?

```bash
config="/boot/config-$(uname -r)"
if [ -r "$config" ]; then
  grep -E 'CONFIG_LEDS_CLASS|CONFIG_HID_SONY' "$config"
elif [ -r /proc/config.gz ]; then
  zgrep -E 'CONFIG_LEDS_CLASS|CONFIG_HID_SONY' /proc/config.gz
else
  echo "kernel config not readable"
fi
```

You want:

```text
CONFIG_LEDS_CLASS=y   # or =m
CONFIG_HID_SONY=y     # or =m, for the DS4 LED entries
```

If `CONFIG_LEDS_CLASS` is `n` or absent, this kernel cannot expose
`/sys/class/leds`. Install or boot a standard distribution kernel.

### 3. Install or boot a distribution kernel

Package names vary by distribution. Examples:

```bash
# Arch
sudo pacman -Syu linux          # or linux-lts
# Debian / Ubuntu
sudo apt update && sudo apt install linux-generic
# Fedora
sudo dnf install kernel
# openSUSE
sudo zypper install kernel-default
```

Reboot into the new kernel, then check again:

```bash
uname -r
ls -ld /sys/class/leds
```

If you use a custom or minimal kernel, enable `CONFIG_LEDS_CLASS` and
`CONFIG_HID_SONY`, rebuild, and boot it.

### 4. Still missing after a kernel change?

Check:

```bash
dmesg | grep -iE 'leds|led class|sony'
cat /proc/filesystems | grep sysfs
```

On WSL2, the Linux kernel may lack the needed device/driver setup. USB
passthrough with `usbipd` can expose a DS4 to WSL, but Bluetooth passthrough
and LED class support vary. WSL is not officially supported.

### If you cannot change the kernel

The TUI still runs as a **preview/demo**:

```bash
./bin/ds4-tui --demo
```

The picker, presets, transparency, and preview work. Apply and Rainbow cannot
work on that kernel. A future direct HID backend could use `/dev/hidraw` and
avoid the LED class, but it is not implemented in this sysfs-backed version.

## B. LED class exists but there are no DS4 entries

This means the LED class is available but `hid-sony` is not bound to the
controller. Confirm the controller itself is connected first:

```bash
lsusb | grep -i sony
```

If that prints nothing, the controller is not connected; use USB or pair it
over Bluetooth. If it prints a Sony device but there are still no LED entries,
then continue with `hid-sony`:

```bash
ls /sys/class/leds
lsmod | grep hid_sony
modinfo hid_sony
sudo modprobe hid-sony
```

Then unplug and replug the controller (or reconnect Bluetooth) and check:

```bash
ls /sys/class/leds/*:red
dmesg | grep -i sony
```

If `modprobe` says the module is missing, install your distribution's kernel
modules/extra package, or boot a kernel built with `CONFIG_HID_SONY`.

If the module exists but does not bind:

```bash
lsusb | grep -i sony
ls /sys/bus/hid/drivers/
dmesg | grep -iE 'sony|hid'
```

Stop userspace tools that claim the controller, such as `ds4drv`, and replug.
Steam Input can also take control of the device; close Steam or disable Steam
Input for the controller and retry.

For Bluetooth, make sure BlueZ is running, remove and re-pair the controller,
and then check `dmesg` again:

```bash
bluetoothctl
# remove XX:XX:XX:XX:XX:XX
# scan on
# pair XX:XX:XX:XX:XX:XX
# connect XX:XX:XX:XX:XX:XX
```

USB is generally the easier path when diagnosing.

## C. LED entries exist but writing is denied

This is the normal root-owned sysfs case. Apply the one-time udev rule:

```bash
sudo ./scripts/install-udev.sh
```

Then press `r` in the TUI. Alternatively, let DUALSHINE ask
for sudo when applying.

## D. Sony hidraw exists but no LED class

Check:

```bash
ls /dev/hidraw*
./bin/ds4-tui --doctor
```

A `/dev/hidrawN` node means the kernel exposed a raw HID interface even though
the LED class entries are absent. The current backend does not use hidraw
directly, so this is **not enough** to make Apply/Rainbow work.

Today your options are:

1. Get `hid-sony` bound so LED class entries appear (Section B).
2. Run `--demo` for the UI/preview.
3. Wait for or contribute a direct HID backend.

Do not write arbitrary bytes to hidraw unless you know the DS4 output-report
format and the risks; a wrong report can affect the controller.

## Collect support information

If you need to ask for help, include:

```bash
uname -a
cat /etc/os-release
lsusb | grep -i sony
ls -ld /sys/class/leds
ls /sys/class/leds
lsmod | grep -E 'hid_sony|hid_generic|hid_playstation'
dmesg | grep -i sony
./bin/ds4-tui --doctor
```

Avoid posting serial numbers or other personal data.
