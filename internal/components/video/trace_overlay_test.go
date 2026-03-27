package video

import (
	"strings"
	"testing"
)

func TestTraceOverlayStripsANSIAndKeepsLatestLines(t *testing.T) {
	overlay := NewTraceOverlay(3)

	_, err := overlay.Write([]byte("\x1b[38;5;81mFIRST\x1b[0m\nSECOND\nTHIRD\nFOURTH\n"))
	if err != nil {
		t.Fatalf("write overlay: %v", err)
	}

	got := overlay.Text(0)
	want := strings.Join([]string{"SECOND", "THIRD", "FOURTH"}, "\n")
	if got != want {
		t.Fatalf("unexpected overlay text:\n%s\nwant:\n%s", got, want)
	}
}

func TestTraceOverlayPreservesPartialLine(t *testing.T) {
	overlay := NewTraceOverlay(8)

	_, err := overlay.Write([]byte("HEADER\nVALUE"))
	if err != nil {
		t.Fatalf("write partial overlay: %v", err)
	}

	if got, want := overlay.Text(0), "HEADER\nVALUE"; got != want {
		t.Fatalf("unexpected overlay text %q, want %q", got, want)
	}

	_, err = overlay.Write([]byte("-DONE\n"))
	if err != nil {
		t.Fatalf("flush partial overlay: %v", err)
	}

	if got, want := overlay.Text(0), "HEADER\nVALUE-DONE"; got != want {
		t.Fatalf("unexpected merged partial line %q, want %q", got, want)
	}
}

func TestTraceOverlayTracksStatusAndClock(t *testing.T) {
	overlay := NewTraceOverlay(8)
	overlay.SetClock(12345)
	overlay.SetCPUState(0x00, 0x01, 0x02, 0xFD, 0xA5)

	_, err := overlay.Write([]byte("$0200  A9   LDA #$42\n"))
	if err != nil {
		t.Fatalf("write trace status: %v", err)
	}

	got := overlay.Status()
	if got.Cycle != 12345 {
		t.Fatalf("unexpected cycle %d", got.Cycle)
	}
	if got.Registers != "A:00 X:01 Y:02 SP:FD P:A5" {
		t.Fatalf("unexpected registers %q", got.Registers)
	}
	if got.Flags != "NUIC" {
		t.Fatalf("unexpected flags %q", got.Flags)
	}
}
