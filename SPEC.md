# RTT3168CG2 Protocol Specification

This document describes the RTT3168CG2 mouse control protocol used by `rtt3168ctl`.

## 1. Transport

- Interface: USB control transfer (vendor specific).
- Request types:
  - `bmRequestType = 0x40` (`Vendor | Device | OUT`)
  - `bmRequestType = 0xC0` (`Vendor | Device | IN`)
- Request codes:
  - `bRequest = 0x01` (`ReqCodeControl`)
  - `bRequest = 0x06` (`ReqCodeReset`)
- Base `wValue`: `0x0100` (`ControlValDefault`).

## 2. Operation Format

### 2.1 Register Read

- `bmRequestType = 0xC0`
- `bRequest = 0x01`
- `wValue = 0x0100`
- `wIndex = reg_id`
- `data_len = 1`
- Response: 1 byte register value.

### 2.2 Register Write

- `bmRequestType = 0x40`
- `bRequest = 0x01`
- `wValue = 0x0100`
- `wIndex = (value << 8) | reg_id`
- `data_len = 0`
- A ~20 ms delay is applied after each write.
- In this write format, `reg_id` effectively occupies the low byte, so writable register
  addresses are `0x00..0xFF`.

## 3. Banks and Control Indexes

- Select Bank0: `wIndex = 0x007F` (`127`)
- Select Bank1: `wIndex = 0x017F` (`383`)
- Bank1 I/O sync: `wIndex = 0x2009` (`8201`)
- Unlock: `wIndex = 0x5A09` (`23049`)
- Lock: `wIndex = 0x0009` (`9`)

All operations above use `OUT + bRequest=0x01 + wValue=0x0100`.

## 4. Session Lifecycle

### 4.1 BeginSession

1. `OUT, req=0x01, val=0x0000, idx=0x0000`
2. `OUT, req=0x01, val=0x0100, idx=0x5A09` (unlock)
3. `OUT, req=0x01, val=0x0100, idx=0x017F` (bank1)
4. `OUT, req=0x01, val=0x0100, idx=0x2009` (bank1 I/O sync)

### 4.2 EndSession

1. `OUT, req=0x01, val=0x0100, idx=0x007F` (bank0)
2. `OUT, req=0x01, val=0x0100, idx=0x0009` (lock)
3. `OUT, req=0x06, val=0x0000, idx=0x0000` (reset/commit)

## 5. Register Map (Bank1)

- `0x01` (`1`): RGB speed
- `0x02` (`2`): DPI slot 1 (`color<<4 | dpi_idx`)
- `0x03` (`3`): DPI slot 2
- `0x04` (`4`): DPI slot 3
- `0x05` (`5`): DPI slot 4
- `0x06` (`6`): DPI slot 5
- `0x07` (`7`): DPI slot 6
- `0x09` (`9`): active DPI slot selector
- `0x0A` (`10`): RGB mode
- `0x0B` (`11`): CPI button action
- `0x0E` (`14`): polling rate
- `0x1D` (`29`): sensor ID

## 6. Field Semantics

### 6.1 DPI Slots (`reg 0x02..0x05`)

- Byte format: `bits[7:4]=color`, `bits[3:0]=dpi_idx`
- Conversion:
  - `dpi = 200 + dpi_idx*200`
  - Tool-enforced range: `200..3200`, step `200`

### 6.2 Active DPI Slot

- Write to `reg 0x09`: `(slot-1)*0x20`
  - slot1=`0x00`, slot2=`0x20`, slot3=`0x40`, slot4=`0x60`
- During `read`, active slot is read from **Bank0 reg 2**, decode rules:
  - `0x00..0x03 => slot = raw+1`
  - `0x20 => 2`, `0x40 => 3`, `0x60 => 4`
  - fallback: `(raw & 0x03)+1`

### 6.3 Polling Rate (`reg 0x0E`)

- `125Hz = 0xC2`
- `250Hz = 0x82`
- `500Hz = 0x42`
- `1000Hz = 0x02`

Note: for `1000Hz`, extra stabilization is used:
- initial pause ~60 ms;
- up to 2 retries with 50/100 ms backoff;
- Bank1 re-sync before retry.

### 6.4 RGB Mode (`reg 0x0A`)

Upper nibble (`bits[7:4]`) defines the mode:
- `0x00`: Always On
- `0x20`: Breathing
- `0x40`: Breathing + Segment Cycle
- `0x60`: 6 Color Cycle
- `0x80`: 12 Color Cycle
- `0xA0`: 768 Color Cycle
- `0xE0`: Off

When writing, the tool only updates the upper nibble and preserves the lower nibble:
- `new = (mode & 0xF0) | (old & 0x0F)`

### 6.5 CPI Action (`reg 0x0B`)

Known codes:

| Action | Code (hex) | Code (dec) |
|---|---:|---:|
| backward | `0xE0` | 224 |
| forward | `0xE1` | 225 |
| ctrl | `0xE2` | 226 |
| win | `0xE3` | 227 |
| browser | `0xE4` | 228 |
| double_click | `0xE5` | 229 |
| sniper | `0xE6` | 230 |
| rgb_switch | `0xE7` | 231 |
| dpi_cycle | `0xE8` | 232 |
| play_pause | `0xEC` | 236 |
| mute | `0xED` | 237 |
| next_track | `0xEE` | 238 |
| prev_track | `0xEF` | 239 |
| stop | `0xF0` | 240 |
| vol_up | `0xF2` | 242 |
| vol_down | `0xF3` | 243 |
| win_d | `0xF5` | 245 |
| copy | `0xF6` | 246 |
| paste | `0xF7` | 247 |
| prev_page | `0xF8` | 248 |
| next_page | `0xF9` | 249 |
| my_computer | `0xFA` | 250 |
| calculator | `0xFB` | 251 |
| ctrl_w | `0xFC` | 252 |