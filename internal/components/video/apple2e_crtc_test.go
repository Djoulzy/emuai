package video

import (
	"context"
	"testing"

	"github.com/Djoulzy/emuai/internal/components/memory"
	"github.com/Djoulzy/emuai/internal/emulator"
)

func TestAppleIIeCRTCDefaultMode(t *testing.T) {
	crtc, err := NewAppleIIeCRTC("crtc", Config{}, AppleIIeOptions{})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	mode := crtc.Mode()
	if !mode.Text {
		t.Fatal("expected default mode to start in text")
	}
	if mode.Columns80 {
		t.Fatal("expected default mode to start in 40 columns")
	}
	if mode.Page2 {
		t.Fatal("expected default mode to start on page 1")
	}
	if crtc.renderMode != appleIIeRenderText40 {
		t.Fatalf("expected text40 render mode, got %d", crtc.renderMode)
	}
	if got := crtc.ModeString(); got != "TEXT 40COL PAGE1" {
		t.Fatalf("unexpected mode string %q", got)
	}
}

func TestAppleIIeCRTCSwitchesUpdateMode(t *testing.T) {
	crtc, err := NewAppleIIeCRTC("crtc", Config{}, AppleIIeOptions{})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	sequence := []uint16{
		appleIIeSwitchGraphics,
		appleIIeSwitchHiRes,
		appleIIeSwitch80ColOn,
		appleIIeSwitchPage2,
		appleIIeSwitchMixed,
	}
	for _, addr := range sequence {
		if err := crtc.Write(addr, 0); err != nil {
			t.Fatalf("write 0x%04X: %v", addr, err)
		}
	}

	mode := crtc.Mode()
	if mode.Text {
		t.Fatal("expected graphics mode after graphics switch")
	}
	if !mode.HiRes {
		t.Fatal("expected hires mode after hires switch")
	}
	if !mode.Columns80 {
		t.Fatal("expected 80-column mode after soft switch")
	}
	if !mode.Page2 {
		t.Fatal("expected page 2 after soft switch")
	}
	if !mode.Mixed {
		t.Fatal("expected mixed mode after soft switch")
	}
	if status, err := crtc.Read(appleIIeSwitchPage2); err != nil || status != 0x80 {
		t.Fatalf("expected page2 status=0x80, got 0x%02X err=%v", status, err)
	}
	if crtc.renderMode != appleIIeRenderDoubleHiRes {
		t.Fatalf("expected double hires render mode, got %d", crtc.renderMode)
	}
}

func TestAppleIIeCRTCTextRenderingUsesCharacterROM(t *testing.T) {
	charROM := make([]byte, appleIIeCharROMSize)
	for row := 0; row < 8; row++ {
		charROM[(0x41*8)+row] = 0x7E
	}
	memory := AppleIIeMemory{}
	memory.MainText[0] = make([]byte, appleIIeTextPageSize)
	memory.MainText[0][appleIIeTextOffset(0, 0)] = 0xC1

	crtc, err := NewAppleIIeCRTC("crtc", Config{}, AppleIIeOptions{CharacterROM: charROM, Memory: memory, ColorDisplay: true})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	crtc.redrawFrame()
	snapshot := crtc.Framebuffer().Snapshot(1)

	litPixels := 0
	for x := 0; x < 14; x++ {
		if snapshot.Pixels[x] != appleIIeBlackColor {
			litPixels++
		}
	}
	if litPixels == 0 {
		t.Fatal("expected rendered glyph pixels in the first text cell")
	}
}

func TestAppleIIeCRTCDetectsEnhancedCharacterROM(t *testing.T) {
	charROM := make([]byte, appleIIeCharROMSizeEx)
	crtc, err := NewAppleIIeCRTC("crtc", Config{}, AppleIIeOptions{CharacterROM: charROM})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	if crtc.charROMKind != appleIIeCharacterROMEnhanced {
		t.Fatalf("unexpected character ROM kind: got %d want %d", crtc.charROMKind, appleIIeCharacterROMEnhanced)
	}
}

func TestDescribeAppleIIeCharacterROM(t *testing.T) {
	if got := DescribeAppleIIeCharacterROM(make([]byte, appleIIeCharROMSize)); got != "Apple II compatible chargen (2 KB)" {
		t.Fatalf("unexpected classic chargen description: %q", got)
	}
	if got := DescribeAppleIIeCharacterROM(make([]byte, appleIIeCharROMSizeEx)); got != "Apple IIe enhanced chargen (4 KB)" {
		t.Fatalf("unexpected enhanced chargen description: %q", got)
	}
}

func TestAppleIIeCRTCEnhancedCharacterROMMapsUppercaseAndLowercase(t *testing.T) {
	charROM := make([]byte, appleIIeCharROMSizeEx)
	for row := 0; row < 8; row++ {
		charROM[(0x01*8)+row] = 0x01
		charROM[(0x61*8)+row] = 0x20
		charROM[(0x41*8)+row] = 0x7E
		charROM[(0xE1*8)+row] = 0x7C
	}

	memory := AppleIIeMemory{}
	memory.MainText[0] = make([]byte, appleIIeTextPageSize)
	memory.MainText[0][appleIIeTextOffset(0, 0)] = 0xC1
	memory.MainText[0][appleIIeTextOffset(0, 1)] = 0xE1

	crtc, err := NewAppleIIeCRTC("crtc", Config{CRT: CRTConfig{Width: 280, Height: 192, RefreshHz: 60}}, AppleIIeOptions{CharacterROM: charROM, Memory: memory, ColorDisplay: true})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	crtc.redrawFrame()
	snapshot := crtc.Framebuffer().Snapshot(1)
	cellWidth := crtc.Config().CRT.Width / 40

	if snapshot.Pixels[0] == appleIIeBlackColor {
		t.Fatal("expected uppercase glyph to use enhanced Apple IIe normal bank")
	}
	if snapshot.Pixels[cellWidth+5] == appleIIeBlackColor {
		t.Fatal("expected lowercase glyph to use enhanced Apple IIe lowercase bank")
	}
	if snapshot.Pixels[cellWidth] != appleIIeBlackColor {
		t.Fatal("expected lowercase glyph left edge to remain dark with selected bank")
	}
}

func TestAppleIIeCRTCEnhancedCharacterROMUsesInverseBanks(t *testing.T) {
	charROM := make([]byte, appleIIeCharROMSizeEx)
	for row := 0; row < 8; row++ {
		charROM[(0x01*8)+row] = 0x00
		charROM[(0x41*8)+row] = 0x01
	}

	memory := AppleIIeMemory{}
	memory.MainText[0] = make([]byte, appleIIeTextPageSize)
	memory.MainText[0][appleIIeTextOffset(0, 0)] = 0x41

	crtc, err := NewAppleIIeCRTC("crtc", Config{CRT: CRTConfig{Width: 280, Height: 192, RefreshHz: 60}}, AppleIIeOptions{CharacterROM: charROM, Memory: memory, ColorDisplay: true})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	crtc.blinkOn = true
	crtc.redrawFrame()
	snapshot := crtc.Framebuffer().Snapshot(1)

	if snapshot.Pixels[0] == appleIIeBlackColor {
		t.Fatal("expected flashing uppercase glyph to use enhanced inverse bank when flash is active")
	}
}

func TestAppleIIeCRTCEnhancedCharacterROMDoesNotMirrorGlyphs(t *testing.T) {
	charROM := make([]byte, appleIIeCharROMSizeEx)
	for row := 0; row < 8; row++ {
		charROM[(0x61*8)+row] = 0x20
	}

	memory := AppleIIeMemory{}
	memory.MainText[0] = make([]byte, appleIIeTextPageSize)
	memory.MainText[0][appleIIeTextOffset(0, 0)] = 0xE1

	crtc, err := NewAppleIIeCRTC("crtc", Config{CRT: CRTConfig{Width: 280, Height: 192, RefreshHz: 60}}, AppleIIeOptions{CharacterROM: charROM, Memory: memory, ColorDisplay: true})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	crtc.redrawFrame()
	snapshot := crtc.Framebuffer().Snapshot(1)

	if snapshot.Pixels[5] == appleIIeBlackColor {
		t.Fatal("expected enhanced Apple IIe glyph bit order to light the right-facing pixel")
	}
	if snapshot.Pixels[1] != appleIIeBlackColor {
		t.Fatal("expected enhanced Apple IIe glyph bit order to keep the mirrored pixel dark")
	}
}

func TestAppleIIeCRTCTickAdvancesVBlankAndPresents(t *testing.T) {
	renderer := &spyRenderer{}
	crtc, err := newAppleIIeCRTC("crtc", Config{ClockHz: 120, CRT: CRTConfig{Width: 560, Height: 384, RefreshHz: 60}}, AppleIIeOptions{}, renderer)
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	if err := crtc.Reset(context.Background(), nil); err != nil {
		t.Fatalf("reset: %v", err)
	}

	for cycle := uint64(0); cycle <= 4; cycle++ {
		if err := crtc.Tick(context.Background(), emulator.Tick{Cycle: cycle}, nil); err != nil {
			t.Fatalf("tick %d: %v", cycle, err)
		}
	}

	if len(renderer.frames) != 2 {
		t.Fatalf("expected 2 rendered frames, got %d", len(renderer.frames))
	}
	if crtc.VBL != 0x80 && crtc.VBL != 0x00 {
		t.Fatalf("unexpected VBL value 0x%02X", crtc.VBL)
	}
}

func TestAppleIIeCRTCReadsTextPageFromBusMemory(t *testing.T) {
	charROM := make([]byte, appleIIeCharROMSize)
	for row := 0; row < 8; row++ {
		charROM[(0x41*8)+row] = 0x7E
	}

	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	page := make([]byte, appleIIeTextPageSize)
	for idx := range page {
		page[idx] = 0xA0
	}
	page[appleIIeTextOffset(0, 0)] = 0xC1
	if err := ram.Load(appleIIeTextBaseAddress(false), page); err != nil {
		t.Fatalf("load text page: %v", err)
	}

	bus := emulator.NewBus()
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	crtc, err := NewAppleIIeCRTC("crtc", Config{}, AppleIIeOptions{CharacterROM: charROM, ColorDisplay: true})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	if err := crtc.Reset(context.Background(), bus); err != nil {
		t.Fatalf("reset with bus: %v", err)
	}

	crtc.redrawFrame()
	snapshot := crtc.Framebuffer().Snapshot(1)

	litPixels := 0
	for x := 0; x < 14; x++ {
		if snapshot.Pixels[x] != appleIIeBlackColor {
			litPixels++
		}
	}
	if litPixels == 0 {
		t.Fatal("expected rendered glyph pixels from bus-backed text page")
	}
}

func TestAppleIIeCRTCPreservesTraceConfig(t *testing.T) {
	overlay := NewTraceOverlay(16)
	crtc, err := NewAppleIIeCRTC("crtc", Config{
		Backend: BackendNull,
		Trace:   overlay,
		TraceOn: true,
	}, AppleIIeOptions{})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	if crtc.cfg.Trace != overlay {
		t.Fatal("expected trace overlay to be preserved in renderer config")
	}
	if !crtc.cfg.TraceOn {
		t.Fatal("expected trace flag to be preserved in renderer config")
	}
}

type testKeyEventSink struct{}

func (testKeyEventSink) HandleKeyEvent(emulator.KeyEvent) {}

func TestAppleIIeCRTCPreservesKeyboardConfig(t *testing.T) {
	keyboard := testKeyEventSink{}
	crtc, err := NewAppleIIeCRTC("crtc", Config{
		Backend:  BackendNull,
		Keyboard: keyboard,
	}, AppleIIeOptions{})
	if err != nil {
		t.Fatalf("new crtc: %v", err)
	}
	t.Cleanup(func() { _ = crtc.Close() })

	if crtc.cfg.Keyboard == nil {
		t.Fatal("expected keyboard sink to be preserved in renderer config")
	}
	if _, ok := crtc.cfg.Keyboard.(testKeyEventSink); !ok {
		t.Fatalf("unexpected keyboard sink type %T", crtc.cfg.Keyboard)
	}
}
