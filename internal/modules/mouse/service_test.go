package mouse

import "testing"

func TestDecodeActiveSlot(t *testing.T) {
	for raw, want := range map[uint8]int{0: 1, 1: 2, 2: 3, 3: 4, 0x20: 2, 0x40: 3, 0x60: 4} {
		if got := decodeActiveSlot(raw); got != want {
			t.Errorf("decodeActiveSlot(%#02x) = %d, want %d", raw, got, want)
		}
	}
}

func TestDecodeRGBMode(t *testing.T) {
	modes := map[uint8]string{
		0x01: "Always On",
		0x21: "Breathing",
		0x2f: "Breathing",
		0x41: "Breathing + Segment Cycle",
		0x61: "6 Color Cycle",
		0x81: "12 Color Cycle",
		0xa1: "768 Color Cycle",
		0xe1: "Off",
		0xff: "Unknown",
	}
	for raw, want := range modes {
		if got := decodeRGBMode(raw); got != want {
			t.Errorf("decodeRGBMode(%#02x) = %q, want %q", raw, got, want)
		}
	}
}
