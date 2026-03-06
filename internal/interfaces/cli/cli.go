package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"rtt3168ctl/internal/facade"
)

func Parse(args []string, binName string, errOut io.Writer) (facade.Command, bool, error) {
	fs := flag.NewFlagSet(binName, flag.ContinueOnError)
	fs.SetOutput(errOut)

	modePtr := fs.String("mode", "", "Operation mode")
	dpiSlotPtr := fs.Int("slot", 1, "Target DPI slot (1-4)")
	dpiPtr := fs.String("dpi", "", "Value for -slot as DPI or DPI:Color (e.g. 800 or 800:1)")
	colorIdxPtr := fs.Int("color", -1, "In 'apply' mode with -dpi: color index for -slot (0-15)")
	switchSlotPtr := fs.Bool("switch-slot", false, "In 'apply' mode with -dpi/-slot: also activate target slot")
	rgbSpeedPtr := fs.Int("speed", -1, "RGB Animation Speed (0-255)")
	dpi1Ptr := fs.String("dpi1", "", "Slot 1 value as DPI or DPI:Color (e.g. 800 or 800:3)")
	dpi2Ptr := fs.String("dpi2", "", "Slot 2 value as DPI or DPI:Color (e.g. 1200 or 1200:5)")
	dpi3Ptr := fs.String("dpi3", "", "Slot 3 value as DPI or DPI:Color (e.g. 1600 or 1600:7)")
	dpi4Ptr := fs.String("dpi4", "", "Slot 4 value as DPI or DPI:Color (e.g. 2000 or 2000:9)")
	color1Ptr := fs.Int("color1", -1, "Color index for slot 1 (0-15)")
	color2Ptr := fs.Int("color2", -1, "Color index for slot 2 (0-15)")
	color3Ptr := fs.Int("color3", -1, "Color index for slot 3 (0-15)")
	color4Ptr := fs.Int("color4", -1, "Color index for slot 4 (0-15)")
	activeSlotPtr := fs.Int("active-slot", -1, "Activate DPI slot (1-4)")
	ratePtr := fs.Int("rate", -1, "USB polling rate (125/250/500/1000)")
	rgbModePtr := fs.String("rgb-mode", "", "RGB mode (off/on/breath/cycle6/cycle12/cycle768)")
	cpiActionPtr := fs.String("cpi-action", "", "CPI button action")
	jsonPtr := fs.Bool("json", false, "JSON output for 'read' mode")
	regPtr := fs.Int("reg", -1, "Raw register address")
	regValPtr := fs.Int("regval", -1, "Raw register value")

	fs.Usage = func() {
		helpText := fmt.Sprintf(`Usage: %s -mode <command> [options]
		
Commands:
  read      Read current hardware configuration
  apply     Apply one or more settings
  dump      Dump bank 1 registers (1..30)
  write     Write a raw byte to a memory register (Advanced)

Options:
  -mode string
        Operation mode (required)
  -dpi string
        Value for -slot as DPI or DPI:Color (e.g. 800 or 800:1)
  -slot int
        Target slot for -dpi and -switch-slot (1-4) (default 1)
  -color int
        With -dpi: color index for -slot (0-15). -1 keeps current. (default -1)
  -switch-slot
     	With -dpi/-slot, also activate target DPI slot
  -speed int
        RGB Animation Speed (0-255). -1 keeps current. (default -1)
  -dpi1..-dpi4 string
        Slot value as DPI or DPI:Color (e.g. 800 or 800:3). Empty/-1 skips.
  -color1..-color4 int
        Color index per slot (0-15). -1 keeps current.
  -active-slot int
        Slot to activate (1-4). -1 skips.
  -rate int
        USB polling rate (125/250/500/1000). -1 skips.
  -rgb-mode string
        RGB mode (off/on/breath/cycle6/cycle12/cycle768)
  -cpi-action string
        CPI action value
  -json
        JSON output for 'read' mode
  -reg int
        Register address for 'write' mode (default -1)
  -regval int
        Register value for 'write' mode (default -1)

Arguments detail:
  [apply]
    -dpi <value>          : Apply to one slot selected by -slot. Format: DPI or DPI:Color
    -slot <1-4>           : Slot for -dpi and -switch-slot
    -color <0-15>         : Optional color for -dpi (if not in -dpi value)
    -switch-slot          : Activate -slot after applying -dpi
    -dpi1..-dpi4 <value>  : DPI or DPI:Color for slots 1..4 (e.g. 800 or 800:3), empty/-1 = skip
    -color1..-color4 <int>: Color for slots 1..4 (0..15), -1 = keep current
    -active-slot <1-4>    : Slot to activate after applying
    -rate <int>           : Polling rate (125/250/500/1000)
    -rgb-mode <string>    : off, on, breath, cycle6, cycle12, cycle768
    -speed <0-255>        : RGB speed (requires -rgb-mode)
    -cpi-action <string>  : Same action values as in [cpi]

Examples:
  %s -mode read
  %s -mode read -json
  %s -mode apply -dpi 800:1 -slot 1 -switch-slot
  %s -mode apply -rgb-mode on -rate 1000 -cpi-action vol_up
  %s -mode apply -dpi1 800:3 -dpi2 1200:5 -active-slot 2 -rate 1000 -rgb-mode breath -speed 40 -cpi-action vol_up
`, binName, binName, binName, binName, binName, binName)
		fmt.Fprint(errOut, helpText)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return facade.Command{}, true, nil
		}
		return facade.Command{}, false, err
	}

	if *modePtr == "" {
		fs.Usage()
		return facade.Command{}, true, nil
	}

	cmd := facade.Command{
		Mode: *modePtr,
		DPI:  [4]int{-1, -1, -1, -1},
		Color: [4]int{
			*color1Ptr,
			*color2Ptr,
			*color3Ptr,
			*color4Ptr,
		},
		ActiveSlot: *activeSlotPtr,
		RateHz:     *ratePtr,
		RGBMode:    *rgbModePtr,
		RGBSpeed:   *rgbSpeedPtr,
		CPIAction:  *cpiActionPtr,
		JSONOutput: *jsonPtr,
		Register:   *regPtr,
		RegisterV:  *regValPtr,
	}

	switch cmd.Mode {
	case "read", "apply", "dump", "write":
	default:
		return facade.Command{}, false, fmt.Errorf("unknown mode %q; use read, apply, dump, write", cmd.Mode)
	}

	dpiSpecs := [4]string{*dpi1Ptr, *dpi2Ptr, *dpi3Ptr, *dpi4Ptr}
	for i := 0; i < len(dpiSpecs); i++ {
		dpiValue, colorValue, hasValue, err := parseDPISpec(dpiSpecs[i])
		if err != nil {
			return facade.Command{}, false, fmt.Errorf("invalid dpi%d value %q: %w", i+1, dpiSpecs[i], err)
		}
		if !hasValue {
			continue
		}
		cmd.DPI[i] = dpiValue
		if colorValue < 0 {
			continue
		}
		if cmd.Color[i] >= 0 && cmd.Color[i] != colorValue {
			return facade.Command{}, false, fmt.Errorf("conflicting color for slot %d: dpi%d uses %d but color%d is %d", i+1, i+1, colorValue, i+1, cmd.Color[i])
		}
		cmd.Color[i] = colorValue
	}

	if *dpiSlotPtr < 1 || *dpiSlotPtr > 4 {
		return facade.Command{}, false, fmt.Errorf("invalid slot %d; must be 1-4", *dpiSlotPtr)
	}

	singleDPI, singleColor, hasSingleDPI, err := parseDPISpec(*dpiPtr)
	if err != nil {
		return facade.Command{}, false, fmt.Errorf("invalid dpi value %q: %w", *dpiPtr, err)
	}

	if hasSingleDPI || *colorIdxPtr >= 0 || *switchSlotPtr {
		slotIndex := *dpiSlotPtr - 1
		if hasSingleDPI {
			if cmd.DPI[slotIndex] >= 0 && cmd.DPI[slotIndex] != singleDPI {
				return facade.Command{}, false, fmt.Errorf("conflicting DPI for slot %d: -dpi gives %d but -dpi%d gives %d", *dpiSlotPtr, singleDPI, *dpiSlotPtr, cmd.DPI[slotIndex])
			}
			cmd.DPI[slotIndex] = singleDPI

			if singleColor >= 0 {
				if cmd.Color[slotIndex] >= 0 && cmd.Color[slotIndex] != singleColor {
					return facade.Command{}, false, fmt.Errorf("conflicting color for slot %d: -dpi gives %d but color for this slot is %d", *dpiSlotPtr, singleColor, cmd.Color[slotIndex])
				}
				cmd.Color[slotIndex] = singleColor
			}
		}

		if *colorIdxPtr >= 0 {
			if cmd.Color[slotIndex] >= 0 && cmd.Color[slotIndex] != *colorIdxPtr {
				return facade.Command{}, false, fmt.Errorf("conflicting color for slot %d: -color is %d but another source gives %d", *dpiSlotPtr, *colorIdxPtr, cmd.Color[slotIndex])
			}
			cmd.Color[slotIndex] = *colorIdxPtr
		}

		if *switchSlotPtr {
			if cmd.ActiveSlot >= 0 && cmd.ActiveSlot != *dpiSlotPtr {
				return facade.Command{}, false, fmt.Errorf("conflicting active slot: -switch-slot uses %d but -active-slot is %d", *dpiSlotPtr, cmd.ActiveSlot)
			}
			cmd.ActiveSlot = *dpiSlotPtr
		}
	}

	return cmd, false, nil
}

func parseDPISpec(spec string) (dpi int, color int, hasValue bool, err error) {
	normalized := strings.TrimSpace(spec)
	if normalized == "" || normalized == "-1" {
		return -1, -1, false, nil
	}

	parts := strings.Split(normalized, ":")
	if len(parts) > 2 {
		return 0, 0, false, errors.New("expected format <dpi> or <dpi>:<color>")
	}

	dpiValue, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false, errors.New("invalid dpi")
	}

	colorValue := -1
	if len(parts) == 2 {
		if parts[1] == "" {
			return 0, 0, false, errors.New("color is empty")
		}
		colorValue, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, false, errors.New("invalid color")
		}
	}

	return dpiValue, colorValue, true, nil
}
