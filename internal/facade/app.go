package facade

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"rtt3168ctl/internal/core/kernel"
	"rtt3168ctl/internal/modules/mouse"
)

type Command struct {
	Mode       string
	Value      string
	DPISlot    int
	ColorIndex int
	SwitchSlot bool
	RGBSpeed   int
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
		return errors.Join(workErr, endErr)
	})
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
	case "switch":
		if err := svc.SwitchDPISlot(cmd.DPISlot); err != nil {
			return err
		}
		fmt.Fprintf(out, "Activated DPI Slot %d\n", cmd.DPISlot)
		return nil
	case "rgb":
		if err := svc.SetRGB(cmd.Value, cmd.RGBSpeed); err != nil {
			return err
		}
		fmt.Fprintf(out, "RGB Mode set to %s\n", cmd.Value)
		return nil
	case "rate":
		rateHz, err := strconv.Atoi(cmd.Value)
		if err != nil {
			return fmt.Errorf("invalid rate value %q", cmd.Value)
		}
		if err := svc.SetRate(rateHz); err != nil {
			return err
		}
		fmt.Fprintf(out, "Polling rate set to %dHz\n", rateHz)
		return nil
	case "dpi":
		dpi, err := strconv.Atoi(cmd.Value)
		if err != nil {
			return fmt.Errorf("invalid DPI value %q", cmd.Value)
		}
		if err := svc.SetDPI(cmd.DPISlot, dpi, cmd.ColorIndex, cmd.SwitchSlot); err != nil {
			return err
		}
		switchText := ""
		if cmd.SwitchSlot {
			switchText = ", active slot switched"
		}
		if cmd.ColorIndex < 0 {
			fmt.Fprintf(out, "DPI Slot %d set to %d (Color: unchanged%s)\n", cmd.DPISlot, dpi, switchText)
			return nil
		}
		fmt.Fprintf(out, "DPI Slot %d set to %d (Color: %d%s)\n", cmd.DPISlot, dpi, cmd.ColorIndex, switchText)
		return nil
	case "cpi":
		if cmd.Value == "" {
			return errors.New("provide action for CPI")
		}
		if err := svc.SetCPIAction(cmd.Value); err != nil {
			return err
		}
		fmt.Fprintf(out, "CPI Button bound to: %s\n", cmd.Value)
		return nil
	default:
		return fmt.Errorf("unknown mode %q; use -h for help", cmd.Mode)
	}
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
