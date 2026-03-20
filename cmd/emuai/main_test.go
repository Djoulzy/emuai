package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Djoulzy/emuai/internal/components/cpu"
	"github.com/Djoulzy/emuai/internal/components/memory"
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

	processor := cpu.NewCPU6502("cpu-test", 0x0200)
	if err := board.AddComponent(ram); err != nil {
		t.Fatalf("add RAM: %v", err)
	}
	if err := board.AddComponent(processor); err != nil {
		t.Fatalf("add CPU: %v", err)
	}

	if err := board.Reset(context.Background()); err != nil {
		t.Fatalf("reset board: %v", err)
	}

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
