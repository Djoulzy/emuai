package video

import (
	"context"

	"github.com/Djoulzy/emuAI/internal/emulator"
)

type NullVideo struct {
	name string
}

func NewNullVideo(name string) *NullVideo {
	return &NullVideo{name: name}
}

func (v *NullVideo) Name() string { return v.name }

func (v *NullVideo) Reset(_ context.Context) error { return nil }

func (v *NullVideo) Tick(_ context.Context, _ emulator.Tick, _ *emulator.Bus) error { return nil }

func (v *NullVideo) Close() error { return nil }
