package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadROMSetResolvesRelativePaths(t *testing.T) {
	tempDir := t.TempDir()
	romPath := filepath.Join(tempDir, "monitor.bin")
	if err := os.WriteFile(romPath, []byte{0xEA, 0x00}, 0o644); err != nil {
		t.Fatalf("write ROM: %v", err)
	}

	configPath := filepath.Join(tempDir, "roms.yaml")
	configData := []byte("roms:\n  - name: monitor\n    path: monitor.bin\n    start: 0xD000\n")
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
