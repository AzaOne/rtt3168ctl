# rtt3168ctl

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI utility for controlling a USB mouse based on the **RTT3168CG2** chip via vendor control transfers. This project provides a native command-line interface to read device status, configure DPI and RGB lighting, and perform advanced register-level diagnostics.

## Features

- **Status Reading:** Instantly read the current configuration and status of your mouse.
- **DPI Management:** Configure single or multiple DPI profiles and seamlessly switch the active DPI slot.
- **RGB Customization:** Change RGB lighting modes, pick specific colors, and adjust animation speeds.
- **Performance Tuning:** Change the mouse polling rate (`125`, `250`, `500`, or `1000` Hz).
- **Button Binding:** Bind custom actions to the CPI button.
- **Advanced Diagnostics:** Dump raw unique registers from banks or write raw byte values directly to registers.
- **Experimental Mode:** Stream inferred runtime/event registers in a loop for reverse-engineering and monitoring.

## Getting Started

### Prerequisites

- **OS:** Linux
- **Go:** Version 1.25+
- **Permissions:** USB device access (root permissions or udev rules may be required).

### Installation (Dependencies)

`rtt3168ctl` uses the `gousb` package, which requires `libusb-1.0` to be installed on your system.

**Debian / Ubuntu:**
```bash
sudo apt update
sudo apt install -y libusb-1.0-0-dev
```

**Arch Linux:**
```bash
sudo pacman -S libusb
```

### Install the CLI

From the repository, run as your regular user:

```bash
./build.sh
./install.sh --user
```

This builds the native binary and installs it in `~/.local/bin`. Add that directory to your `PATH` to run `rtt3168ctl` from any directory. Rerun both commands after updating the source.

## Usage

### Check the version

Run `rtt3168ctl --version` to print the build version without opening the mouse.

Use one dash for a one-letter option (`-h`) and two for a long option (`--help`, `--mode read`, `--version`). When updating an older script, change long options such as `-mode read -json` to `--mode read --json`.

### Main Parameters

You can pass various flags to customize the command behavior:

- **Slots & DPI:**
  - `--slot` - slot used by `--dpi` and `--switch-slot` (`1-4`)
  - `--dpi` - single-slot DPI value as `DPI` or `DPI:color` (e.g., `800` or `800:1`)
  - `--color` - with `--dpi`: color index (`0-15`), `-1` = keep current
  - `--switch-slot` - with `--dpi/--slot`: activate this slot after write
  - `--dpi1..--dpi4` - slot value as `DPI` or `DPI:color` (e.g., `800` or `800:3`)
  - `--color1..--color4` - color for each slot (`0-15`), `-1` = keep current
  - `--active-slot` - activate slot (`1-4`) after applying settings
- **RGB & Performance:**
  - `--speed` - RGB speed (`0-255`), `-1` = keep current
  - `--rgb-mode` - RGB mode value (`off`, `on`, `breath`, `breath_segment`, `cycle6`, `cycle12`, `cycle768`)
  - `--rate` - polling rate (`125/250/500/1000`)
  - `--cpi-action` - CPI action value
- **Advanced / Output:**
  - `--json` - JSON output for `--mode read`
  - `--reg`, `--regval` - raw values for `write`
  - `--dump-banks` - banks to read in `dump` mode (`0,1`, `0-7`, or `all`)
  - `--exp-interval-ms` - poll interval for `experimental` mode (`>0`, default `20`)
  - `--exp-count` - number of printed samples in `experimental` mode (`0` = infinite)
  - `--exp-all` - print every sample in `experimental` mode (default: only changes)

---

### Operating Modes (`--mode`)

#### Read Settings (`--mode read`)
Read the current device settings.

Human-readable output:
```bash
./build/rtt3168ctl --mode read
```

JSON output:
```bash
./build/rtt3168ctl --mode read --json
```

#### Apply Settings (`--mode apply`)
Apply several settings in one run. Ideal for cron jobs or startup scripts.

**Single-slot shortcut:**
```bash
./build/rtt3168ctl --mode apply --dpi 800:1 --slot 1 --switch-slot
```

**Full profile configuration:**
```bash
./build/rtt3168ctl --mode apply \
  --dpi1 800:3 \
  --dpi2 1600:0 \
  --dpi3 1600:7 \
  --dpi4 2000:9 \
  --active-slot 2 \
  --rate 1000 \
  --rgb-mode breath --speed 40 \
  --cpi-action vol_up
```
*Note: You can still use `--color1..--color4` separately; if both are set, values must match.*

#### Dump Registers (`--mode dump`)
Dump raw register values from one or more banks (unique registers `0..127` for each bank; `+0x80` is mirrored). By default, the tool reads banks `0` and `1`.

```bash
./build/rtt3168ctl --mode dump
```

Probe additional candidate banks:
```bash
./build/rtt3168ctl --mode dump --dump-banks 0-7
```

#### Write Registers (`--mode write`)
Write a raw byte to a specific register (for advanced diagnostics):

```bash
./build/rtt3168ctl --mode write --reg 14 --regval 2
```

#### Experimental Monitoring (`--mode experimental`)
Stream inferred runtime/event registers in a loop. By default, it updates one console line in place, printing only changed samples until `Ctrl+C` is pressed.

```bash
./build/rtt3168ctl --mode experimental
```

Output as JSON lines for a fixed number of samples:
```bash
./build/rtt3168ctl --mode experimental --json --exp-all --exp-interval-ms 100 --exp-count 50
```

## Device IDs and udev Rules

By default, the utility targets:
- `VID = 0x093A`
- `PID = 0x2533`

**Override example:**
If your mouse uses a different VID/PID, you can override the defaults using environment variables:
```bash
MOUSE_VID=0x093A MOUSE_PID=0x2533 ./build/rtt3168ctl --mode read
```

**Configuring udev rules:**
To install the CLI and the bundled rule for `093a:2533`, run:

```bash
./build.sh
./install.sh --user --udev
```

The installer uses `sudo` to copy the rule to `/etc/udev/rules.d/` and reload udev. Reconnect the mouse afterward. The rule grants all local users read/write access to matching device nodes.

## Building from Source

Run `./build.sh` from the checkout. It performs the same checks used by CI and release builds:

1. `gofmt -l cmd internal` lists files that need formatting and the script stops if any are found. Run `go fmt ./...` to format them.
2. `go mod verify` checks downloaded dependencies.
3. `go vet ./...` analyzes the target code.
4. `go test -race ./...` runs the hardware-independent tests on the build host.

After these checks pass, the script writes `build/rtt3168ctl`. It accepts Go build flags, for example `./build.sh -ldflags="-s -w"`. For cross-builds, the host needs race-detector support and native libusb development files in addition to the target toolchain and libraries.

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/AzaOne/rtt3168ctl/issues).

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
