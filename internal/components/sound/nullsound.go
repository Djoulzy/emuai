package sound

import (
	"context"

	"github.com/Djoulzy/emuai/internal/emulator"
)

type NullSound struct {
	name string
}

func NewNullSound(name string) *NullSound {
	return &NullSound{name: name}
}

func (s *NullSound) Name() string { return s.name }

func (s *NullSound) Reset(_ context.Context) error { return nil }

func (s *NullSound) Tick(_ context.Context, _ emulator.Tick, _ *emulator.Bus) error { return nil }

func (s *NullSound) Close() error { return nil }
