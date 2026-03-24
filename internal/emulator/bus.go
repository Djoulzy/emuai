package emulator

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrBusNoDevice = errors.New("bus: no mapped device for address")
	ErrBusOverlap  = errors.New("bus: mapping overlaps existing range")
)

type mapping struct {
	start  uint16
	end    uint16
	device AddressableDevice
	name   string
}

// Bus routes memory accesses to mapped devices.
type Bus struct {
	mu       sync.RWMutex
	mappings []mapping
}

func NewBus() *Bus {
	return &Bus{}
}

func (b *Bus) MapDevice(start, end uint16, name string, device AddressableDevice) error {
	if device == nil {
		return errors.New("bus: device is nil")
	}
	if end < start {
		return fmt.Errorf("bus: invalid range 0x%04X-0x%04X", start, end)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, m := range b.mappings {
		if rangesOverlap(start, end, m.start, m.end) {
			return fmt.Errorf("%w: %s [0x%04X-0x%04X]", ErrBusOverlap, m.name, m.start, m.end)
		}
	}

	b.mappings = append(b.mappings, mapping{
		start:  start,
		end:    end,
		device: device,
		name:   name,
	})

	return nil
}

func (b *Bus) Read(addr uint16) (byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	m, ok := b.findMapping(addr)
	if !ok {
		return 0, fmt.Errorf("%w: 0x%04X", ErrBusNoDevice, addr)
	}

	return m.device.Read(addr)
}

func (b *Bus) ReadWord(addr uint16) (uint16, error) {
	lo, err := b.Read(addr)
	if err != nil {
		return 0, err
	}
	hi, err := b.Read(addr + 1)
	if err != nil {
		return 0, err
	}
	return uint16(hi)<<8 | uint16(lo), nil
}

func (b *Bus) Write(addr uint16, value byte) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	m, ok := b.findMapping(addr)
	if !ok {
		return fmt.Errorf("%w: 0x%04X", ErrBusNoDevice, addr)
	}

	return m.device.Write(addr, value)
}

func (b *Bus) findMapping(addr uint16) (mapping, bool) {
	for _, m := range b.mappings {
		if addr >= m.start && addr <= m.end {
			return m, true
		}
	}
	return mapping{}, false
}

func rangesOverlap(aStart, aEnd, bStart, bEnd uint16) bool {
	return aStart <= bEnd && bStart <= aEnd
}
