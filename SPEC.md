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

Observed/inferred bank-select pattern:

- Candidate `BankN` can be probed with `wIndex = (N << 8) | 0x007F`
- Confirmed examples: `N=0 -> 0x007F`, `N=1 -> 0x017F`
- Only `Bank1` is currently known to require the extra `0x2009` I/O sync
- In surveyed banks `0..255`, `reg 0x7F` and `reg 0xFF` read back the bank id itself
  (`BankN: reg 0x7F = reg 0xFF = N`)

Observed bank roles from dump/action surveys:

- `Bank1` behaves as a separate compact config/button-event bank.
- `Bank0` behaves as the primary runtime/event bank.
- Surveyed banks `2, 7, 128, 133, 147, 255` behave as additional **Bank0-like runtime
  windows**, not as empty/unused banks.
- `Bank2+` are not byte-identical to `Bank0`, but they react to motion/button activity
  through largely the same register families.

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

## 7. Inferred Runtime/Event Registers (Experimental)

These registers were inferred from read-only behavior during guided interaction tests.
They are not confirmed as stable protocol fields for writing.

Confidence labels used in this section:

- **High-confidence**: observed repeatedly and/or across multiple banks/steps with a
  coherent interpretation.
- **Medium-confidence**: repeated, but interpretation is still partly inferential.
- **Single-run / narrow-confidence**: useful in the surveyed runs, but not yet confirmed
  broadly enough to treat as stable protocol behavior.

### 7.1 Bank Roles and Runtime Windows

High-confidence observations from `bank-survey` and `bank-action` runs:

- `Bank1` is the clearest source for compact button/action state.
- `Bank0` and surveyed `Bank2+` windows expose richer runtime/event-like state.
- No fully identical banks were observed in a full `0..255` dump survey.

High-confidence `Bank0`-like windows confirmed by guided action tests:

- `Bank0`
- `Bank2`
- `Bank7`
- `Bank128`
- `Bank133`
- `Bank147`
- `Bank255`

These windows share a common runtime shortlist (Section 7.4), while still differing in
some per-bank values and specialist registers.

Survey-backed but still inferential:

- Surveyed `Bank2+` windows were action-sensitive for `move`, `left`, `right`, `middle`,
  `scroll`, `side`, and `CPI` steps.
- The most useful working model is that these are additional **Bank0-like runtime
  windows**, not empty/unused banks.

### 7.2 Button Bitmask (Bank1)

High-confidence candidates:

- `reg 0x28` (`40`) and mirror `reg 0xA8` (`168`): button bitmask
  - left click: `0x01`
  - right click: `0x02`
  - middle click: `0x04`
  - side buttons: `0x08`, `0x10`
  - CPI/DPI button: `0x20`

Additional related candidates:

- `reg 0x2A` (`42`) and mirror `reg 0xAA` (`170`): observed values `0x1F/0x2F/0x4F`
  during primary/side/CPI button actions.
- `reg 0x2B` (`43`) and mirror `reg 0xAB` (`171`): event status-like transitions,
  often involving `0x00/0x02/0x11`.
- `reg 0x75` (`117`) and mirror `reg 0xF5` (`245`): action-correlated state, common
  transitions `0x14 -> 0x15/0x16`.

### 7.3 Motion/Event Candidates (Bank0-like Windows)

High-confidence candidates:

- Move-related deltas: `reg 0x03` (`3`), `reg 0x04` (`4`), with mirrored/paired activity
  around `0x13` (`19`) and `0x93` (`147`).

Medium-confidence shared event/status group:

- `reg 0x08` (`8`) / `0x88` (`136`)
- `reg 0x33` (`51`) / `0xB3` (`179`)
- `reg 0x6C` (`108`) / `0xEC` (`236`)
- `reg 0x6B` (`107`) / `0xEB` (`235`)
- `reg 0x61` (`97`) / `0xE1` (`225`)
- `reg 0x82..0x84` (`130..132`) (especially move-related)

Medium-confidence observations:

- `reg 0x02` and mirror `reg 0x82` frequently toggle between `0x01` and `0x81`
  during motion/button/CPI activity in `Bank0`-like windows.
- `reg 0x12` (`18`) / `0x92` (`146`) frequently flip to `0xFF` on runtime activity.
- `reg 0x6B` (`107`) / `0xEB` (`235`) are among the most stable cross-bank action flags.
- `reg 0x61` (`97`) / `0xE1` (`225`) often carry the clearest action-correlated payload
  across multiple `Bank0`-like windows.

### 7.4 Cross-Bank Runtime Shortlist

High-confidence cross-bank shortlist:

The following shortlist was shared by all surveyed `Bank0`-like windows
(`0, 2, 7, 128, 133, 147, 255`) for the given action categories.

Move:

- `0x02`, `0x03`, `0x04`
- `0x12`, `0x13`
- `0x61`, `0x6B`
- `0x82`, `0x83`, `0x84`
- `0x92`, `0x93`
- `0xE1`, `0xEB`

Buttons (`left/right/middle/side`):

- `0x02`, `0x03`, `0x04`
- `0x12`
- `0x61`, `0x6B`
- `0x82`, `0x83`, `0x84`
- `0x92`
- `0xE1`, `0xEB`

Medium-confidence scroll shortlist:

- `0x6B`
- often also `0x02`, `0x03`, `0x04`, `0x12`, `0x82`, `0x83`, `0x84`, `0x92`,
  `0xE1`, `0xEB` in `Bank7`, `Bank128`, `Bank133`, `Bank147`, `Bank255`

Medium-confidence CPI shortlist:

- `0x02`, `0x82`
- commonly `0x03`, `0x12`, `0x6B`, `0x83`, `0x92`, `0xEB`
- weaker/less universal: `0x04`, `0x61`, `0x84`, `0xE1`

Single-run / narrow-confidence specialist observations:

- `Bank1 reg 0x28 / 0xA8`: clean per-button bitmask view.
- `Bank2 reg 0xE3`: right/side-button specialist.
- `Bank7 reg 0x63`: right/side-button specialist.
- `Bank128 reg 0x13`: CPI-button specialist in the surveyed run.

### 7.5 Mirror Pattern

Many volatile/event-like registers appear mirrored by `+0x80` offset.
Examples seen in the experiment:

- `0x28 <-> 0xA8`
- `0x2A <-> 0xAA`
- `0x2B <-> 0xAB`
- `0x75 <-> 0xF5`

## 8. Method and Provenance

Method used to derive Section 7:

1. Full baseline dump and idle-control step.
2. Guided per-action capture (`move`, `left`, `right`, `middle`, `side`).
3. Unknown-register diff against baseline.
4. Noise filtering: any key changing in idle-control was removed.
5. Aggregation of action-specific changes across steps.

Local tooling and artifacts:

- Capture script: `scripts/unknown-register-experiment.sh`
- Post-filter script: `scripts/unknown-register-action-specific.sh`
- Bank survey: `scripts/bank-survey.sh`
- Bank action survey: `scripts/bank-action-survey.sh`
- Bank shortlist post-process: `scripts/bank-action-shortlist.sh`

Status note:

- Section 7 is empirical and should be treated as *experimental* until confirmed by
  repeated runs on multiple units/firmware revisions.
- High-confidence items are the best current working map for read-only diagnostics.
- Medium-confidence and single-run items should not yet be relied on as stable semantics.

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
