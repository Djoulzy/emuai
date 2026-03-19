package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRAMLoad(t *testing.T) {
	ram, err := NewRAM("ram", 0x0200, 0x02FF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}

	data := []byte{0xA9, 0x42, 0x00}
	if err := ram.Load(0x0200, data); err != nil {
		t.Fatalf("load: %v", err)
	}

	for i, want := range data {
		got, err := ram.Read(0x0200 + uint16(i))
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got != want {
			t.Fatalf("unexpected byte at offset %d: got 0x%02X want 0x%02X", i, got, want)
		}
	}
}

func TestRAMLoadOutOfRange(t *testing.T) {
	ram, err := NewRAM("ram", 0x0200, 0x0202)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}

	if err := ram.Load(0x0201, []byte{0x01, 0x02, 0x03}); err == nil {
		t.Fatalf("expected out-of-range load error")
	}
}

func TestRAMLoadFile(t *testing.T) {
	ram, err := NewRAM("ram", 0x0400, 0x04FF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "program.bin")
	data := []byte{0xEA, 0xEA, 0x00}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	if err := ram.LoadFile(path, 0x0400); err != nil {
		t.Fatalf("load file: %v", err)
	}

	for i, want := range data {
		got, err := ram.Read(0x0400 + uint16(i))
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got != want {
			t.Fatalf("unexpected byte at offset %d: got 0x%02X want 0x%02X", i, got, want)
		}
	}
}
