package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadROMSetResolvesRelativePaths(t *testing.T) {
	tempDir := t.TempDir()
	romPath := filepath.Join(tempDir, "monitor.bin")
	chargenPath := filepath.Join(tempDir, "chargen.bin")
	slot6Path := filepath.Join(tempDir, "disk2.bin")
	if err := os.WriteFile(romPath, []byte{0xEA, 0x00}, 0o644); err != nil {
		t.Fatalf("write ROM: %v", err)
	}
	if err := os.WriteFile(chargenPath, make([]byte, 256*8), 0o644); err != nil {
		t.Fatalf("write chargen ROM: %v", err)
	}
	if err := os.WriteFile(slot6Path, make([]byte, 256), 0o644); err != nil {
		t.Fatalf("write slot ROM: %v", err)
	}

	configPath := filepath.Join(tempDir, "roms.yaml")
	configData := []byte("chargen:\n  path: chargen.bin\nslots:\n  slot6:\n    path: disk2.bin\nroms:\n  - name: monitor\n    path: monitor.bin\n    start: 0xD000\n")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	set, err := LoadROMSet(configPath)
	if err != nil {
		t.Fatalf("load ROM set: %v", err)
	}

	if len(set.ROMs) != 1 {
		t.Fatalf("unexpected ROM count: got %d want 1", len(set.ROMs))
	}
	if got := set.ResolveChargenPath(filepath.Dir(configPath)); got != chargenPath {
		t.Fatalf("unexpected resolved chargen path: got %q want %q", got, chargenPath)
	}
	if got := set.ResolveSlotROMPath(6, filepath.Dir(configPath)); got != slot6Path {
		t.Fatalf("unexpected resolved slot6 path: got %q want %q", got, slot6Path)
	}
	if got := set.ConfiguredSlots(); len(got) != 1 || got[0] != 6 {
		t.Fatalf("unexpected configured slots: got %v want [6]", got)
	}

	rom := set.ROMs[0]
	if got := rom.ResolvePath(filepath.Dir(configPath)); got != romPath {
		t.Fatalf("unexpected resolved path: got %q want %q", got, romPath)
	}

	if got := rom.Start.Uint16(); got != 0xD000 {
		t.Fatalf("unexpected start address: got 0x%04X want 0xD000", got)
	}
}

func TestLoadROMSetRejectsMissingROMList(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "roms.yaml")
	if err := os.WriteFile(configPath, []byte("roms: []\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadROMSet(configPath); err == nil {
		t.Fatal("expected empty ROM list to fail")
	}
}

func TestLoadROMSetRejectsMissingROMFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "roms.yaml")
	configData := []byte("roms:\n  - path: missing.bin\n    start: 0xD000\n")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadROMSet(configPath); err == nil {
		t.Fatal("expected missing ROM file to fail")
	}
}

func TestLoadROMSetRejectsMissingChargenFile(t *testing.T) {
	tempDir := t.TempDir()
	romPath := filepath.Join(tempDir, "monitor.bin")
	if err := os.WriteFile(romPath, []byte{0xEA, 0x00}, 0o644); err != nil {
		t.Fatalf("write ROM: %v", err)
	}

	configPath := filepath.Join(tempDir, "roms.yaml")
	configData := []byte("chargen:\n  path: missing.bin\nroms:\n  - path: monitor.bin\n    start: 0xD000\n")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadROMSet(configPath); err == nil {
		t.Fatal("expected missing chargen ROM file to fail")
	}
}

func TestLoadROMSetRejectsMissingSlotROMFile(t *testing.T) {
	tempDir := t.TempDir()
	romPath := filepath.Join(tempDir, "monitor.bin")
	if err := os.WriteFile(romPath, []byte{0xEA, 0x00}, 0o644); err != nil {
		t.Fatalf("write ROM: %v", err)
	}

	configPath := filepath.Join(tempDir, "roms.yaml")
	configData := []byte("slots:\n  slot6:\n    path: missing.bin\nroms:\n  - path: monitor.bin\n    start: 0xD000\n")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadROMSet(configPath); err == nil {
		t.Fatal("expected missing slot ROM file to fail")
	}
}
