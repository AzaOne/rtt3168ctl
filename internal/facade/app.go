package facade

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"rtt3168ctl/internal/core/kernel"
	"rtt3168ctl/internal/modules/mouse"
)

type Command struct {
	Mode  string
	DPI   [4]int
	Color [4]int

	ActiveSlot int
	RateHz     int
	RGBMode    string
	RGBSpeed   int
	CPIAction  string

	JSONOutput bool
	Register   int
	RegisterV  int
}

type App struct {
	kernel *kernel.Kernel
}

func New(k *kernel.Kernel) *App {
	return &App{kernel: k}
}

func (a *App) Execute(cmd Command, out io.Writer) error {
	if cmd.Mode == "" {
		return errors.New("mode is required")
	}

	return a.kernel.Run(func(dev kernel.Device) error {
		repo := mouse.NewRepository(dev)
		svc := mouse.NewService(repo)

		if err := svc.BeginSession(); err != nil {
			return fmt.Errorf("begin session: %w", err)
		}

		workErr := executeMode(svc, cmd, out)
		endErr := svc.EndSession()
		if endErr != nil {
			endErr = fmt.Errorf("end session: %w", endErr)
		}

		if shouldSuppressEndSessionError(cmd, workErr, endErr) {
			fmt.Fprintf(out, "Warning: %v\n", endErr)
			endErr = nil
		}

		return errors.Join(workErr, endErr)
	})
}

func shouldSuppressEndSessionError(cmd Command, workErr, endErr error) bool {
	if endErr == nil || workErr != nil {
		return false
	}
	if cmd.Mode != "apply" || cmd.RateHz < 0 {
		return false
	}
	return isTransientUSBError(endErr)
}

func isTransientUSBError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "libusb: i/o error") || strings.Contains(msg, "libusb: pipe error")
}

func executeMode(svc *mouse.Service, cmd Command, out io.Writer) error {
	switch cmd.Mode {
	case "read":
		status, err := svc.ReadStatus()
		if err != nil {
			return err
		}
		if cmd.JSONOutput {
			return printStatusJSON(out, status)
		}
		printStatus(out, status)
		return nil
	case "dump":
		dump, err := svc.DumpRegisters(1, 30)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "Memory Dump (Bank 1)")
		for _, item := range dump {
			fmt.Fprintf(out, "%02d: 0x%02X\n", item.Register, item.Value)
		}
		return nil
	case "write":
		if cmd.Register < 0 || cmd.RegisterV < 0 || cmd.RegisterV > 255 {
			return errors.New("invalid register or value for write mode")
		}
		if err := svc.WriteRaw(uint16(cmd.Register), uint8(cmd.RegisterV)); err != nil {
			return err
		}
		fmt.Fprintf(out, "Written 0x%02X to Register %d\n", cmd.RegisterV, cmd.Register)
		return nil
	case "apply":
		return applyAllSettings(svc, cmd, out)
	default:
		return fmt.Errorf("unknown mode %q; use read, apply, dump, write", cmd.Mode)
	}
}

func applyAllSettings(svc *mouse.Service, cmd Command, out io.Writer) error {
	applied := false

	for slot := 1; slot <= 4; slot++ {
		dpi := cmd.DPI[slot-1]
		color := cmd.Color[slot-1]
		if dpi < 0 && color < 0 {
			continue
		}
		if dpi < 0 {
			return fmt.Errorf("apply: dpi%d is required when color%d is set", slot, slot)
		}
		if err := svc.SetDPI(slot, dpi, color, false); err != nil {
			return fmt.Errorf("apply slot %d: %w", slot, err)
		}
		applied = true
		if color < 0 {
			fmt.Fprintf(out, "DPI Slot %d set to %d (Color: unchanged)\n", slot, dpi)
			continue
		}
		fmt.Fprintf(out, "DPI Slot %d set to %d (Color: %d)\n", slot, dpi, color)
	}

	if cmd.RGBMode != "" || cmd.RGBSpeed >= 0 {
		if cmd.RGBMode == "" {
			return errors.New("apply: rgb-mode is required when speed is set")
		}
		if err := svc.SetRGB(cmd.RGBMode, cmd.RGBSpeed); err != nil {
			return fmt.Errorf("apply RGB: %w", err)
		}
		applied = true
		if cmd.RGBSpeed >= 0 {
			fmt.Fprintf(out, "RGB Mode set to %s (Speed: %d)\n", cmd.RGBMode, cmd.RGBSpeed)
		} else {
			fmt.Fprintf(out, "RGB Mode set to %s\n", cmd.RGBMode)
		}
	}

	if cmd.CPIAction != "" {
		if err := svc.SetCPIAction(cmd.CPIAction); err != nil {
			return fmt.Errorf("apply CPI: %w", err)
		}
		applied = true
		fmt.Fprintf(out, "CPI Button bound to: %s\n", cmd.CPIAction)
	}

	if cmd.ActiveSlot >= 0 {
		if err := svc.SwitchDPISlot(cmd.ActiveSlot); err != nil {
			return fmt.Errorf("apply active slot: %w", err)
		}
		applied = true
		fmt.Fprintf(out, "Activated DPI Slot %d\n", cmd.ActiveSlot)
	}

	if cmd.RateHz >= 0 {
		if err := svc.SetRate(cmd.RateHz); err != nil {
			return fmt.Errorf("apply rate: %w", err)
		}
		applied = true
		fmt.Fprintf(out, "Polling rate set to %dHz\n", cmd.RateHz)
	}

	if !applied {
		return errors.New("apply: no settings provided")
	}

	return nil
}

func printStatus(out io.Writer, status mouse.Status) {
	fmt.Fprintf(out, "Sensor ID:         0x%02X\n", status.SensorID)
	fmt.Fprintf(out, "Active DPI Slot:   %d\n", status.ActiveSlot)

	for _, slot := range status.Slots {
		marker := "[ ]"
		if slot.Active {
			marker = "[+]"
		}
		fmt.Fprintf(out, "%s Slot %d: %4d DPI (Color: %2d, Raw: 0x%02X)\n", marker, slot.Slot, slot.DPI, slot.Color, slot.Raw)
	}

	fmt.Fprintf(out, "Polling Rate:      %s\n", status.Rate)
	fmt.Fprintf(out, "RGB Mode:          %s (Speed: %d)\n", status.RGBMode, status.RGBSpeed)
	fmt.Fprintf(out, "CPI Button:        %s (Raw: 0x%02X)\n", status.CPIAction, status.CPIRaw)
}

type jsonStatus struct {
	SensorID   uint8            `json:"sensor_id"`
	ActiveSlot int              `json:"active_slot"`
	Slots      []jsonStatusSlot `json:"slots"`
	Rate       string           `json:"rate"`
	RGBMode    string           `json:"rgb_mode"`
	RGBSpeed   uint8            `json:"rgb_speed"`
	CPIAction  string           `json:"cpi_action"`
	CPIRaw     uint8            `json:"cpi_raw"`
}

type jsonStatusSlot struct {
	Slot   int   `json:"slot"`
	DPI    int   `json:"dpi"`
	Color  int   `json:"color"`
	Raw    uint8 `json:"raw"`
	Active bool  `json:"active"`
}

func printStatusJSON(out io.Writer, status mouse.Status) error {
	jsonSlots := make([]jsonStatusSlot, 0, len(status.Slots))
	for _, slot := range status.Slots {
		jsonSlots = append(jsonSlots, jsonStatusSlot{
			Slot:   slot.Slot,
			DPI:    slot.DPI,
			Color:  slot.Color,
			Raw:    slot.Raw,
			Active: slot.Active,
		})
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonStatus{
		SensorID:   status.SensorID,
		ActiveSlot: status.ActiveSlot,
		Slots:      jsonSlots,
		Rate:       status.Rate,
		RGBMode:    status.RGBMode,
		RGBSpeed:   status.RGBSpeed,
		CPIAction:  status.CPIAction,
		CPIRaw:     status.CPIRaw,
	})
}
