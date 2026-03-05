package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/gousb"
)

const (
	defaultVID gousb.ID = 0x093A
	defaultPID gousb.ID = 0x2533
)

type DeviceConfig struct {
	VID gousb.ID
	PID gousb.ID
}

type AppConfig struct {
	Device DeviceConfig
}

func Default() AppConfig {
	return AppConfig{
		Device: DeviceConfig{
			VID: defaultVID,
			PID: defaultPID,
		},
	}
}

func LoadFromEnv() (AppConfig, error) {
	cfg := Default()

	if rawVID := strings.TrimSpace(os.Getenv("MOUSE_VID")); rawVID != "" {
		vid, err := parseUSBID(rawVID)
		if err != nil {
			return AppConfig{}, fmt.Errorf("invalid MOUSE_VID: %w", err)
		}
		cfg.Device.VID = vid
	}

	if rawPID := strings.TrimSpace(os.Getenv("MOUSE_PID")); rawPID != "" {
		pid, err := parseUSBID(rawPID)
		if err != nil {
			return AppConfig{}, fmt.Errorf("invalid MOUSE_PID: %w", err)
		}
		cfg.Device.PID = pid
	}

	return cfg, nil
}

func parseUSBID(raw string) (gousb.ID, error) {
	parsed, err := strconv.ParseUint(raw, 0, 16)
	if err != nil {
		return 0, err
	}
	return gousb.ID(parsed), nil
}
