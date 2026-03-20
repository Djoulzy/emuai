package video

import (
	"context"
	"strings"
	"testing"

	"github.com/Djoulzy/emuai/internal/emulator"
)

type spyRenderer struct {
	frames []Frame
}

func (r *spyRenderer) Name() string { return "spy" }

func (r *spyRenderer) Init(_ Config) error { return nil }

func (r *spyRenderer) Present(frame Frame) error {
	r.frames = append(r.frames, frame)
	return nil
}

func (r *spyRenderer) Close() error { return nil }

func TestFramebufferSetPixelAndSnapshot(t *testing.T) {
	fb := NewFramebuffer(4, 3)
	if ok := fb.SetPixel(2, 1, 0xFF00FF00); !ok {
		t.Fatal("expected in-bounds write to succeed")
	}
	if ok := fb.SetPixel(4, 1, 0xFFFFFFFF); ok {
		t.Fatal("expected out-of-bounds write to fail")
	}

	snapshot := fb.Snapshot(7)
	if snapshot.Sequence != 7 {
		t.Fatalf("expected sequence 7, got %d", snapshot.Sequence)
	}
	if snapshot.Width != 4 || snapshot.Height != 3 {
		t.Fatalf("unexpected snapshot size %dx%d", snapshot.Width, snapshot.Height)
	}
	if got := snapshot.Pixels[(1*4)+2]; got != 0xFF00FF00 {
		t.Fatalf("expected snapshot pixel 0xFF00FF00, got 0x%08X", got)
	}
}

func TestDevicePresentsFramesAtRefreshCadence(t *testing.T) {
	renderer := &spyRenderer{}
	device := &Device{
		name:           "video-test",
		cfg:            Config{Backend: BackendNull, ClockHz: 600, CRT: CRTConfig{Width: 4, Height: 4, RefreshHz: 60}},
		renderer:       renderer,
		framebuffer:    NewFramebuffer(4, 4),
		cyclesPerFrame: 10,
	}
	if err := device.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	for cycle := uint64(0); cycle <= 20; cycle++ {
		if err := device.Tick(context.Background(), emulator.Tick{Cycle: cycle}, nil); err != nil {
			t.Fatalf("tick %d: %v", cycle, err)
		}
	}

	if len(renderer.frames) != 2 {
		t.Fatalf("expected 2 presented frames, got %d", len(renderer.frames))
	}
	if renderer.frames[0].Sequence != 1 || renderer.frames[1].Sequence != 2 {
		t.Fatalf("unexpected frame sequences: %+v", renderer.frames)
	}
}

func TestNewDeviceRejectsUnknownBackend(t *testing.T) {
	_, err := NewDevice("video-test", Config{Backend: Backend("mystery")})
	if err == nil {
		t.Fatal("expected unsupported backend error")
	}
}

func TestNewDeviceVulkanBackendReturnsExplicitStatus(t *testing.T) {
	_, err := NewDevice("video-test", Config{Backend: BackendVulkan})
	if err == nil {
		t.Fatal("expected vulkan backend error without build tag")
	}
	if !strings.Contains(err.Error(), "-tags vulkan") && !strings.Contains(err.Error(), "not implemented yet") {
		t.Fatalf("expected build-tag hint, got %v", err)
	}
}
