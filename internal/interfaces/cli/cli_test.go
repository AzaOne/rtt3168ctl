package cli

import (
	"reflect"
	"testing"
)

func TestParseBankSpec(t *testing.T) {
	t.Run("list and range with dedup", func(t *testing.T) {
		got, err := parseBankSpec("0,1,1,3-5,4,7")
		if err != nil {
			t.Fatalf("parseBankSpec returned error: %v", err)
		}

		want := []uint16{0, 1, 3, 4, 5, 7}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected banks: got %v want %v", got, want)
		}
	})

	t.Run("all", func(t *testing.T) {
		got, err := parseBankSpec("all")
		if err != nil {
			t.Fatalf("parseBankSpec returned error: %v", err)
		}
		if len(got) != 256 {
			t.Fatalf("unexpected bank count: got %d want 256", len(got))
		}
		if got[0] != 0 || got[len(got)-1] != 255 {
			t.Fatalf("unexpected boundaries: first=%d last=%d", got[0], got[len(got)-1])
		}
	})
}

func TestParseBankSpecErrors(t *testing.T) {
	cases := []string{
		"",
		"256",
		"-1",
		"7-2",
		"1,,2",
		"a",
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, err := parseBankSpec(tc); err == nil {
				t.Fatalf("expected error for %q", tc)
			}
		})
	}
}
