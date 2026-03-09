package cpu

import (
	"context"
	"testing"

	"github.com/Djoulzy/emuai/internal/components/memory"
	"github.com/Djoulzy/emuai/internal/emulator"
)

func TestCPU6502_ProgramFlow(t *testing.T) {
	bus := emulator.NewBus()
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	program := []byte{
		0xA9, 0x05, // LDA #$05
		0x69, 0x03, // ADC #$03 => A = 8
		0x85, 0x10, // STA $10
		0xA5, 0x10, // LDA $10
		0xAA, // TAX
		0xE8, // INX => X = 9
		0x00, // BRK
	}
	for i, b := range program {
		if err := bus.Write(0x0200+uint16(i), b); err != nil {
			t.Fatalf("seed RAM: %v", err)
		}
	}

	c := NewCPU6502("cpu", 0x0200)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	for i := 0; i < 64 && !c.Halted(); i++ {
		if err := c.Tick(context.Background(), emulator.Tick{Cycle: uint64(i)}, bus); err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
	}
	if !c.Halted() {
		t.Fatalf("expected CPU to halt on BRK")
	}

	if c.A != 0x08 {
		t.Fatalf("expected A=0x08, got 0x%02X", c.A)
	}
	if c.X != 0x09 {
		t.Fatalf("expected X=0x09, got 0x%02X", c.X)
	}
	v, err := bus.Read(0x0010)
	if err != nil {
		t.Fatalf("read ram: %v", err)
	}
	if v != 0x08 {
		t.Fatalf("expected RAM[0x0010]=0x08, got 0x%02X", v)
	}
}

func TestCPU6502_InstructionResumesOnNextCycle(t *testing.T) {
	bus := emulator.NewBus()
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	program := []byte{0xA9, 0x7F, 0x00} // LDA #$7F ; BRK
	for i, b := range program {
		if err := bus.Write(0x0200+uint16(i), b); err != nil {
			t.Fatalf("seed RAM: %v", err)
		}
	}

	c := NewCPU6502("cpu", 0x0200)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	// Cycle 1: opcode fetch only (LDA), instruction not completed yet.
	if err := c.Tick(context.Background(), emulator.Tick{Cycle: 0}, bus); err != nil {
		t.Fatalf("tick 0: %v", err)
	}
	if c.A != 0x00 {
		t.Fatalf("expected A unchanged after fetch cycle, got 0x%02X", c.A)
	}
	if c.PC != 0x0201 {
		t.Fatalf("expected PC=0x0201 after fetch, got 0x%04X", c.PC)
	}

	// Cycle 2: LDA immediate data cycle, now A is updated.
	if err := c.Tick(context.Background(), emulator.Tick{Cycle: 1}, bus); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	if c.A != 0x7F {
		t.Fatalf("expected A=0x7F after second cycle, got 0x%02X", c.A)
	}
	if c.PC != 0x0202 {
		t.Fatalf("expected PC=0x0202 after operand read, got 0x%04X", c.PC)
	}
}

func TestCPU6502_UnsupportedOpcode(t *testing.T) {
	bus := emulator.NewBus()
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	if err := bus.Write(0x0200, 0x02); err != nil {
		t.Fatalf("seed RAM: %v", err)
	}

	c := NewCPU6502("cpu", 0x0200)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	err = c.Tick(context.Background(), emulator.Tick{Cycle: 0}, bus)
	if err == nil {
		t.Fatalf("expected unsupported opcode error")
	}
}
