package mouse

import (
	"fmt"
	"time"

	"rtt3168ctl/internal/core/kernel"
)

type Repository struct {
	dev kernel.Device
}

func NewRepository(dev kernel.Device) *Repository {
	return &Repository{dev: dev}
}

func (r *Repository) ReadRegister(regID uint16) (uint8, error) {
	buf := make([]byte, 1)
	_, err := r.dev.Control(ReqTypeRead, 1, 0x0100, regID, buf)
	if err != nil {
		return 0, fmt.Errorf("read register %d: %w", regID, err)
	}
	return buf[0], nil
}

func (r *Repository) WriteRegister(regID uint16, value uint8) error {
	_, err := r.dev.Control(ReqTypeWrite, 1, 0x0100, (uint16(value)<<8)|regID, nil)
	if err != nil {
		return fmt.Errorf("write register %d: %w", regID, err)
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (r *Repository) SendControl(reqType, req uint8, val, idx uint16) error {
	_, err := r.dev.Control(reqType, req, val, idx, nil)
	if err != nil {
		return fmt.Errorf("control request type=0x%X req=0x%X val=0x%X idx=0x%X: %w", reqType, req, val, idx, err)
	}
	return nil
}
