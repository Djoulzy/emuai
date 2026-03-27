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
