package sound

import (
	"context"
	"sync"

	"github.com/Djoulzy/emuai/internal/emulator"
)

const appleIISpeakerToggleAddress uint16 = 0xC030

type NullSound struct {
	mu          sync.Mutex
	name        string
	phaseHigh   bool
	toggleCount uint64
}

func NewNullSound(name string) *NullSound {
	return &NullSound{name: name}
}

func (s *NullSound) Name() string { return s.name }

func (s *NullSound) Reset(_ context.Context, _ *emulator.Bus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.phaseHigh = false
	s.toggleCount = 0
	return nil
}

func (s *NullSound) Tick(_ context.Context, _ emulator.Tick, _ *emulator.Bus) error { return nil }

func (s *NullSound) Close() error { return nil }

func (s *NullSound) Read(addr uint16) (byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if addr == appleIISpeakerToggleAddress {
		s.toggleLocked()
	}
	return 0x00, nil
}

func (s *NullSound) Write(addr uint16, _ byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if addr == appleIISpeakerToggleAddress {
		s.toggleLocked()
	}
	return nil
}

func (s *NullSound) ToggleCount() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.toggleCount
}

func (s *NullSound) PhaseHigh() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.phaseHigh
}

func (s *NullSound) toggleLocked() {
	s.phaseHigh = !s.phaseHigh
	s.toggleCount++
}
