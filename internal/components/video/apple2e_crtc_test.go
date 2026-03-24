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
