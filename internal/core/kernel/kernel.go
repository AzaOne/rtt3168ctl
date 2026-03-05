package kernel

import (
	"errors"
	"fmt"
	"log"

	"rtt3168ctl/internal/core/config"

	"github.com/google/gousb"
)

type Device interface {
	Control(rType, request uint8, val, idx uint16, data []byte) (int, error)
	SetAutoDetach(autodetach bool) error
	Close() error
}

type Kernel struct {
	cfg    config.AppConfig
	logger *log.Logger
}

func New(cfg config.AppConfig, logger *log.Logger) *Kernel {
	return &Kernel{cfg: cfg, logger: logger}
}

func (k *Kernel) Run(work func(Device) error) error {
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(k.cfg.Device.VID, k.cfg.Device.PID)
	if err != nil {
		return fmt.Errorf("could not open device: %w", err)
	}
	if dev == nil {
		return errors.New("usb device not found; ensure the mouse is connected and accessible")
	}

	if err := dev.SetAutoDetach(true); err != nil {
		k.logger.Printf("warning: failed to enable auto-detach: %v", err)
	}
	defer func() {
		if err := dev.Close(); err != nil {
			k.logger.Printf("warning: failed to close device: %v", err)
		}
	}()

	return work(dev)
}
