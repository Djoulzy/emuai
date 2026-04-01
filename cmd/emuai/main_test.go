package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Djoulzy/emuai/internal/components/cpu"
	"github.com/Djoulzy/emuai/internal/components/memory"
	"github.com/Djoulzy/emuai/internal/components/peripheral"
	"github.com/Djoulzy/emuai/internal/components/sound"
	"github.com/Djoulzy/emuai/internal/components/video"
	"github.com/Djoulzy/emuai/internal/emulator"
)

func TestRunControlTogglePaused(t *testing.T) {
	control := &runControl{}

	if control.Paused() {
		t.Fatal("expected control to start unpaused")
	}

	if !control.TogglePaused() {
		t.Fatal("expected first toggle to pause execution")
	}

	if !control.Paused() {
		t.Fatal("expected paused state after first toggle")
	}

	if control.TogglePaused() {
		t.Fatal("expected second toggle to resume execution")
	}

	if control.Paused() {
		t.Fatal("expected resumed state after second toggle")
	}
}

func TestWaitWhilePausedReturnsWhenResumed(t *testing.T) {
	control := &runControl{}
	control.SetPaused(true)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(pausePollInterval / 2)
		control.SetPaused(false)
	}()

	if err := waitWhilePaused(ctx, control); err != nil {
		t.Fatalf("waitWhilePaused returned error: %v", err)
	}
}

func TestWaitWhilePausedStopsOnContextCancellation(t *testing.T) {
	control := &runControl{}
	control.SetPaused(true)

	ctx, cancel := context.WithTimeout(context.Background(), pausePollInterval)
	defer cancel()

	err := waitWhilePaused(ctx, control)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestProcessControlKeyPauseResume(t *testing.T) {
	control := &runControl{}

	if action := processControlKey(control, nil, ' '); action != "pause" {
		t.Fatalf("expected pause action, got %q", action)
	}

	if !control.Paused() {
		t.Fatal("expected paused state after space")
	}

	if action := processControlKey(control, nil, ' '); action != "resume" {
		t.Fatalf("expected resume action, got %q", action)
	}

	if control.Paused() {
		t.Fatal("expected resumed state after second space")
	}
}

func TestProcessControlKeyQuit(t *testing.T) {
	called := false
	action := processControlKey(&runControl{}, func() {
		called = true
	}, 'q')

	if action != "quit" {
		t.Fatalf("expected quit action, got %q", action)
	}

	if !called {
		t.Fatal("expected quit callback to be called")
	}
}

func TestSelectTraceWriter(t *testing.T) {
	t.Run("prefers overlay when available", func(t *testing.T) {
		fallback := &bytes.Buffer{}
		overlay := video.NewTraceOverlay(8)

		got := selectTraceWriter(overlay, fallback)
		if got != overlay {
			t.Fatal("expected trace overlay writer to be selected")
		}
	})

	t.Run("falls back when overlay missing", func(t *testing.T) {
		fallback := &bytes.Buffer{}

		got := selectTraceWriter(nil, fallback)
		if got != fallback {
			t.Fatal("expected fallback writer to be selected")
		}
	})
}

func TestRunMachineStopsAtProgramCounter(t *testing.T) {
	board, err := emulator.NewMotherboard(emulator.Config{FrequencyHz: motherboardFrequencyHz})
	if err != nil {
		t.Fatalf("new motherboard: %v", err)
	}
	defer func() {
		if err := board.Close(); err != nil {
			t.Fatalf("close motherboard: %v", err)
		}
	}()

	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := board.Bus().MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	processor := cpu.NewCPU6502("cpu-test")
	if err := board.AddComponent(ram); err != nil {
		t.Fatalf("add RAM: %v", err)
	}
	if err := board.AddComponent(processor); err != nil {
		t.Fatalf("add CPU: %v", err)
	}

	if err := board.Reset(context.Background(), nil); err != nil {
		t.Fatalf("reset board: %v", err)
	}
	processor.SetPC(0x0200)

	program := []byte{0xEA, 0xEA, 0x00}
	if err := ram.Load(0x0200, program); err != nil {
		t.Fatalf("load program: %v", err)
	}

	stopPC := &uint16Flag{value: 0x0201, set: true}
	if err := runMachine(context.Background(), board, processor, false, 0, nil, stopPC); err != nil {
		t.Fatalf("runMachine returned error: %v", err)
	}

	if processor.PC != 0x0201 {
		t.Fatalf("expected stop at PC 0x0201, got 0x%04X", processor.PC)
	}
	if processor.Halted() {
		t.Fatal("expected CPU not to halt when stop-pc triggers")
	}
	if board.Cycle() == 0 {
		t.Fatal("expected at least one cycle before stop-pc triggers")
	}
	if !processor.ReadyForInstruction() {
		t.Fatal("expected stop-pc to trigger at an instruction boundary")
	}
}

func TestLoadROMsFromConfig(t *testing.T) {
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}

	tempDir := t.TempDir()
	romPath := filepath.Join(tempDir, "monitor.bin")
	romData := []byte{0xA9, 0x42, 0x00}
	if err := os.WriteFile(romPath, romData, 0o644); err != nil {
		t.Fatalf("write ROM: %v", err)
	}

	configPath := filepath.Join(tempDir, "roms.yaml")
	configData := []byte("roms:\n  - name: monitor\n    path: monitor.bin\n    start: 0xD000\n")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loadedROMs, err := loadROMsFromConfig(ram, configPath)
	if err != nil {
		t.Fatalf("load ROMs from config: %v", err)
	}

	if len(loadedROMs) != 1 {
		t.Fatalf("unexpected loaded ROM count: got %d want 1", len(loadedROMs))
	}

	for idx, want := range romData {
		got, err := ram.Read(0xD000 + uint16(idx))
		if err != nil {
			t.Fatalf("read loaded ROM byte %d: %v", idx, err)
		}
		if got != want {
			t.Fatalf("unexpected ROM byte at offset %d: got 0x%02X want 0x%02X", idx, got, want)
		}
	}
}

func TestLoadROMsFromConfigLoadsSlotROMs(t *testing.T) {
	mmu, err := memory.NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	tempDir := t.TempDir()
	romPath := filepath.Join(tempDir, "monitor.bin")
	slotROMPath := filepath.Join(tempDir, "disk2.bin")
	romData := []byte{0xA9, 0x42, 0x00}
	slotROMData := []byte{0xEA, 0xEA, 0x00}
	if err := os.WriteFile(romPath, romData, 0o644); err != nil {
		t.Fatalf("write ROM: %v", err)
	}
	if err := os.WriteFile(slotROMPath, slotROMData, 0o644); err != nil {
		t.Fatalf("write slot ROM: %v", err)
	}

	configPath := filepath.Join(tempDir, "roms.yaml")
	configData := []byte("slots:\n  slot6:\n    path: disk2.bin\nroms:\n  - name: monitor\n    path: monitor.bin\n    start: 0xD000\n")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loadedROMs, err := loadROMsFromConfig(mmu, configPath)
	if err != nil {
		t.Fatalf("load ROMs from config: %v", err)
	}
	if len(loadedROMs) != 2 {
		t.Fatalf("unexpected loaded ROM count: got %d want 2", len(loadedROMs))
	}

	if got, err := mmu.Read(0xD000); err != nil || got != romData[0] {
		t.Fatalf("unexpected system ROM byte: got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0xC600); err != nil || got != slotROMData[0] {
		t.Fatalf("unexpected slot ROM byte: got 0x%02X err=%v", got, err)
	}
}

func TestMapAppleIIeSoftSwitchesRoutesVideoKeyboardAndSound(t *testing.T) {
	bus := emulator.NewBus()
	mmu, err := memory.NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "mmu", mmu); err != nil {
		t.Fatalf("map MMU: %v", err)
	}

	videoDevice, err := video.NewAppleIIeCRTC("video", video.Config{}, video.AppleIIeOptions{})
	if err != nil {
		t.Fatalf("new video device: %v", err)
	}
	t.Cleanup(func() { _ = videoDevice.Close() })

	soundDevice := sound.NewNullSound("sound")
	keyboardDevice := peripheral.NewKeyboard("keyboard")

	if err := mapAppleIIeSoftSwitches(bus, mmu, videoDevice, soundDevice, keyboardDevice); err != nil {
		t.Fatalf("mapAppleIIeSoftSwitches: %v", err)
	}

	if err := bus.Write(0xC050, 0); err != nil {
		t.Fatalf("write graphics soft-switch: %v", err)
	}
	graphicsStatus, err := bus.Read(0xC050)
	if err != nil {
		t.Fatalf("read graphics soft-switch: %v", err)
	}
	if graphicsStatus != 0x80 {
		t.Fatalf("unexpected graphics status: got 0x%02X want 0x80", graphicsStatus)
	}

	if err := bus.Write(0xC00D, 0); err != nil {
		t.Fatalf("write 80-column soft-switch: %v", err)
	}
	columnsStatus, err := bus.Read(0xC00D)
	if err != nil {
		t.Fatalf("read 80-column soft-switch: %v", err)
	}
	if columnsStatus != 0x80 {
		t.Fatalf("unexpected 80-column status: got 0x%02X want 0x80", columnsStatus)
	}

	if err := bus.Write(0xC000, 0); err != nil {
		t.Fatalf("write 80STORE soft-switch: %v", err)
	}
	store80Status, err := bus.Read(0xC018)
	if err != nil {
		t.Fatalf("read 80STORE status: %v", err)
	}
	if store80Status != 0x00 {
		t.Fatalf("unexpected 80STORE status after off write: got 0x%02X want 0x00", store80Status)
	}
	if err := bus.Write(0xC001, 0); err != nil {
		t.Fatalf("write 80STORE-on soft-switch: %v", err)
	}
	store80Status, err = bus.Read(0xC018)
	if err != nil {
		t.Fatalf("read 80STORE-on status: %v", err)
	}
	if store80Status != 0x80 {
		t.Fatalf("unexpected 80STORE status after on write: got 0x%02X want 0x80", store80Status)
	}

	if err := bus.Write(0xC00F, 0); err != nil {
		t.Fatalf("write ALTCHARSET soft-switch: %v", err)
	}
	altCharsetStatus, err := bus.Read(0xC01E)
	if err != nil {
		t.Fatalf("read ALTCHARSET status: %v", err)
	}
	if altCharsetStatus != 0x80 {
		t.Fatalf("unexpected ALTCHARSET status: got 0x%02X want 0x80", altCharsetStatus)
	}

	if err := bus.Write(0xC005, 0); err != nil {
		t.Fatalf("write RAMWRT-on soft-switch: %v", err)
	}
	if err := bus.Write(0x0200, 0x42); err != nil {
		t.Fatalf("write aux memory byte: %v", err)
	}
	if err := bus.Write(0xC003, 0); err != nil {
		t.Fatalf("write RAMRD-on soft-switch: %v", err)
	}
	auxValue, err := bus.Read(0x0200)
	if err != nil {
		t.Fatalf("read aux-backed memory byte: %v", err)
	}
	if auxValue != 0x42 {
		t.Fatalf("unexpected aux-backed memory byte: got 0x%02X want 0x42", auxValue)
	}
	if got, err := bus.Read(0xC013); err != nil || got != 0x80 {
		t.Fatalf("expected RAMRD status=0x80, got 0x%02X err=%v", got, err)
	}
	if got, err := bus.Read(0xC014); err != nil || got != 0x80 {
		t.Fatalf("expected RAMWRT status=0x80, got 0x%02X err=%v", got, err)
	}

	if err := bus.Write(0xC009, 0); err != nil {
		t.Fatalf("write ALTZP-on soft-switch: %v", err)
	}
	if err := bus.Write(0x0001, 0x99); err != nil {
		t.Fatalf("write aux zero-page byte: %v", err)
	}
	zeroPageAux, err := bus.Read(0x0001)
	if err != nil {
		t.Fatalf("read aux zero-page byte: %v", err)
	}
	if zeroPageAux != 0x99 {
		t.Fatalf("unexpected aux zero-page byte: got 0x%02X want 0x99", zeroPageAux)
	}
	if got, err := bus.Read(0xC016); err != nil || got != 0x80 {
		t.Fatalf("expected ALTZP status=0x80, got 0x%02X err=%v", got, err)
	}

	if err := bus.Write(0xC007, 0); err != nil {
		t.Fatalf("write INTCXROM-on soft-switch: %v", err)
	}
	if got, err := bus.Read(0xC015); err != nil || got != 0x80 {
		t.Fatalf("expected INTCXROM status=0x80, got 0x%02X err=%v", got, err)
	}

	if got, err := bus.Read(0xC017); err != nil || got != 0x00 {
		t.Fatalf("expected SLOTC3ROM status=0x00 by default, got 0x%02X err=%v", got, err)
	}
	if err := bus.Write(0xC00B, 0); err != nil {
		t.Fatalf("write SLOTC3ROM-on soft-switch: %v", err)
	}
	if got, err := bus.Read(0xC017); err != nil || got != 0x80 {
		t.Fatalf("expected SLOTC3ROM status=0x80, got 0x%02X err=%v", got, err)
	}

	keyboardDevice.HandleKeyEvent(emulator.KeyEvent{Rune: 'a', Action: emulator.KeyActionPress})
	keyData, err := bus.Read(0xC000)
	if err != nil {
		t.Fatalf("read keyboard data: %v", err)
	}
	if keyData != 0xC1 {
		t.Fatalf("unexpected keyboard data: got 0x%02X want 0xC1", keyData)
	}

	if _, err := bus.Read(0xC030); err != nil {
		t.Fatalf("read speaker soft-switch: %v", err)
	}
	if soundDevice.ToggleCount() != 1 {
		t.Fatalf("unexpected speaker toggle count: got %d want 1", soundDevice.ToggleCount())
	}
}

func TestResolveAppleIIeCharacterROMPathUsesROMConfigChargen(t *testing.T) {
	tempDir := t.TempDir()
	romPath := filepath.Join(tempDir, "monitor.bin")
	chargenPath := filepath.Join(tempDir, "custom-chargen.bin")
	if err := os.WriteFile(romPath, []byte{0xEA, 0x00}, 0o644); err != nil {
		t.Fatalf("write ROM: %v", err)
	}
	if err := os.WriteFile(chargenPath, make([]byte, 256*8), 0o644); err != nil {
		t.Fatalf("write chargen: %v", err)
	}

	configPath := filepath.Join(tempDir, "roms.yaml")
	configData := []byte("chargen:\n  path: custom-chargen.bin\nroms:\n  - path: monitor.bin\n    start: 0xD000\n")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := resolveAppleIIeCharacterROMPath(configPath)
	if err != nil {
		t.Fatalf("resolveAppleIIeCharacterROMPath returned error: %v", err)
	}
	if got != chargenPath {
		t.Fatalf("unexpected config chargen path: got %q want %q", got, chargenPath)
	}
}

func TestParseHexAddress(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    uint16
		wantErr bool
	}{
		{name: "0x prefix", raw: "0xD000", want: 0xD000},
		{name: "assembler prefix", raw: "$D000", want: 0xD000},
		{name: "bare hex", raw: "D000", want: 0xD000},
		{name: "empty", raw: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHexAddress(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected parse error")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseHexAddress returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected parsed address: got 0x%04X want 0x%04X", got, tt.want)
			}
		})
	}
}

func TestDumpMemoryBlock(t *testing.T) {
	program := []byte{0xA9, 0x42, 0xAA, 0x00}
	board, err := emulator.NewMotherboard(emulator.Config{FrequencyHz: motherboardFrequencyHz})
	if err != nil {
		t.Fatalf("new motherboard: %v", err)
	}
	defer func() {
		if err := board.Close(); err != nil {
			t.Fatalf("close motherboard: %v", err)
		}
	}()

	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := board.Bus().MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}
	if err := ram.Load(0x0200, program); err != nil {
		t.Fatalf("load program: %v", err)
	}

	var out bytes.Buffer
	if err := dumpMemoryBlock(&out, board.Bus(), 0x0200, 3); err != nil {
		t.Fatalf("dumpMemoryBlock: %v", err)
	}

	got := strings.TrimSpace(out.String())
	want := strings.Join([]string{
		"PC     BYTES     ASM",
		"$0200  A9 42     LDA  #$42",
		"$0202  AA        TAX  ",
		"$0203  00        BRK",
	}, "\n")
	if got != want {
		t.Fatalf("unexpected dump output:\n%s\nwant:\n%s", got, want)
	}
}

func TestWriteStartupTextToRAMEncodesUppercaseASCII(t *testing.T) {
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}

	writeStartupTextToRAM(ram, []string{"Hello, emuai!"})

	page := make([]byte, appleIIeTextPageSize)
	for idx := range page {
		value, err := ram.Read(appleIIeTextPage1Address + uint16(idx))
		if err != nil {
			t.Fatalf("read startup page byte %d: %v", idx, err)
		}
		page[idx] = value
	}

	tests := []struct {
		col  int
		want byte
	}{
		{col: 0, want: 0xC8},
		{col: 1, want: 0xC5},
		{col: 2, want: 0xCC},
		{col: 5, want: 0xAC},
		{col: 7, want: 0xC5},
		{col: 12, want: 0xA1},
	}

	for _, tt := range tests {
		if got := page[startupTextOffset(0, tt.col)]; got != tt.want {
			t.Fatalf("unexpected glyph at column %d: got 0x%02X want 0x%02X", tt.col, got, tt.want)
		}
	}

	if got := page[startupTextOffset(1, 0)]; got != 0xA0 {
		t.Fatalf("expected blank-filled second line, got 0x%02X", got)
	}
}

func TestWriteResetVectorStoresLittleEndianAddress(t *testing.T) {
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}

	if err := writeResetVector(ram, 0xD000); err != nil {
		t.Fatalf("writeResetVector: %v", err)
	}

	lo, err := ram.Read(resetVectorAddress)
	if err != nil {
		t.Fatalf("read reset vector low: %v", err)
	}
	hi, err := ram.Read(resetVectorAddress + 1)
	if err != nil {
		t.Fatalf("read reset vector high: %v", err)
	}

	if lo != 0x00 || hi != 0xD0 {
		t.Fatalf("unexpected reset vector bytes: got [%02X %02X] want [00 D0]", lo, hi)
	}
}

func TestStartupScreenLinesDefaultsToROMBootMessaging(t *testing.T) {
	lines := startupScreenLines(string(video.BackendNull), filepath.Join("ROMs", defaultROMConfigName), filepath.Join("ROMs", "custom-chargen.bin"))

	if lines[0] != "EMUAI APPLE IIE ROM BOOT" {
		t.Fatalf("unexpected boot banner: %q", lines[0])
	}
	if lines[3] != "CHAR ROM      : CUSTOM-CHARGEN.BIN" {
		t.Fatalf("unexpected chargen line: %q", lines[3])
	}
	if lines[4] != "ROM CONFIG    : APPLE2-ROMS.YAML" {
		t.Fatalf("unexpected ROM config line: %q", lines[4])
	}
	if lines[5] != "CPU RESET AFTER ROM LOAD" {
		t.Fatalf("unexpected reset line: %q", lines[5])
	}
}
