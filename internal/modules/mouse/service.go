package mouse

import (
	"errors"
	"fmt"
	"time"
)

type SlotStatus struct {
	Slot   int
	DPI    int
	Color  int
	Raw    uint8
	Active bool
}


type Status struct {
	SensorID   uint8
	ActiveSlot int
	Slots      []SlotStatus
	Rate       string
	RGBMode    string
	RGBSpeed   uint8
	CPIAction  string
	CPIRaw     uint8
}

type RegisterValue struct {
	Register uint16
	Value    uint8
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) BeginSession() error {
	return errors.Join(
		s.repo.SendControl(ReqTypeWrite, ReqCodeControl, 0, 0),
		s.repo.SendControl(ReqTypeWrite, ReqCodeControl, ControlValDefault, ControlIdxUnlock),
		s.enterBank(1),
	)
}

func (s *Service) EndSession() error {
	return errors.Join(
		s.enterBank0(),
		s.repo.SendControl(ReqTypeWrite, ReqCodeControl, ControlValDefault, ControlIdxLock),
		s.repo.SendControl(ReqTypeWrite, ReqCodeReset, 0, 0),
	)
}

func (s *Service) ReadStatus() (Status, error) {
	if err := s.enterBank0(); err != nil {
		return Status{}, err
	}
	activeVal, err := s.repo.ReadRegister(2)
	if err != nil {
		return Status{}, err
	}
	activeSlot := decodeActiveSlot(activeVal)

	if err := s.enterBank1(); err != nil {
		return Status{}, err
	}

	sensorID, err := s.repo.ReadRegister(RegSensorID)
	if err != nil {
		return Status{}, err
	}

	slots := make([]SlotStatus, 0, 4)
	for i := 1; i <= 4; i++ {
		regID := uint16(i + 1)
		val, err := s.repo.ReadRegister(regID)
		if err != nil {
			return Status{}, err
		}
		slots = append(slots, SlotStatus{
			Slot:   i,
			DPI:    200 + (int(val&0x0F) * 200),
			Color:  int((val & 0xF0) >> 4),
			Raw:    val,
			Active: i == activeSlot,
		})
	}

	rateVal, err := s.repo.ReadRegister(RegRate)
	if err != nil {
		return Status{}, err
	}

	rgbVal, err := s.repo.ReadRegister(RegRGBMode)
	if err != nil {
		return Status{}, err
	}

	rgbSpeed, err := s.repo.ReadRegister(RegRGBSpeed)
	if err != nil {
		return Status{}, err
	}

	cpiVal, err := s.repo.ReadRegister(RegCPIButton)
	if err != nil {
		return Status{}, err
	}

	return Status{
		SensorID:   sensorID,
		ActiveSlot: activeSlot,
		Slots:      slots,
		Rate:       decodeRate(rateVal),
		RGBMode:    decodeRGBMode(rgbVal),
		RGBSpeed:   rgbSpeed,
		CPIAction:  decodeCPIAction(cpiVal),
		CPIRaw:     cpiVal,
	}, nil
}


func (s *Service) DumpRegisters(start, end uint16) ([]RegisterValue, error) {
	if end < start {
		return nil, fmt.Errorf("invalid dump range: %d-%d", start, end)
	}

	out := make([]RegisterValue, 0, end-start+1)
	for i := start; i <= end; i++ {
		val, err := s.repo.ReadRegister(i)
		if err != nil {
			return nil, err
		}
		out = append(out, RegisterValue{Register: i, Value: val})
	}
	return out, nil
}

func (s *Service) DumpBank0Registers(start, end uint16) ([]RegisterValue, error) {
	return s.DumpBankRegisters(0, start, end)
}

func (s *Service) DumpBank1Registers(start, end uint16) ([]RegisterValue, error) {
	return s.DumpBankRegisters(1, start, end)
}

func (s *Service) DumpBankRegisters(bank, start, end uint16) ([]RegisterValue, error) {
	if err := s.enterBank(bank); err != nil {
		return nil, err
	}
	return s.DumpRegisters(start, end)
}

func (s *Service) WriteRaw(regID uint16, value uint8) error {
	return s.repo.WriteRegister(regID, value)
}

func (s *Service) SwitchDPISlot(slot int) error {
	if slot < 1 || slot > 4 {
		return fmt.Errorf("slot must be in range 1-4")
	}
	return s.repo.WriteRegister(RegDPISelect, uint8((slot-1)*32))
}

func (s *Service) SetRGB(mode string, speed int) error {
	var base uint8
	switch mode {
	case "off":
		base = RGBOff
	case "on":
		base = RGBAlwaysOn
	case "breath":
		base = RGBBreath
	case "breath_segment", "segment", "breath_seg":
		base = RGBBreathSegment
	case "cycle6":
		base = RGBCycle6
	case "cycle12":
		base = RGBCycle12
	case "cycle768":
		base = RGBCycle768
	default:
		return fmt.Errorf("invalid RGB mode %q; use off, on, breath, breath_segment, cycle6, cycle12, or cycle768", mode)
	}

	currentRGB, err := s.repo.ReadRegister(RegRGBMode)
	if err != nil {
		return err
	}
	if err := s.repo.WriteRegister(RegRGBMode, (base&0xF0)|(currentRGB&0x0F)); err != nil {
		return err
	}

	if speed >= 0 {
		if speed > 255 {
			return fmt.Errorf("RGB speed must be between 0 and 255")
		}
		if err := s.repo.WriteRegister(RegRGBSpeed, uint8(speed)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) SetRate(rateHz int) error {
	var rate uint8
	switch rateHz {
	case 125:
		rate = Rate125
	case 250:
		rate = Rate250
	case 500:
		rate = Rate500
	case 1000:
		rate = Rate1000
	default:
		return fmt.Errorf("invalid rate %d; use 125, 250, 500, or 1000", rateHz)
	}

	if rateHz != 1000 {
		return s.repo.WriteRegister(RegRate, rate)
	}

	// Some devices briefly stall when switching to 1000Hz.
	// Retry with small backoff and re-sync control before next attempt.
	time.Sleep(60 * time.Millisecond)

	var lastErr error
	backoffs := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
	}
	for attempt := 0; ; attempt++ {
		err := s.repo.WriteRegister(RegRate, rate)
		if err == nil {
			return nil
		}
		lastErr = err

		if attempt >= len(backoffs) {
			break
		}

		if syncErr := s.enterBank1(); syncErr != nil {
			lastErr = errors.Join(lastErr, fmt.Errorf("resync before retry %d: %w", attempt+1, syncErr))
		}
		time.Sleep(backoffs[attempt])
	}

	return fmt.Errorf("set 1000Hz with retries: %w", lastErr)
}

func (s *Service) SetDPI(slot, dpi, colorIdx int, switchSlot bool) error {
	if slot < 1 || slot > 4 {
		return fmt.Errorf("invalid DPI slot %d; must be 1-4", slot)
	}
	if dpi < 200 || dpi > 3200 || dpi%200 != 0 {
		return fmt.Errorf("invalid DPI %d; must be 200..3200 in steps of 200", dpi)
	}
	if colorIdx > 15 {
		return fmt.Errorf("invalid color index %d; must be 0..15", colorIdx)
	}

	dpiIdx := uint8((dpi - 200) / 200)
	target := uint16(slot + 1)

	color := uint8(colorIdx)
	if colorIdx < 0 {
		current, err := s.repo.ReadRegister(target)
		if err != nil {
			return err
		}
		color = (current & 0xF0) >> 4
	}

	newVal := (color << 4) | dpiIdx
	if err := s.repo.WriteRegister(target, newVal); err != nil {
		return err
	}

	if !switchSlot {
		return nil
	}

	return s.repo.WriteRegister(RegDPISelect, uint8((slot-1)*32))
}

func (s *Service) SetCPIAction(action string) error {
	val, ok := CPIActionMap[action]
	if !ok {
		return fmt.Errorf("invalid CPI action %q", action)
	}
	return s.repo.WriteRegister(RegCPIButton, val)
}

func decodeRate(raw uint8) string {
	switch raw {
	case Rate125:
		return "125Hz"
	case Rate250:
		return "250Hz"
	case Rate500:
		return "500Hz"
	case Rate1000:
		return "1000Hz"
	default:
		return "Unknown"
	}
}

func decodeActiveSlot(raw uint8) int {
	if raw <= 3 {
		return int(raw) + 1
	}

	switch raw {
	case 0x20:
		return 2
	case 0x40:
		return 3
	case 0x60:
		return 4
	default:
		return int(raw&0x03) + 1
	}
}

func decodeButtonMask(mask uint8) (left, right, middle, sideBack, sideForward bool) {
	return mask&0x01 != 0,
		mask&0x02 != 0,
		mask&0x04 != 0,
		mask&0x08 != 0,
		mask&0x10 != 0
}

func (s *Service) enterBank0() error {
	return s.enterBank(0)
}

func (s *Service) enterBank1() error {
	return s.enterBank(1)
}

func (s *Service) enterBank(bank uint16) error {
	if bank > MaxBankID {
		return fmt.Errorf("invalid bank %d; must be 0..%d", bank, MaxBankID)
	}

	selectIdx := (bank << 8) | ControlIdxBankSelect
	if bank != 1 {
		return s.repo.SendControl(ReqTypeWrite, ReqCodeControl, ControlValDefault, selectIdx)
	}

	return errors.Join(
		s.repo.SendControl(ReqTypeWrite, ReqCodeControl, ControlValDefault, selectIdx),
		s.repo.SendControl(ReqTypeWrite, ReqCodeControl, ControlValDefault, ControlIdxBank1IO),
	)
}

func decodeRGBMode(raw uint8) string {
	switch raw & 0xF0 {
	case 0x20:
		return "Breathing"
	case 0x40:
		return "Breathing + Segment Cycle"
	case 0xE0:
		return "Off"
	case 0x00:
		return "Always On"
	case 0x60:
		return "6 Color Cycle"
	case 0x80:
		return "12 Color Cycle"
	case 0xA0:
		return "768 Color Cycle"
	default:
		return "Unknown"
	}
}

func decodeCPIAction(raw uint8) string {
	for name, code := range CPIActionMap {
		if code == raw {
			return name
		}
	}
	return "Unknown"
}
