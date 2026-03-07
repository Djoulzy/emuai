package emulator

import "context"

// Tick carries cycle information for one motherboard step.
type Tick struct {
	Cycle uint64
}

// ClockedComponent is driven by the motherboard on every cycle.
type ClockedComponent interface {
	Name() string
	Reset(ctx context.Context) error
	Tick(ctx context.Context, tick Tick, bus *Bus) error
	Close() error
}

// AddressableDevice is a memory-mapped module available on the bus.
type AddressableDevice interface {
	Read(addr uint16) (byte, error)
	Write(addr uint16, value byte) error
}
