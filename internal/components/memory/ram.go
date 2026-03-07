package memory

import (
	"context"
	"fmt"

	"github.com/Djoulzy/emuAI/internal/emulator"
)

type RAM struct {
	name  string
	start uint16
	end   uint16
	data  []byte
}

func NewRAM(name string, start, end uint16) (*RAM, error) {
	if end < start {
		return nil, fmt.Errorf("ram: invalid range 0x%04X-0x%04X", start, end)
	}

	size := int(end-start) + 1
	return &RAM{
		name:  name,
		start: start,
		end:   end,
		data:  make([]byte, size),
	}, nil
}

func (r *RAM) Name() string {
	return r.name
}

func (r *RAM) Reset(_ context.Context) error {
	for i := range r.data {
		r.data[i] = 0
	}
	return nil
}

func (r *RAM) Tick(_ context.Context, _ emulator.Tick, _ *emulator.Bus) error {
	return nil
}

func (r *RAM) Close() error {
	return nil
}

func (r *RAM) Read(addr uint16) (byte, error) {
	idx, err := r.translate(addr)
	if err != nil {
		return 0, err
	}
	return r.data[idx], nil
}

func (r *RAM) Write(addr uint16, value byte) error {
	idx, err := r.translate(addr)
	if err != nil {
		return err
	}
	r.data[idx] = value
	return nil
}

func (r *RAM) translate(addr uint16) (int, error) {
	if addr < r.start || addr > r.end {
		return 0, fmt.Errorf("ram: address out of range 0x%04X", addr)
	}
	return int(addr - r.start), nil
}
