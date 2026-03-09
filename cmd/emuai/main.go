package main

import (
	"context"
	"log"
	"time"

	"github.com/Djoulzy/emuai/internal/components/cpu"
	"github.com/Djoulzy/emuai/internal/components/memory"
	"github.com/Djoulzy/emuai/internal/components/peripheral"
	"github.com/Djoulzy/emuai/internal/components/sound"
	"github.com/Djoulzy/emuai/internal/components/video"
	"github.com/Djoulzy/emuai/internal/emulator"
)

func main() {
	board, err := emulator.NewMotherboard(emulator.Config{FrequencyHz: 1_000_000})
	if err != nil {
		log.Fatalf("create motherboard: %v", err)
	}
	defer func() {
		if err := board.Close(); err != nil {
			log.Printf("close warning: %v", err)
		}
	}()

	ram, err := memory.NewRAM("main-ram", 0x0000, 0xFFFF)
	if err != nil {
		log.Fatalf("create RAM: %v", err)
	}

	if err := board.Bus().MapDevice(0x0000, 0xFFFF, "main-ram", ram); err != nil {
		log.Fatalf("map RAM: %v", err)
	}

	program := []byte{
		0xA9, 0x42, // LDA #$42
		0x8D, 0x00, 0x10, // STA $1000
		0xAA, // TAX
		0xE8, // INX
		0xEA, // NOP
		0x00, // BRK
	}

	for i, b := range program {
		if err := board.Bus().Write(0x0200+uint16(i), b); err != nil {
			log.Fatalf("seed RAM: %v", err)
		}
	}

	components := []emulator.ClockedComponent{
		ram,
		cpu.NewCPU6502("cpu-main", 0x0200),
		video.NewNullVideo("video-main"),
		sound.NewNullSound("sound-main"),
		peripheral.NewKeyboard("kbd-main"),
	}

	for _, c := range components {
		if err := board.AddComponent(c); err != nil {
			log.Fatalf("add component %s: %v", c.Name(), err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
	defer cancel()

	if err := board.Reset(ctx); err != nil {
		log.Fatalf("reset board: %v", err)
	}

	if err := board.Run(ctx); err != nil && err != context.DeadlineExceeded {
		log.Fatalf("run board: %v", err)
	}

	log.Printf("emulation stopped after %d cycles", board.Cycle())
}
