# rtt3168ctl

CLI utility for controlling a USB mouse based on the **PixArt RTT3168CG2** chip via vendor control transfers.

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
- `dpi` - change slot DPI/color
- `switch` - switch active DPI slot
- `rgb` - configure lighting mode
- `rate` - set USB polling rate (125/250/500/1000)
- `cpi` - set CPI button action
- `apply` - apply multiple settings in one command (cron-friendly)
- `dump` - dump bank 1 registers (1..30)
- `write` - raw register write (advanced)

## Main Parameters

- `-slot` - DPI slot (`1-4`)
- `-val` - mode value
- `-color` - color index (`0-15`), `-1` = keep current
- `-switch-slot` - for `-mode dpi` also activate target slot after write
- `-speed` - RGB speed (`0-255`), `-1` = keep current
- `-json` - JSON output for `-mode read`
- `-reg`, `-regval` - raw values for `write`
- `-dpi1..-dpi4` - in `-mode apply`: DPI for each slot (`200..3200`, step `200`)
- `-color1..-color4` - in `-mode apply`: color for each slot (`0..15`), `-1` = keep current
- `-active-slot` - in `-mode apply`: activate slot (`1-4`) after applying settings
- `-rate` - in `-mode apply`: polling rate (`125/250/500/1000`)
- `-rgb-mode` - in `-mode apply`: RGB mode value
- `-cpi-action` - in `-mode apply`: CPI action value

## Mode Values

### `-mode dpi`
- `-val`: DPI in range `200..3200`, step `200`
- `-slot`: `1..4`
- `-color`: `0..15` (optional)
- `-switch-slot`: optional, activates target slot after updating DPI/color

Example:
```bash
./build/rtt3168ctl -mode dpi -slot 1 -val 800 -color 3
```

Switch active slot while writing:
```bash
./build/rtt3168ctl -mode dpi -slot 2 -val 1600 -color 0 -switch-slot
```

### `-mode rgb`
Supported `-val` values:
- `off`
- `on`
- `breath`
- `cycle6`
- `cycle12`
- `cycle768`

Example:
```bash
./build/rtt3168ctl -mode rgb -val breath -speed 40
```

### `-mode cpi`
Supported actions (`-val`):
- `backward`, `forward`, `ctrl`, `win`, `browser`, `double_click`, `sniper`, `rgb_switch`, `dpi_cycle`
- `play_pause`, `mute`, `next_track`, `prev_track`, `stop`, `vol_up`, `vol_down`, `win_d`
- `copy`, `paste`, `prev_page`, `next_page`, `my_computer`, `calculator`, `ctrl_w`

Example:
```bash
./build/rtt3168ctl -mode cpi -val vol_up
```

### `-mode apply`
Apply several settings in one run, for example from cron/startup scripts:

```bash
./build/rtt3168ctl -mode apply \
  -dpi1 800 -color1 3 \
  -dpi2 1200 -color2 5 \
  -dpi3 1600 -color3 7 \
  -dpi4 2000 -color4 9 \
  -active-slot 2 \
  -rate 1000 \
  -rgb-mode breath -speed 40 \
  -cpi-action vol_up
```

## VID/PID via Environment Variables

Defaults:
- `VID = 0x093A`
- `PID = 0x2533`

Override example:

```bash
MOUSE_VID=0x093A MOUSE_PID=0x2533 ./build/rtt3168ctl -mode read
```
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
