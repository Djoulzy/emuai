package video

import (
	"context"
	"fmt"

	"github.com/Djoulzy/emuai/internal/emulator"
)

type Device struct {
	name            string
	cfg             Config
	renderer        Renderer
	framebuffer     *Framebuffer
	cyclesPerFrame  uint64
	nextPresentTick uint64
	frameSequence   uint64
}

func NewDevice(name string, cfg Config) (*Device, error) {
	if name == "" {
		return nil, fmt.Errorf("video: device name is required")
	}

	cfg = cfg.normalized()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	renderer, err := newRenderer(cfg)
	if err != nil {
		return nil, err
	}
	if err := renderer.Init(cfg); err != nil {
		return nil, err
	}

	cyclesPerFrame := cfg.ClockHz / uint64(cfg.CRT.RefreshHz)
	if cyclesPerFrame == 0 {
		cyclesPerFrame = 1
	}

	return &Device{
		name:           name,
		cfg:            cfg,
		renderer:       renderer,
		framebuffer:    NewFramebuffer(cfg.CRT.Width, cfg.CRT.Height),
		cyclesPerFrame: cyclesPerFrame,
	}, nil
}

func (v *Device) Name() string { return v.name }

func (v *Device) Reset(_ context.Context) error {
	v.frameSequence = 0
	v.nextPresentTick = v.cyclesPerFrame
	v.framebuffer.Fill(0xFF050505)
	return nil
}

func (v *Device) Tick(_ context.Context, tick emulator.Tick, _ *emulator.Bus) error {
	if tick.Cycle < v.nextPresentTick {
		return nil
	}

	v.frameSequence++
	if err := v.renderer.Present(v.framebuffer.Snapshot(v.frameSequence)); err != nil {
		return err
	}
	v.nextPresentTick += v.cyclesPerFrame
	return nil
}

func (v *Device) Close() error {
	if v.renderer == nil {
		return nil
	}
	return v.renderer.Close()
}

func (v *Device) Framebuffer() *Framebuffer {
	return v.framebuffer
}

func (v *Device) Config() Config {
	return v.cfg
}
