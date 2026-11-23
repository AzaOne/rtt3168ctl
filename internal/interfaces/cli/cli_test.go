package cli

import (
	"bytes"
	"io"
	"reflect"
	"strings"
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

func TestVersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_, exit, err := Parse([]string{"--version"}, "rtt3168ctl", &stdout, &stderr)
	fields := strings.Fields(stdout.String())
	if err != nil || !exit || stderr.Len() != 0 || len(fields) != 2 || fields[0] != "rtt3168ctl" {
		t.Fatalf("exit=%v err=%v stdout=%q stderr=%q", exit, err, stdout.String(), stderr.String())
	}
}

func TestOptionSpelling(t *testing.T) {
	for arg, want := range map[string]string{"-version": "--version", "-mode=read": "--mode", "-help": "--help", "--h": "-h", "---json": "--json"} {
		_, _, err := Parse([]string{arg}, "rtt3168ctl", io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "use "+want) {
			t.Errorf("%s: expected suggestion %s, got %v", arg, want, err)
		}
	}
	for _, arg := range []string{"-h", "--help"} {
		var usage bytes.Buffer
		_, exit, err := Parse([]string{arg}, "rtt3168ctl", io.Discard, &usage)
		if err != nil || !exit || !strings.Contains(usage.String(), "--mode") {
			t.Errorf("%s: exit=%v err=%v usage=%q", arg, exit, err, usage.String())
		}
	}
	cmd, exit, err := Parse([]string{"--mode=read", "--speed", "-1", "--json"}, "rtt3168ctl", io.Discard, io.Discard)
	if err != nil || exit || cmd.Mode != "read" || !cmd.JSONOutput {
		t.Fatalf("long options: cmd=%+v exit=%v err=%v", cmd, exit, err)
	}
}
