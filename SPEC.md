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
- Inferred color lookup:
  - `color_idx = 0..15` appears to select a 3-byte palette entry in `Bank1 reg 0x30..0x5F`
  - candidate entry base: `0x30 + color_idx*3`
  - example: slot value `0x43` means `color_idx=4`, which points to `reg 0x3C..0x3E`
  - this mapping is consistent with:
    - the LUT-sized block length (`48 = 16 * 3` bytes);
    - color-index changes touching only `reg 0x02..0x05`, not `reg 0x30..0x5F`;
    - successful direct writes to many bytes inside `reg 0x30..0x5F`

### 6.1.1 RGB Palette LUT (`Bank1 reg 0x30..0x5F`) (Inferred)

- Candidate role: 16-entry color lookup table used by the `color_idx` nibble in DPI slot registers.
- Note: This block is currently **read-only** in application usage, and currently its components remain undefined in code (`registers.go`).
- Practical note:
  - the block behaves like a palette table, not a per-slot color store;
  - many bytes are writable, but some are masked or quantized on write, so entries should not yet be treated as unrestricted raw `RGB888`.

See Section 10 for the full Bank 1 map which incorporates this 48-byte RGB block.


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

## 7. Inferred Runtime/Event Registers (Experimental)

These registers were inferred from read-only behavior during guided interaction tests.
They are not confirmed as stable protocol fields for writing.

### 7.1 Button Bitmask (Bank1)

High-confidence candidates:

- `reg 0x28` (`40`): button bitmask
  - left click: `0x01`
  - right click: `0x02`
  - middle click: `0x04`
  - side buttons: `0x08`, `0x10`

Additional related candidates:

- `reg 0x2A` (`42`): observed values `0x1F/0x2F/0x4F`
  during primary/side button actions.
- `reg 0x2B` (`43`): event status-like transitions,
  often involving `0x00/0x02/0x11`.
- `reg 0x75` (`117`): action-correlated state, common
  transitions `0x14 -> 0x15/0x16`.

### 7.2 Motion/Event Candidates (Bank0)

High-confidence candidates:

- Move-related deltas: `reg 0x03` (`3`), `reg 0x04` (`4`), with activity
  around `0x13` (`19`).

Medium-confidence shared event/status group:

- `reg 0x08` (`8`)
- `reg 0x33` (`51`)
- `reg 0x6C` (`108`)
- `reg 0x6B` (`107`)
- `reg 0x61` (`97`)

### 7.3 Mirror Pattern

Many volatile/event-like registers appear mirrored by a `+0x80` offset.
As detailed in Section 9.4, these are **hardware aliases** resulting from incomplete address decoding. They are not independent registers and reading them provides no additional data while doubling the USB I/O overhead.

All client implementations should query only the unique `0x00..0x7F` range.

Examples seen in raw experiments:

- `0x28` physically aliases `0xA8`
- `0x2A` physically aliases `0xAA`
- `0x2B` physically aliases `0xAB`
- `0x75` physically aliases `0xF5`

## 8. Method and Provenance

Method used to derive Section 7:

1. Baseline dump over the unique `0x00..0x7F` half and idle-control step.
2. Guided per-action capture (`move`, `left`, `right`, `middle`, `side`).
3. Unknown-register diff against baseline.
4. Noise filtering: any key changing in idle-control was removed.
5. Aggregation of action-specific changes across steps.

Local tooling and artifacts:

- Capture script: `scripts/unknown-register-experiment.sh`
- Post-filter script: `scripts/unknown-register-action-specific.sh`

Status note:

- Section 7 is empirical and should be treated as *experimental* until confirmed by
  repeated runs on multiple units/firmware revisions.

## 9. Architectural Hypotheses and Design Rationale

The structure of the RTT3168CG2 protocol exhibits characteristics typical of low-cost microcontrollers (MCUs) or specialized ASICs (like those from PixArt, Sonix, or Holtek) used in PC peripherals. The following hypotheses explain the engineering reasoning behind the observed protocol anomalies.

### 9.1 Control Transfer Optimization (`wIndex` Packing)

**Observation:** Register writes use `wIndex = (value << 8) | reg_id` with `data_len = 0`.
**Rationale:** A standard USB SETUP packet is exactly 8 bytes long (including `bmRequestType`, `bRequest`, `wValue`, `wIndex`, and `wLength`). By packing both the 1-byte payload (`value`) and the 1-byte target address (`reg_id`) into the 16-bit `wIndex` field, the firmware entirely avoids the USB DATA stage. 
- It reduces bus overhead (requiring only a SETUP packet instead of SETUP + DATA + ACK).
- It vastly simplifies the `Endpoint 0` control request handler within the MCU firmware.

### 9.2 Banked Memory Architecture

**Observation:** Registers are divided into banks, switched via `reg 0x7F`.
**Rationale:** Typical 8-bit peripheral microcontrollers (often based on the 8051 architecture) have heavily constrained Special Function Register (SFR) and RAM address spaces (usually 128 or 256 bytes). Memory paging (banks) is required to manage complex peripherals (optical sensor, RGB controller, macro EEPROM).
- **Bank1** likely maps to or configures Non-Volatile Memory (Flash/EEPROM). This explains why it is the primary configuration bank and requires an explicit `I/O sync` (`wIndex = 0x2009`) to commit settings.
- **Bank0** serves as the primary working RAM, holding volatile state such as sensor buffers, live button logic, and coordinate deltas.

### 9.3 Magic Numbers and Protection

**Observation:** The unlock sequence requires writing `0x5A` to `reg 0x09` (`wIndex = 0x5A09`).
**Rationale:** `0x5A` (binary `01011010`) and `0xA5` (`10100101`) are classic embedded "magic numbers" featuring an alternating bit pattern. It is statistically highly improbable for this pattern to be generated accidentally by line noise, EMI, or pointer bugs. Requiring this specific byte acts as a write-protect mechanism, ensuring transient power spikes do not corrupt DPI or RGB configuration memory.

### 9.4 Register Aliasing (+0x80 Mirror Pattern)

**Observation:** Extensive mirroring with a `+0x80` offset (e.g., `0x28 <-> 0xA8` detailed in Section 7.5).
**Rationale:** This is almost certainly hardware-level **incomplete address decoding**. The MCU likely uses only 7 bits for addressing RAM within a bank (`0x00..0x7F`), physically ignoring the most significant bit (MSB). Consequently, addressing `0x28` (`0010 1000`) and `0xA8` (`1010 1000`) resolves to the exact same physical silicon gates.

### 9.5 Phantom Banks (Bank0-like Windows)

**Observation:** Banks `2, 7, 128, 133, 255` are not empty but behave as near-identical variants of `Bank0`.
**Rationale:** Similar to register aliasing, this indicates incomplete decoding of the bank selector register (`reg 0x7F`). If the physical ASIC only implements two primary hardware banks (0 and 1), the address decoder may only evaluate the lowest bit(s) of the bank selector. As a result, out-of-bounds bank indices "fold" back into the primary banks, occasionally exposing undefined test modes or mixed register behavior.

### 9.6 Diagnostic Interface vs. Standard HID

**Observation:** Event and motion polling (Section 7) can be performed via Vendor-Specific Control Transfers (`bmRequestType = 0xC0`).
**Rationale:** During normal OS operation, the mouse transmits motion and clicks via standard USB Interrupt endpoints (as a class-compliant HID device). The Control Transfer interface documented here is a **vendor-specific diagnostic and configuration interface**. The manufacturer tool uses this protocol to bypass the OS HID stack and read the MCU/sensor RAM directly (e.g., for factory calibration or drawing DPI tracking graphs). This explains the raw, unpolished nature of the exposed event registers.

## 10. Complete Register Maps (0x00 - 0x7F)

This section details the full 128-byte address space for both Bank 0 and Bank 1, integrating default values sampled from experimental data and current defined constants.

### 10.1 Bank 0

Defined in code: 7 / 128 (Assumed/Inferred: 0 / 128)

| Reg (Hex) | Dec | Default (`sample-01`) | Status | Function / Role |
|---|---|---|---|---|
| `0x00` | 0 | `0x31` | Undefined | Unknown |
| `0x01` | 1 | `0xA1` | Undefined | Unknown |
| `0x02` | 2 | `0x01` | Undefined | Unknown |
| `0x03` | 3 | `0x00` | **Defined** | Move X (RegExpB0MoveX) |
| `0x04` | 4 | `0x00` | **Defined** | Move Y (RegExpB0MoveY) |
| `0x05` | 5 | `0x80` | Undefined | Unknown |
| `0x06` | 6 | `0x00` | Undefined | Unknown |
| `0x07` | 7 | `0x00` | Undefined | Unknown |
| `0x08` | 8 | `0x02` | **Defined** | Event Latch (RegExpB0EventLatch) |
| `0x09` | 9 | `0x5A` | Undefined | Unknown |
| `0x0A` | 10 | `0x17` | Undefined | Unknown |
| `0x0B` | 11 | `0x00` | Undefined | Unknown |
| `0x0C` | 12 | `0xF0` | Undefined | Unknown |
| `0x0D` | 13 | `0x13` | Undefined | Unknown |
| `0x0E` | 14 | `0x13` | Undefined | Unknown |
| `0x0F` | 15 | `0x00` | Undefined | Unknown |
| `0x10` | 16 | `0x23` | Undefined | Unknown |
| `0x11` | 17 | `0x80` | Undefined | Unknown |
| `0x12` | 18 | `0x00` | Undefined | Unknown |
| `0x13` | 19 | `0x00` | Undefined | Unknown |
| `0x14` | 20 | `0x00` | Undefined | Unknown |
| `0x15` | 21 | `0x3C` | Undefined | Unknown |
| `0x16` | 22 | `0x40` | Undefined | Unknown |
| `0x17` | 23 | `0x32` | Undefined | Unknown |
| `0x18` | 24 | `0x40` | Undefined | Unknown |
| `0x19` | 25 | `0x4B` | Undefined | Unknown |
| `0x1A` | 26 | `0xFC` | Undefined | Unknown |
| `0x1B` | 27 | `0x4B` | Undefined | Unknown |
| `0x1C` | 28 | `0xD5` | Undefined | Unknown |
| `0x1D` | 29 | `0xC8` | Undefined | Unknown |
| `0x1E` | 30 | `0xAA` | Undefined | Unknown |
| `0x1F` | 31 | `0x97` | Undefined | Unknown |
| `0x20` | 32 | `0x28` | Undefined | Unknown |
| `0x21` | 33 | `0x1E` | Undefined | Unknown |
| `0x22` | 34 | `0x1E` | Undefined | Unknown |
| `0x23` | 35 | `0x14` | Undefined | Unknown |
| `0x24` | 36 | `0xEA` | Undefined | Unknown |
| `0x25` | 37 | `0x32` | Undefined | Unknown |
| `0x26` | 38 | `0x22` | Undefined | Unknown |
| `0x27` | 39 | `0x05` | Undefined | Unknown |
| `0x28` | 40 | `0x14` | Undefined | Unknown |
| `0x29` | 41 | `0x04` | Undefined | Unknown |
| `0x2A` | 42 | `0x00` | Undefined | Unknown |
| `0x2B` | 43 | `0x02` | Undefined | Unknown |
| `0x2C` | 44 | `0x00` | Undefined | Unknown |
| `0x2D` | 45 | `0x0F` | Undefined | Unknown |
| `0x2E` | 46 | `0x00` | Undefined | Unknown |
| `0x2F` | 47 | `0x05` | Undefined | Unknown |
| `0x30` | 48 | `0x08` | Undefined | Unknown |
| `0x31` | 49 | `0x01` | Undefined | Unknown |
| `0x32` | 50 | `0x92` | Undefined | Unknown |
| `0x33` | 51 | `0x02` | **Defined** | Event Group (RegExpB0EventGroup) |
| `0x34` | 52 | `0x00` | Undefined | Unknown |
| `0x35` | 53 | `0x22` | Undefined | Unknown |
| `0x36` | 54 | `0x05` | Undefined | Unknown |
| `0x37` | 55 | `0x00` | Undefined | Unknown |
| `0x38` | 56 | `0x00` | Undefined | Unknown |
| `0x39` | 57 | `0x60` | Undefined | Unknown |
| `0x3A` | 58 | `0x00` | Undefined | Unknown |
| `0x3B` | 59 | `0x01` | Undefined | Unknown |
| `0x3C` | 60 | `0x23` | Undefined | Unknown |
| `0x3D` | 61 | `0x31` | Undefined | Unknown |
| `0x3E` | 62 | `0xA5` | Undefined | Unknown |
| `0x3F` | 63 | `0x02` | Undefined | Unknown |
| `0x40` | 64 | `0x40` | Undefined | Unknown |
| `0x41` | 65 | `0x01` | Undefined | Unknown |
| `0x42` | 66 | `0x1B` | Undefined | Unknown |
| `0x43` | 67 | `0x81` | Undefined | Unknown |
| `0x44` | 68 | `0xEA` | Undefined | Unknown |
| `0x45` | 69 | `0x55` | Undefined | Unknown |
| `0x46` | 70 | `0x45` | Undefined | Unknown |
| `0x47` | 71 | `0x4C` | Undefined | Unknown |
| `0x48` | 72 | `0x50` | Undefined | Unknown |
| `0x49` | 73 | `0x23` | Undefined | Unknown |
| `0x4A` | 74 | `0x95` | Undefined | Unknown |
| `0x4B` | 75 | `0x00` | Undefined | Unknown |
| `0x4C` | 76 | `0x6D` | Undefined | Unknown |
| `0x4D` | 77 | `0x0A` | Undefined | Unknown |
| `0x4E` | 78 | `0x08` | Undefined | Unknown |
| `0x4F` | 79 | `0x04` | Undefined | Unknown |
| `0x50` | 80 | `0x0B` | Undefined | Unknown |
| `0x51` | 81 | `0x04` | Undefined | Unknown |
| `0x52` | 82 | `0x04` | Undefined | Unknown |
| `0x53` | 83 | `0x00` | Undefined | Unknown |
| `0x54` | 84 | `0x00` | Undefined | Unknown |
| `0x55` | 85 | `0x03` | Undefined | Unknown |
| `0x56` | 86 | `0x80` | Undefined | Unknown |
| `0x57` | 87 | `0x06` | Undefined | Unknown |
| `0x58` | 88 | `0x1D` | Undefined | Unknown |
| `0x59` | 89 | `0x07` | Undefined | Unknown |
| `0x5A` | 90 | `0x1B` | Undefined | Unknown |
| `0x5B` | 91 | `0xF0` | Undefined | Unknown |
| `0x5C` | 92 | `0x10` | Undefined | Unknown |
| `0x5D` | 93 | `0x02` | Undefined | Unknown |
| `0x5E` | 94 | `0x00` | Undefined | Unknown |
| `0x5F` | 95 | `0xAA` | Undefined | Unknown |
| `0x60` | 96 | `0x7E` | Undefined | Unknown |
| `0x61` | 97 | `0x3F` | **Defined** | Event State C (RegExpB0EventStateC) |
| `0x62` | 98 | `0x5F` | Undefined | Unknown |
| `0x63` | 99 | `0x00` | Undefined | Unknown |
| `0x64` | 100 | `0x20` | Undefined | Unknown |
| `0x65` | 101 | `0x1F` | Undefined | Unknown |
| `0x66` | 102 | `0x2A` | Undefined | Unknown |
| `0x67` | 103 | `0xC6` | Undefined | Unknown |
| `0x68` | 104 | `0xEA` | Undefined | Unknown |
| `0x69` | 105 | `0xC6` | Undefined | Unknown |
| `0x6A` | 106 | `0x00` | Undefined | Unknown |
| `0x6B` | 107 | `0x00` | **Defined** | Event State A (RegExpB0EventStateA) |
| `0x6C` | 108 | `0x05` | **Defined** | Event State B (RegExpB0EventStateB) |
| `0x6D` | 109 | `0xA0` | Undefined | Unknown |
| `0x6E` | 110 | `0x00` | Undefined | Unknown |
| `0x6F` | 111 | `0x00` | Undefined | Unknown |
| `0x70` | 112 | `0x00` | Undefined | Unknown |
| `0x71` | 113 | `0x00` | Undefined | Unknown |
| `0x72` | 114 | `0x00` | Undefined | Unknown |
| `0x73` | 115 | `0x00` | Undefined | Unknown |
| `0x74` | 116 | `0x00` | Undefined | Unknown |
| `0x75` | 117 | `0x00` | Undefined | Unknown |
| `0x76` | 118 | `0x00` | Undefined | Unknown |
| `0x77` | 119 | `0x00` | Undefined | Unknown |
| `0x78` | 120 | `0x00` | Undefined | Unknown |
| `0x79` | 121 | `0x00` | Undefined | Unknown |
| `0x7A` | 122 | `0x00` | Undefined | Unknown |
| `0x7B` | 123 | `0x00` | Undefined | Unknown |
| `0x7C` | 124 | `0x00` | Undefined | Unknown |
| `0x7D` | 125 | `0x00` | Undefined | Unknown |
| `0x7E` | 126 | `0x00` | Undefined | Unknown |
| `0x7F` | 127 | `0x00` | Undefined | Unknown |

### 10.2 Bank 1

Defined in code: 16 / 128 (Assumed/Inferred: 48 / 128)

| Reg (Hex) | Dec | Default (`sample-01`) | Status | Function / Role |
|---|---|---|---|---|
| `0x00` | 0 | `0x8C` | Undefined | Unknown |
| `0x01` | 1 | `0x80` | **Defined** | RGB Speed (RegRGBSpeed) |
| `0x02` | 2 | `0x43` | **Defined** | DPI 1 (RegDPI1) |
| `0x03` | 3 | `0x07` | **Defined** | DPI 2 (RegDPI2) |
| `0x04` | 4 | `0x4B` | **Defined** | DPI 3 (RegDPI3) |
| `0x05` | 5 | `0xAF` | **Defined** | DPI 4 (RegDPI4) |
| `0x06` | 6 | `0x27` | **Defined** | DPI 5 (RegDPI5) |
| `0x07` | 7 | `0x6F` | **Defined** | DPI 6 (RegDPI6) |
| `0x08` | 8 | `0x43` | Undefined | Unknown |
| `0x09` | 9 | `0x20` | **Defined** | DPI Select (RegDPISelect) |
| `0x0A` | 10 | `0x01` | **Defined** | RGB Mode (RegRGBMode) |
| `0x0B` | 11 | `0xE8` | **Defined** | CPI Button (RegCPIButton) |
| `0x0C` | 12 | `0x45` | Undefined | Unknown |
| `0x0D` | 13 | `0x24` | Undefined | Unknown |
| `0x0E` | 14 | `0x82` | **Defined** | Rate (RegRate) |
| `0x0F` | 15 | `0x23` | Undefined | Unknown |
| `0x10` | 16 | `0x4B` | Undefined | Unknown |
| `0x11` | 17 | `0x07` | Undefined | Unknown |
| `0x12` | 18 | `0x33` | Undefined | Unknown |
| `0x13` | 19 | `0x25` | Undefined | Unknown |
| `0x14` | 20 | `0x3A` | Undefined | Unknown |
| `0x15` | 21 | `0x09` | Undefined | Unknown |
| `0x16` | 22 | `0xFF` | Undefined | Unknown |
| `0x17` | 23 | `0xFF` | Undefined | Unknown |
| `0x18` | 24 | `0xFF` | Undefined | Unknown |
| `0x19` | 25 | `0xFF` | Undefined | Unknown |
| `0x1A` | 26 | `0xFF` | Undefined | Unknown |
| `0x1B` | 27 | `0xFF` | Undefined | Unknown |
| `0x1C` | 28 | `0xFF` | Undefined | Unknown |
| `0x1D` | 29 | `0xEF` | **Defined** | Sensor ID (RegSensorID) |
| `0x1E` | 30 | `0x24` | Undefined | Unknown |
| `0x1F` | 31 | `0x0D` | Undefined | Unknown |
| `0x20` | 32 | `0x58` | Undefined | Unknown |
| `0x21` | 33 | `0xAC` | Undefined | Unknown |
| `0x22` | 34 | `0x13` | Undefined | Unknown |
| `0x23` | 35 | `0x00` | Undefined | Unknown |
| `0x24` | 36 | `0x00` | Undefined | Unknown |
| `0x25` | 37 | `0x00` | Undefined | Unknown |
| `0x26` | 38 | `0x00` | Undefined | Unknown |
| `0x27` | 39 | `0x00` | Undefined | Unknown |
| `0x28` | 40 | `0x00` | **Defined** | Buttons Mask (RegExpB1ButtonsMask) |
| `0x29` | 41 | `0x00` | Undefined | Unknown |
| `0x2A` | 42 | `0x1E` | **Defined** | Buttons State A (RegExpB1ButtonsStateA) |
| `0x2B` | 43 | `0x11` | **Defined** | Buttons State B (RegExpB1ButtonsStateB) |
| `0x2C` | 44 | `0x00` | Undefined | Unknown |
| `0x2D` | 45 | `0x08` | Undefined | Unknown |
| `0x2E` | 46 | `0x0B` | Undefined | Unknown |
| `0x2F` | 47 | `0x13` | Undefined | Unknown |
| `0x30` | 48 | `0x02` | *Inferred* | RGB Palette Entry 0 - Component 1 |
| `0x31` | 49 | `0x62` | *Inferred* | RGB Palette Entry 0 - Component 2 |
| `0x32` | 50 | `0x55` | *Inferred* | RGB Palette Entry 0 - Component 3 |
| `0x33` | 51 | `0x1F` | *Inferred* | RGB Palette Entry 1 - Component 1 |
| `0x34` | 52 | `0x4C` | *Inferred* | RGB Palette Entry 1 - Component 2 |
| `0x35` | 53 | `0x0A` | *Inferred* | RGB Palette Entry 1 - Component 3 |
| `0x36` | 54 | `0x20` | *Inferred* | RGB Palette Entry 2 - Component 1 |
| `0x37` | 55 | `0x80` | *Inferred* | RGB Palette Entry 2 - Component 2 |
| `0x38` | 56 | `0x00` | *Inferred* | RGB Palette Entry 2 - Component 3 |
| `0x39` | 57 | `0xAA` | *Inferred* | RGB Palette Entry 3 - Component 1 |
| `0x3A` | 58 | `0x12` | *Inferred* | RGB Palette Entry 3 - Component 2 |
| `0x3B` | 59 | `0x24` | *Inferred* | RGB Palette Entry 3 - Component 3 |
| `0x3C` | 60 | `0x50` | *Inferred* | RGB Palette Entry 4 - Component 1 |
| `0x3D` | 61 | `0x7F` | *Inferred* | RGB Palette Entry 4 - Component 2 |
| `0x3E` | 62 | `0xFA` | *Inferred* | RGB Palette Entry 4 - Component 3 |
| `0x3F` | 63 | `0xDE` | *Inferred* | RGB Palette Entry 5 - Component 1 |
| `0x40` | 64 | `0x58` | *Inferred* | RGB Palette Entry 5 - Component 2 |
| `0x41` | 65 | `0xA0` | *Inferred* | RGB Palette Entry 5 - Component 3 |
| `0x42` | 66 | `0x3C` | *Inferred* | RGB Palette Entry 6 - Component 1 |
| `0x43` | 67 | `0x08` | *Inferred* | RGB Palette Entry 6 - Component 2 |
| `0x44` | 68 | `0xC8` | *Inferred* | RGB Palette Entry 6 - Component 3 |
| `0x45` | 69 | `0x00` | *Inferred* | RGB Palette Entry 7 - Component 1 |
| `0x46` | 70 | `0x01` | *Inferred* | RGB Palette Entry 7 - Component 2 |
| `0x47` | 71 | `0x64` | *Inferred* | RGB Palette Entry 7 - Component 3 |
| `0x48` | 72 | `0x01` | *Inferred* | RGB Palette Entry 8 - Component 1 |
| `0x49` | 73 | `0xA0` | *Inferred* | RGB Palette Entry 8 - Component 2 |
| `0x4A` | 74 | `0x55` | *Inferred* | RGB Palette Entry 8 - Component 3 |
| `0x4B` | 75 | `0x77` | *Inferred* | RGB Palette Entry 9 - Component 1 |
| `0x4C` | 76 | `0xBB` | *Inferred* | RGB Palette Entry 9 - Component 2 |
| `0x4D` | 77 | `0x77` | *Inferred* | RGB Palette Entry 9 - Component 3 |
| `0x4E` | 78 | `0x11` | *Inferred* | RGB Palette Entry 10 - Component 1 |
| `0x4F` | 79 | `0x50` | *Inferred* | RGB Palette Entry 10 - Component 2 |
| `0x50` | 80 | `0x87` | *Inferred* | RGB Palette Entry 10 - Component 3 |
| `0x51` | 81 | `0x25` | *Inferred* | RGB Palette Entry 11 - Component 1 |
| `0x52` | 82 | `0x80` | *Inferred* | RGB Palette Entry 11 - Component 2 |
| `0x53` | 83 | `0xEB` | *Inferred* | RGB Palette Entry 11 - Component 3 |
| `0x54` | 84 | `0x3F` | *Inferred* | RGB Palette Entry 12 - Component 1 |
| `0x55` | 85 | `0x77` | *Inferred* | RGB Palette Entry 12 - Component 2 |
| `0x56` | 86 | `0x47` | *Inferred* | RGB Palette Entry 12 - Component 3 |
| `0x57` | 87 | `0xE7` | *Inferred* | RGB Palette Entry 13 - Component 1 |
| `0x58` | 88 | `0x0C` | *Inferred* | RGB Palette Entry 13 - Component 2 |
| `0x59` | 89 | `0x0C` | *Inferred* | RGB Palette Entry 13 - Component 3 |
| `0x5A` | 90 | `0x00` | *Inferred* | RGB Palette Entry 14 - Component 1 |
| `0x5B` | 91 | `0x17` | *Inferred* | RGB Palette Entry 14 - Component 2 |
| `0x5C` | 92 | `0x18` | *Inferred* | RGB Palette Entry 14 - Component 3 |
| `0x5D` | 93 | `0x8C` | *Inferred* | RGB Palette Entry 15 - Component 1 |
| `0x5E` | 94 | `0x00` | *Inferred* | RGB Palette Entry 15 - Component 2 |
| `0x5F` | 95 | `0x00` | *Inferred* | RGB Palette Entry 15 - Component 3 |
| `0x60` | 96 | `0x00` | Undefined | Unknown |
| `0x61` | 97 | `0x00` | Undefined | Unknown |
| `0x62` | 98 | `0x08` | Undefined | Unknown |
| `0x63` | 99 | `0x00` | Undefined | Unknown |
| `0x64` | 100 | `0x40` | Undefined | Unknown |
| `0x65` | 101 | `0x00` | Undefined | Unknown |
| `0x66` | 102 | `0x60` | Undefined | Unknown |
| `0x67` | 103 | `0xB0` | Undefined | Unknown |
| `0x68` | 104 | `0xBB` | Undefined | Unknown |
| `0x69` | 105 | `0x00` | Undefined | Unknown |
| `0x6A` | 106 | `0x50` | Undefined | Unknown |
| `0x6B` | 107 | `0xBB` | Undefined | Unknown |
| `0x6C` | 108 | `0x00` | Undefined | Unknown |
| `0x6D` | 109 | `0x00` | Undefined | Unknown |
| `0x6E` | 110 | `0x00` | Undefined | Unknown |
| `0x6F` | 111 | `0x10` | Undefined | Unknown |
| `0x70` | 112 | `0x6C` | Undefined | Unknown |
| `0x71` | 113 | `0x01` | Undefined | Unknown |
| `0x72` | 114 | `0x02` | Undefined | Unknown |
| `0x73` | 115 | `0x6D` | Undefined | Unknown |
| `0x74` | 116 | `0x57` | Undefined | Unknown |
| `0x75` | 117 | `0x16` | **Defined** | Event State (RegExpB1EventState) |
| `0x76` | 118 | `0x80` | Undefined | Unknown |
| `0x77` | 119 | `0xA4` | Undefined | Unknown |
| `0x78` | 120 | `0x00` | Undefined | Unknown |
| `0x79` | 121 | `0xCC` | Undefined | Unknown |
| `0x7A` | 122 | `0xCC` | Undefined | Unknown |
| `0x7B` | 123 | `0x74` | Undefined | Unknown |
| `0x7C` | 124 | `0xFF` | Undefined | Unknown |
| `0x7D` | 125 | `0xFF` | Undefined | Unknown |
| `0x7E` | 126 | `0x7F` | Undefined | Unknown |
| `0x7F` | 127 | `0x01` | Undefined | Unknown |
