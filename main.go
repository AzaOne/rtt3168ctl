package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/gousb"
)

// USB Request Types & Constants
const (
	VID = 0x093A // 2362
	PID = 0x2533 // 9523

	ReqTypeWrite = 0x40 // Vendor | Device | Out
	ReqTypeRead  = 0xC0 // Vendor | Device | In

	// Register Addresses (Bank 1)
	RegRGBSpeed  = 1
	RegDPI1      = 2
	RegDPI2      = 3
	RegDPI3      = 4
	RegDPI4      = 5
	RegDPI5      = 6 // Buggy
	RegDPI6      = 7 // Buggy
	RegDPISelect = 9
	RegRGBMode   = 10
	RegCPIButton = 11
	RegRate      = 14
	RegSensorID  = 29 // Sensor Identification Register

	// Polling Rate Values
	Rate125  = 194
	Rate250  = 130
	Rate500  = 66
	Rate1000 = 2

	// RGB Mode Base Values
	RGB_AlwaysOn  = 0x01
	RGB_Breathing = 0x41
	RGB_Cycle6    = 0x61
	RGB_Cycle12   = 0x81
	RGB_Cycle768  = 0xA1
	RGB_Off       = 0xE1
)

// CPI Hardware Mappings
var cpiActionMap = map[string]uint8{
	"backward":     224, "forward":      225,
	"ctrl":         226, "win":          227,
	"browser":      228, "double_click": 229,
	"sniper":       230, "rgb_switch":   231,
	"dpi_cycle":    232, "play_pause":   236,
	"mute":         237, "next_track":   238,
	"prev_track":   239, "stop":         240,
	"vol_up":       242, "vol_down":     243,
	"win_d":        245, "copy":         246,
	"paste":        247, "prev_page":    248,
	"next_page":    249, "my_computer":  250,
	"calculator":   251, "ctrl_w":       252,
}

func main() {
	modePtr := flag.String("mode", "", "Operation mode")
	valPtr := flag.String("val", "", "Value for operation")
	dpiSlotPtr := flag.Int("slot", 1, "DPI Slot (1-6)")
	colorIdxPtr := flag.Int("color", -1, "Color index for DPI slot (0-15)")
	rgbSpeedPtr := flag.Int("speed", -1, "RGB Animation Speed (0-255)")
	regPtr := flag.Int("reg", -1, "Raw register address")
	regValPtr := flag.Int("regval", -1, "Raw register value")

	flag.Usage = func() {
		binName := os.Args[0]
		helpText := fmt.Sprintf(`Mouse Hardware Configuration Tool

Usage: %s -mode <command> [options]

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
        Target DPI slot (1-6) (default 1)
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
    -slot <1-6>   : Target DPI slot (Slots 5 and 6 are hidden hardware slots)
    -val  <int>   : DPI speed (200 to 3200, step 200)
    -color <0-15> : LED color index

  [rgb]
    -val <string> : off, on, breath, cycle6, cycle12, cycle768

  [cpi]
    -val <string> : vol_up, vol_down, mute, play_pause, next_track, prev_track,
                    stop, copy, paste, ctrl, win, sniper, dpi_cycle, rgb_switch

Examples:
  %s -mode read
  %s -mode dpi -slot 2 -val 1200 -color 5
  %s -mode cpi -val vol_up
  %s -mode rate -val 1000
`, binName, binName, binName, binName, binName)
		fmt.Fprint(os.Stderr, helpText)
	}

	flag.Parse()

	if *modePtr == "" {
		flag.Usage()
		return
	}

	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(VID, PID)
	if err != nil || dev == nil {
		log.Fatalf("Error: Could not open device. Ensure you are running as root/sudo.\nDetails: %v", err)
	}
	defer dev.Close()
	dev.SetAutoDetach(true)

	// Unlock & Enter Bank 1
	sendControlEx(dev, ReqTypeWrite, 1, 0, 0)
	sendControlEx(dev, ReqTypeWrite, 1, 0x0100, 23049)
	sendControlEx(dev, ReqTypeWrite, 1, 0x0100, 383)

	switch *modePtr {
	case "read":
		printStatus(dev)
	case "dump":
		handleDump(dev)
	case "write":
		if *regPtr < 0 || *regValPtr < 0 || *regValPtr > 255 {
			log.Fatal("Error: Invalid register or value for write mode.")
		}
		handleWrite(dev, uint16(*regPtr), uint8(*regValPtr))
	case "switch":
		handleSwitch(dev, *dpiSlotPtr)
	case "rgb":
		handleRGB(dev, *valPtr, *rgbSpeedPtr)
	case "rate":
		handleRate(dev, *valPtr)
	case "dpi":
		handleDPI(dev, *dpiSlotPtr, *valPtr, *colorIdxPtr)
	case "cpi":
		if *valPtr == "" {
			log.Fatal("Error: Provide action for CPI.")
		}
		handleCPI(dev, *valPtr)
	default:
		fmt.Println("Error: Unknown mode. Use -h for help.")
	}

	// Lock & Save
	sendControlEx(dev, ReqTypeWrite, 1, 0x0100, 127) // Enter Bank 0
	sendControlEx(dev, ReqTypeWrite, 1, 0x0100, 9)   // Enable Protect
	sendControlEx(dev, ReqTypeWrite, 6, 0, 0)        // Exit Sensor Mode
}

// =========================================================================

func printStatus(dev *gousb.Device) {
	fmt.Println("Mouse Hardware Status")

	// Switch to Bank 0 to read active slot
	sendControlEx(dev, ReqTypeWrite, 1, 0x0100, 127)
	activeVal := readRegister(dev, 2)
	activeSlot := (activeVal & 0x0F) + 1

	// Switch back to Bank 1
	sendControlEx(dev, ReqTypeWrite, 1, 0x0100, 383)

	// Sensor ID
	sensorID := readRegister(dev, RegSensorID)
	fmt.Printf("Sensor ID:         0x%02X\n", sensorID)
	fmt.Printf("Active DPI Slot:   %d\n", activeSlot)

	// Read all 6 slots (including secret ones)
	for i := 1; i <= 6; i++ {
		regID := uint16(i + 1)
		val := readRegister(dev, regID)
		dpiReal := 200 + (int(val&0x0F) * 200)
		colorIdx := (val & 0xF0) >> 4
		marker := " "
		if i == int(activeSlot) {
			marker = "*"
		}
		fmt.Printf("%s Slot %d: %4d DPI (Color: %2d, Raw: 0x%02X)\n", marker, i, dpiReal, colorIdx, val)
	}

	rateVal := readRegister(dev, RegRate)
	rateHz := "Unknown"
	switch rateVal {
	case Rate125:
		rateHz = "125Hz"
	case Rate250:
		rateHz = "250Hz"
	case Rate500:
		rateHz = "500Hz"
	case Rate1000:
		rateHz = "1000Hz"
	}
	fmt.Printf("Polling Rate:      %s\n", rateHz)

	rgbVal := readRegister(dev, RegRGBMode)
	rgbName := "Unknown"
	switch rgbVal & 0xF0 {
	case 0x40:
		rgbName = "Breathing"
	case 0xE0:
		rgbName = "Off"
	case 0x00:
		rgbName = "Always On"
	case 0x60:
		rgbName = "6 Color Cycle"
	case 0x80:
		rgbName = "12 Color Cycle"
	case 0xA0:
		rgbName = "768 Color Cycle"
	}
	fmt.Printf("RGB Mode:          %s (Speed: %d)\n", rgbName, readRegister(dev, RegRGBSpeed))

	cpiVal := readRegister(dev, RegCPIButton)
	cpiActionName := "Unknown"
	for name, code := range cpiActionMap {
		if code == cpiVal {
			cpiActionName = name
			break
		}
	}
	fmt.Printf("CPI Button:        %s (Raw: 0x%02X)\n", cpiActionName, cpiVal)
}

func handleCPI(dev *gousb.Device, action string) {
	val, ok := cpiActionMap[action]
	if !ok {
		log.Fatalf("Invalid CPI action: '%s'. Run with -h for valid options.", action)
	}
	writeRegister(dev, RegCPIButton, val)
	fmt.Printf("CPI Button bound to: %s (0x%02X)\n", action, val)
}

func handleDump(dev *gousb.Device) {
	fmt.Println("Memory Dump (Bank 1)")
	// Dump up to 30 to see Sensor ID
	for i := uint16(1); i <= 30; i++ {
		val := readRegister(dev, i)
		fmt.Printf("%02d: 0x%02X\n", i, val)
	}
}

func handleWrite(dev *gousb.Device, regID uint16, value uint8) {
	writeRegister(dev, regID, value)
	fmt.Printf("Written 0x%02X to Register %d\n", value, regID)
}

func handleSwitch(dev *gousb.Device, slot int) {
	if slot < 1 || slot > 6 {
		log.Fatal("Error: Slot must be 1-6")
	}
	// Note: Switching active slot is done by writing to RegDPISelect (9)
	// Value is (slot-1) * 32.
	// Slot 1=0x00, Slot 2=0x20, Slot 3=0x40, Slot 4=0x60
	// It is assumed slots 5 and 6 follow this pattern (0x80, 0xA0)
	writeRegister(dev, RegDPISelect, uint8((slot-1)*32))
	fmt.Printf("Activated DPI Slot %d\n", slot)
}

func handleRGB(dev *gousb.Device, val string, speed int) {
	var base uint8
	switch val {
	case "off":
		base = RGB_Off
	case "on":
		base = RGB_AlwaysOn
	case "breath":
		base = RGB_Breathing
	case "cycle6":
		base = RGB_Cycle6
	case "cycle12":
		base = RGB_Cycle12
	case "cycle768":
		base = RGB_Cycle768
	default:
		log.Fatal("Error: Invalid RGB value")
	}
	writeRegister(dev, RegRGBMode, (base&0xF0)|(readRegister(dev, RegRGBMode)&0x0F))
	if speed >= 0 {
		writeRegister(dev, RegRGBSpeed, uint8(speed))
	}
	fmt.Printf("RGB Mode set to %s\n", val)
}

func handleRate(dev *gousb.Device, val string) {
	var rate uint8
	switch val {
	case "125":
		rate = Rate125
	case "250":
		rate = Rate250
	case "500":
		rate = Rate500
	case "1000":
		rate = Rate1000
	default:
		log.Fatal("Error: Invalid Rate. Use 125, 250, 500, or 1000.")
	}
	writeRegister(dev, RegRate, rate)
	fmt.Printf("Polling rate set to %sHz\n", val)
}

func handleDPI(dev *gousb.Device, slot int, val string, colorIdx int) {
	if slot < 1 || slot > 6 {
		log.Fatalf("Error: Invalid DPI Slot %d. Must be 1-6.", slot)
	}

	dpiInt, err := strconv.Atoi(val)
	if err != nil || dpiInt < 200 || dpiInt > 3200 || dpiInt%200 != 0 {
		log.Fatalf("Error: Invalid DPI %s. Must be between 200 and 3200, in steps of 200.", val)
	}

	dpiIdx := uint8((dpiInt - 200) / 200)
	target := uint16(slot + 1)

	col := uint8(colorIdx)
	if colorIdx < 0 {
		col = (readRegister(dev, target) & 0xF0) >> 4
	}

	newVal := (col << 4) | dpiIdx
	writeRegister(dev, target, newVal)

	// Auto-activate this slot so the user sees the change immediately
	writeRegister(dev, RegDPISelect, uint8((slot-1)*32))

	fmt.Printf("DPI Slot %d set to %d (Color: %d)\n", slot, dpiInt, col)
}

// =========================================================================
// LOW-LEVEL USB OPERATIONS
// =========================================================================

func readRegister(dev *gousb.Device, regID uint16) uint8 {
	buf := make([]byte, 1)
	dev.Control(ReqTypeRead, 1, 0x0100, regID, buf)
	return buf[0]
}

func writeRegister(dev *gousb.Device, regID uint16, value uint8) {
	dev.Control(ReqTypeWrite, 1, 0x0100, (uint16(value)<<8)|regID, nil)
	time.Sleep(20 * time.Millisecond)
}

func sendControlEx(dev *gousb.Device, reqType, req uint8, val, idx uint16) {
	dev.Control(reqType, req, val, idx, nil)
}
