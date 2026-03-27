package video

import "testing"

func TestComposeFrameWithTraceExtendsFrameAndDrawsPanel(t *testing.T) {
	frame := Frame{
		Sequence: 3,
		Width:    4,
		Height:   96,
		Pixels:   make([]uint32, 4*96),
	}
	for i := range frame.Pixels {
		frame.Pixels[i] = 0xFF112233
	}
	frame.Pixels[1*frame.Width+1] = 0xFF445566
	frame.Pixels[1*frame.Width+2] = 0xFF445566

	got := composeFrameWithTrace(frame, "CYC PC ASM\n0 $0200 LDA", true)

	if got.Width != frame.Width+vulkanTracePanelWidth {
		t.Fatalf("unexpected composite width %d", got.Width)
	}
	if got.Height != frame.Height {
		t.Fatalf("unexpected composite height %d", got.Height)
	}
	if got.Sequence != frame.Sequence {
		t.Fatalf("unexpected sequence %d", got.Sequence)
	}
	if got.Pixels[0] != frame.Pixels[0] {
		t.Fatalf("expected CRT pixels preserved, got 0x%08X", got.Pixels[0])
	}

	panelPixel := got.Pixels[frame.Width+10]
	if panelPixel != tracePanelBackground && panelPixel != tracePanelDivider {
		return
	}

	textFound := false
	for y := 0; y < got.Height; y++ {
		for x := frame.Width; x < got.Width; x++ {
			pixel := got.Pixels[y*got.Width+x]
			if pixel == tracePanelHeader || pixel == tracePanelText {
				textFound = true
				break
			}
		}
		if textFound {
			break
		}
	}
	if !textFound {
		t.Fatal("expected trace panel text pixels to be drawn")
	}
}

func TestVisibleTraceLinesFallsBackWhenEmpty(t *testing.T) {
	lines := visibleTraceLines("", 3, true)
	if len(lines) != 3 {
		t.Fatalf("expected 3 fallback lines, got %d", len(lines))
	}
	if lines[0] != "TRACE ACTIVE" {
		t.Fatalf("unexpected first fallback line %q", lines[0])
	}
}

func TestVisibleTraceLinesShowsDisabledState(t *testing.T) {
	lines := visibleTraceLines("", 3, false)
	if lines[0] != "TRACE DISABLED" {
		t.Fatalf("unexpected disabled line %q", lines[0])
	}
}
