package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"rtt3168ctl/internal/facade"
)

func Parse(args []string, binName string, errOut io.Writer) (facade.Command, bool, error) {
	fs := flag.NewFlagSet(binName, flag.ContinueOnError)
	fs.SetOutput(errOut)

	modePtr := fs.String("mode", "", "Operation mode")
	valPtr := fs.String("val", "", "Value for operation")
	dpiSlotPtr := fs.Int("slot", 1, "DPI Slot (1-4)")
	colorIdxPtr := fs.Int("color", -1, "Color index for DPI slot (0-15)")
	rgbSpeedPtr := fs.Int("speed", -1, "RGB Animation Speed (0-255)")
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
  -speed int
        RGB Animation Speed (0-255). -1 keeps current. (default -1)
  -reg int
        Register address for 'write' mode (default -1)
  -regval int
        Register value for 'write' mode (default -1)

Arguments detail:
  [dpi]
    -slot <1-4>   : Target DPI slot
    -val  <int>   : DPI speed (200 to 3200, step 200)
    -color <0-15> : LED color index

  [rgb]
    -val <string> : off, on, breath, cycle6, cycle12, cycle768

  [cpi]
    -val <string> : backward, forward, ctrl, win, browser, double_click, sniper, rgb_switch, dpi_cycle,
					play_pause, mute, next_track, prev_track, stop, vol_up, vol_down, win_d,
					copy, paste, prev_page, next_page, my_computer, calculator, ctrl_w

Examples:
  %s -mode read
  %s -mode dpi -slot 2 -val 1200 -color 5
  %s -mode cpi -val vol_up
  %s -mode rate -val 1000
`, binName, binName, binName, binName, binName)
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

	return facade.Command{
		Mode:       *modePtr,
		Value:      *valPtr,
		DPISlot:    *dpiSlotPtr,
		ColorIndex: *colorIdxPtr,
		RGBSpeed:   *rgbSpeedPtr,
		Register:   *regPtr,
		RegisterV:  *regValPtr,
	}, false, nil
}
