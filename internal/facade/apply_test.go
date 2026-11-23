package facade_test

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"rtt3168ctl/internal/core/kernel"
	"rtt3168ctl/internal/facade"
	"rtt3168ctl/internal/interfaces/cli"
)

type transfer struct {
	requestType, request uint8
	value, index         uint16
	length               int
}

type profileDevice struct {
	transfers []transfer
	failIndex uint16
	writeErr  error
}

func (d *profileDevice) Run(work func(kernel.Device) error) error { return work(d) }
func (*profileDevice) SetAutoDetach(bool) error                   { return nil }
func (*profileDevice) Close() error                               { return nil }

func (d *profileDevice) Control(requestType, request uint8, value, index uint16, data []byte) (int, error) {
	d.transfers = append(d.transfers, transfer{requestType, request, value, index, len(data)})
	if d.writeErr != nil && index == d.failIndex && requestType == 0x40 {
		return 0, d.writeErr
	}
	if requestType == 0xc0 && index == 0x0a && len(data) == 1 {
		data[0] = 0x2b // Switching modes must preserve the low nibble.
	}
	return len(data), nil
}

func TestApplyHyprlandProfile(t *testing.T) {
	args := strings.Fields("--mode apply --dpi1 1200:0 --dpi2 1600:4 --dpi3 1200:0 --dpi4 1600:4 --active-slot 2 --rgb-mode on --cpi-action dpi_cycle")
	cmd, exit, err := cli.Parse(args, "rtt3168ctl", io.Discard, io.Discard)
	if err != nil || exit {
		t.Fatalf("parse profile: exit=%v err=%v", exit, err)
	}

	// Wire values from the RTT3168CG2 protocol, independent of production constants.
	begin := []transfer{
		{0x40, 1, 0, 0, 0},
		{0x40, 1, 0x100, 0x5a09, 0}, // unlock
		{0x40, 1, 0x100, 0x017f, 0}, // bank 1
		{0x40, 1, 0x100, 0x2009, 0},
	}
	slots := []transfer{
		{0x40, 1, 0x100, 0x0502, 0}, // 1200 DPI, color 0
		{0x40, 1, 0x100, 0x4703, 0}, // 1600 DPI, color 4
		{0x40, 1, 0x100, 0x0504, 0},
		{0x40, 1, 0x100, 0x4705, 0},
	}
	settings := []transfer{
		{0xc0, 1, 0x100, 0x000a, 1}, // read RGB mode
		{0x40, 1, 0x100, 0x0b0a, 0}, // always on, preserve low nibble
		{0x40, 1, 0x100, 0xe80b, 0}, // CPI cycles DPI
		{0x40, 1, 0x100, 0x2009, 0}, // activate slot 2
	}
	end := []transfer{
		{0x40, 1, 0x100, 0x007f, 0}, // bank 0
		{0x40, 1, 0x100, 0x0009, 0}, // lock
		{0x40, 6, 0, 0, 0},          // reset
	}

	for _, tc := range []struct {
		name   string
		device profileDevice
		want   []transfer
	}{
		{"complete", profileDevice{}, slices.Concat(begin, slots, settings, end)},
		{"third slot write fails", profileDevice{failIndex: 0x0504, writeErr: errors.New("USB write failed")}, slices.Concat(begin, slots[:3], end)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := facade.New(&tc.device).Execute(cmd, io.Discard)
			if !errors.Is(err, tc.device.writeErr) {
				t.Errorf("Execute error = %v, want %v", err, tc.device.writeErr)
			}
			if !slices.Equal(tc.device.transfers, tc.want) {
				t.Errorf("USB transfers (type request value index length):\n got %x\nwant %x", tc.device.transfers, tc.want)
			}
		})
	}
}
