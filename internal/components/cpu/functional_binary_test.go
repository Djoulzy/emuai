package cpu

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Djoulzy/emuai/internal/components/memory"
	"github.com/Djoulzy/emuai/internal/emulator"
)

func TestCPU6502_FunctionalBinaryReachesSuccessLoop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional binary regression test in short mode")
	}

	program, err := os.ReadFile(filepath.Join("..", "..", "..", "assets", "6502_functional_test.bin"))
	if err != nil {
		t.Fatalf("read functional binary: %v", err)
	}

	bus := emulator.NewBus()
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}
	if err := ram.Load(0x0000, program); err != nil {
		t.Fatalf("load functional binary: %v", err)
	}

	c := NewCPU6502("cpu", 0x0400)
	c.SetHaltOnBRK(false)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	const (
		successLoopPC        = 0x3469
		stableFetches        = 8
		maxCycles     uint64 = 120_000_000
		historyLen           = 12
	)

	type fetchInfo struct {
		cycle  uint64
		pc     uint16
		opcode byte
	}

	history := make([]fetchInfo, 0, historyLen)
	appendHistory := func(cycle uint64, pc uint16, opcode byte) {
		if len(history) == historyLen {
			copy(history, history[1:])
			history = history[:historyLen-1]
		}
		history = append(history, fetchInfo{cycle: cycle, pc: pc, opcode: opcode})
	}

	var repeatedPC uint16
	var repeatedCount int

	for cycle := uint64(0); cycle < maxCycles; cycle++ {
		if c.current == nil {
			opcode, err := bus.Read(c.PC)
			if err != nil {
				t.Fatalf("read opcode at 0x%04X: %v", c.PC, err)
			}
			appendHistory(cycle, c.PC, opcode)

			if c.PC == repeatedPC {
				repeatedCount++
			} else {
				repeatedPC = c.PC
				repeatedCount = 1
			}

			if repeatedCount >= stableFetches {
				if c.PC != successLoopPC {
					t.Fatalf("functional binary reached stable loop at 0x%04X, want success loop 0x%04X; last fetches: %#v", c.PC, successLoopPC, history)
				}
				return
			}
		}

		if err := c.Tick(context.Background(), emulator.Tick{Cycle: cycle}, bus); err != nil {
			t.Fatalf("tick %d failed: %v; last fetches: %#v", cycle, err, history)
		}
	}

	t.Fatalf("functional binary did not reach a stable loop within %d cycles; last fetches: %#v", maxCycles, history)
}
