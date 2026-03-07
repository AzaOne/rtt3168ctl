# rtt3168ctl

CLI utility for controlling a USB mouse based on the **RTT3168CG2** chip via vendor control transfers.

Supported features:
- read current mouse status;
- configure DPI profiles;
- switch active DPI slot;
- configure RGB mode and animation speed;
- change polling rate;
- bind an action to the CPI button;
- raw register dump/write (advanced).

## Requirements

- Go 1.25+
- Linux
- USB device access (root permissions or udev rules may be required)

`rtt3168ctl` uses `gousb`, which depends on `libusb-1.0`.

Install `libusb`:

```bash
# Debian / Ubuntu
sudo apt update
sudo apt install -y libusb-1.0-0-dev

# Arch Linux
sudo pacman -S libusb
```

## Build

```bash
make build
```

The binary will be available at `build/rtt3168ctl`.

## Modes (`-mode`)

- `read` - read current configuration
- `apply` - apply one or more settings
- `dump` - dump bank 1 registers (1..30)
- `write` - raw register write (advanced)

## Main Parameters

- `-slot` - slot used by `-dpi` and `-switch-slot` (`1-4`)
- `-dpi` - single-slot DPI value as `DPI` or `DPI:color` (e.g. `800` or `800:1`)
- `-color` - with `-dpi`: color index (`0-15`), `-1` = keep current
- `-switch-slot` - with `-dpi/-slot`: activate this slot after write
- `-dpi1..-dpi4` - slot value as `DPI` or `DPI:color` (e.g. `800` or `800:3`)
- `-color1..-color4` - color for each slot (`0..15`), `-1` = keep current
- `-active-slot` - activate slot (`1-4`) after applying settings
- `-speed` - RGB speed (`0-255`), `-1` = keep current
- `-json` - JSON output for `-mode read`
- `-reg`, `-regval` - raw values for `write`
- `-rate` - polling rate (`125/250/500/1000`)
- `-rgb-mode` - RGB mode value
- `-cpi-action` - CPI action value

### `-mode read`
Read current device settings.

Human-readable output:
```bash
./build/rtt3168ctl -mode read
```

JSON output:
```bash
./build/rtt3168ctl -mode read -json
```

### `-mode apply`
Apply several settings in one run, for example from cron/startup scripts:

Single-slot shortcut:
```bash
./build/rtt3168ctl -mode apply -dpi 800:1 -slot 1 -switch-slot
```

Full profile:
```bash
./build/rtt3168ctl -mode apply \
  -dpi1 800:3 \
  -dpi2 1600:0 \
  -dpi3 1600:7 \
  -dpi4 2000:9 \
  -active-slot 2 \
  -rate 1000 \
  -rgb-mode breath -speed 40 \
  -cpi-action vol_up
```

Supported `-rgb-mode` values:
- `off`
- `on`
- `breath`
- `breath_segment`
- `cycle6`
- `cycle12`
- `cycle768`

You can still use `-color1..-color4` separately; if both are set, values must match.

### `-mode dump`
Dump raw register values from bank 1 (registers `1..30`):

```bash
./build/rtt3168ctl -mode dump
```

### `-mode write`
Write a raw byte to a register (advanced diagnostics):

```bash
./build/rtt3168ctl -mode write -reg 14 -regval 2
```

## Device IDs and udev Rules

Defaults:
- `VID = 0x093A`
- `PID = 0x2533`

Override example:

```bash
MOUSE_VID=0x093A MOUSE_PID=0x2533 ./build/rtt3168ctl -mode read
```

Example udev rules (`52-rtt3168ctl-093a-2533.rules`):

```udev
SUBSYSTEM=="usb", ATTRS{idVendor}=="093a", ATTRS{idProduct}=="2533", MODE="0666"
KERNEL=="hidraw*", ATTRS{busnum}=="1", ATTRS{idVendor}=="093a", ATTRS{idProduct}=="2533", MODE="0666"
```

Apply the rule:

```bash
sudo udevadm control --reload-rules
sudo udevadm trigger --subsystem-match=usb
sudo udevadm trigger --subsystem-match=hidraw
```

Replug the mouse (or reboot).

## Useful Make Commands

```bash
make help
make build
make run ARGS='-mode read'
make test
make fmt
make vet
make clean
```

## Note

`write` and `dump` are intended for low-level diagnostics. Invalid register values may cause unpredictable device behavior.
