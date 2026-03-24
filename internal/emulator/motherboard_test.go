package emulator

import (
	"context"
	"testing"
)

type resetSpyComponent struct {
	name     string
	resetBus *Bus
}

func (c *resetSpyComponent) Name() string {
	return c.name
}

func (c *resetSpyComponent) Reset(_ context.Context, bus *Bus) error {
	c.resetBus = bus
	return nil
}

func (c *resetSpyComponent) Tick(_ context.Context, _ Tick, _ *Bus) error {
	return nil
}

func (c *resetSpyComponent) Close() error {
	return nil
}

func TestMotherboardResetUsesInternalBusWhenNil(t *testing.T) {
	board, err := NewMotherboard(Config{FrequencyHz: 1})
	if err != nil {
		t.Fatalf("new motherboard: %v", err)
	}

	component := &resetSpyComponent{name: "spy"}
	if err := board.AddComponent(component); err != nil {
		t.Fatalf("add component: %v", err)
	}

	if err := board.Reset(context.Background(), nil); err != nil {
		t.Fatalf("reset motherboard: %v", err)
	}

	if component.resetBus != board.Bus() {
		t.Fatal("expected reset to receive motherboard bus")
	}
}
