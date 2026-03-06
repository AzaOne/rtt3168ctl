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
	valPtr := fs.String("val", "", "Value for operation")
	dpiSlotPtr := fs.Int("slot", 1, "DPI Slot (1-4)")
	colorIdxPtr := fs.Int("color", -1, "Color index for DPI slot (0-15)")
	switchSlotPtr := fs.Bool("switch-slot", false, "In 'dpi' mode, also activate target DPI slot")
	rgbSpeedPtr := fs.Int("speed", -1, "RGB Animation Speed (0-255)")
	dpi1Ptr := fs.String("dpi1", "", "In 'apply' mode: slot 1 value as DPI or DPI:Color (e.g. 800 or 800:3)")
	dpi2Ptr := fs.String("dpi2", "", "In 'apply' mode: slot 2 value as DPI or DPI:Color (e.g. 1200 or 1200:5)")
	dpi3Ptr := fs.String("dpi3", "", "In 'apply' mode: slot 3 value as DPI or DPI:Color (e.g. 1600 or 1600:7)")
	dpi4Ptr := fs.String("dpi4", "", "In 'apply' mode: slot 4 value as DPI or DPI:Color (e.g. 2000 or 2000:9)")
	color1Ptr := fs.Int("color1", -1, "In 'apply' mode: Color index for slot 1 (0-15)")
	color2Ptr := fs.Int("color2", -1, "In 'apply' mode: Color index for slot 2 (0-15)")
	color3Ptr := fs.Int("color3", -1, "In 'apply' mode: Color index for slot 3 (0-15)")
	color4Ptr := fs.Int("color4", -1, "In 'apply' mode: Color index for slot 4 (0-15)")
	activeSlotPtr := fs.Int("active-slot", -1, "In 'apply' mode: activate DPI slot (1-4)")
	ratePtr := fs.Int("rate", -1, "In 'apply' mode: USB polling rate (125/250/500/1000)")
	rgbModePtr := fs.String("rgb-mode", "", "In 'apply' mode: RGB mode (off/on/breath/cycle6/cycle12/cycle768)")
	cpiActionPtr := fs.String("cpi-action", "", "In 'apply' mode: CPI button action")
	jsonPtr := fs.Bool("json", false, "JSON output for 'read' mode")
	regPtr := fs.Int("reg", -1, "Raw register address")
	regValPtr := fs.Int("regval", -1, "Raw register value")

	fs.Usage = func() {
		helpText := fmt.Sprintf(`Usage: %s -mode <command> [options]
		
Commands:
  read      Read current hardware configuration
  dpi       Configure a specific DPI slot (speed and color)
  switch    Switch active DPI slot
  rgb       Configure RGB lighting mode and animation speed
  rate      Set USB polling rate (Hz)
  cpi       Bind a hardware action to the CPI button (Button 6)
  apply     Apply multiple settings in one run (good for cron/startup)
  dump      Print raw hex dump of Bank 1 memory
  write     Write a raw byte to a memory register (Advanced)

Options:
  -mode string
        Operation mode (required)
  -val string
        Value for operation (DPI speed, RGB mode, etc.)
  -slot int
        Target DPI slot (1-4) (default 1)
  -color int
        Color index for DPI slot (0-15). -1 keeps current. (default -1)
  -switch-slot
        In 'dpi' mode, also activate target DPI slot
  -speed int
        RGB Animation Speed (0-255). -1 keeps current. (default -1)
  -dpi1..-dpi4 string
        In 'apply' mode: slot value as DPI or DPI:Color (e.g. 800 or 800:3). Empty/-1 skips.
  -color1..-color4 int
        In 'apply' mode: color index per slot (0-15). -1 keeps current.
  -active-slot int
        In 'apply' mode: slot to activate (1-4). -1 skips.
  -rate int
        In 'apply' mode: USB polling rate (125/250/500/1000). -1 skips.
  -rgb-mode string
        In 'apply' mode: RGB mode (off/on/breath/cycle6/cycle12/cycle768)
  -cpi-action string
        In 'apply' mode: CPI action value
  -json
        JSON output for 'read' mode
  -reg int
        Register address for 'write' mode (default -1)
  -regval int
        Register value for 'write' mode (default -1)

Arguments detail:
  [dpi]
    -slot <1-4>   : Target DPI slot
    -val  <int>   : DPI speed (200 to 3200, step 200)
    -color <0-15> : LED color index
    -switch-slot  : Also activate this slot after write

  [rgb]
    -val <string> : off, on, breath, cycle6, cycle12, cycle768

  [cpi]
    -val <string> : backward, forward, ctrl, win, browser, double_click, sniper, rgb_switch, dpi_cycle,
					play_pause, mute, next_track, prev_track, stop, vol_up, vol_down, win_d,
					copy, paste, prev_page, next_page, my_computer, calculator, ctrl_w

  [apply]
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
  %s -mode dpi -slot 2 -val 1200 -color 5
  %s -mode cpi -val vol_up
  %s -mode rate -val 1000
  %s -mode apply -dpi1 800:3 -dpi2 1200:5 -active-slot 2 -rate 1000 -rgb-mode breath -speed 40 -cpi-action vol_up
`, binName, binName, binName, binName, binName, binName, binName)
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

	switch cmd.Mode {
	case "dpi":
		if *dpiSlotPtr < 1 || *dpiSlotPtr > 4 {
			return facade.Command{}, false, fmt.Errorf("invalid DPI slot %d; must be 1-4", *dpiSlotPtr)
		}
		if *valPtr == "" {
			return facade.Command{}, false, errors.New("dpi mode requires -val")
		}
		dpiValue, err := strconv.Atoi(*valPtr)
		if err != nil {
			return facade.Command{}, false, fmt.Errorf("invalid DPI value %q", *valPtr)
		}
		cmd.DPI[*dpiSlotPtr-1] = dpiValue
		cmd.Color[*dpiSlotPtr-1] = *colorIdxPtr
		if *switchSlotPtr {
			cmd.ActiveSlot = *dpiSlotPtr
		}
	case "switch":
		if cmd.ActiveSlot < 0 {
			cmd.ActiveSlot = *dpiSlotPtr
		}
	case "rgb":
		if *valPtr != "" {
			if cmd.RGBMode != "" && cmd.RGBMode != *valPtr {
				return facade.Command{}, false, errors.New("rgb mode is ambiguous; use either -val or -rgb-mode")
			}
			cmd.RGBMode = *valPtr
		}
	case "rate":
		if *valPtr != "" {
			rateValue, err := strconv.Atoi(*valPtr)
			if err != nil {
				return facade.Command{}, false, fmt.Errorf("invalid rate value %q", *valPtr)
			}
			if cmd.RateHz >= 0 && cmd.RateHz != rateValue {
				return facade.Command{}, false, errors.New("rate is ambiguous; use either -val or -rate")
			}
			cmd.RateHz = rateValue
		}
	case "cpi":
		if *valPtr != "" {
			if cmd.CPIAction != "" && cmd.CPIAction != *valPtr {
				return facade.Command{}, false, errors.New("cpi action is ambiguous; use either -val or -cpi-action")
			}
			cmd.CPIAction = *valPtr
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
