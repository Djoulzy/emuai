package peripheral

import (
	"context"

	"github.com/Djoulzy/emuai/internal/emulator"
)

type Keyboard struct {
	name string
}

func NewKeyboard(name string) *Keyboard {
	return &Keyboard{name: name}
}

func (k *Keyboard) Name() string { return k.name }

func (k *Keyboard) Reset(_ context.Context, _ *emulator.Bus) error { return nil }

func (k *Keyboard) Tick(_ context.Context, _ emulator.Tick, _ *emulator.Bus) error { return nil }

func (k *Keyboard) Close() error { return nil }
