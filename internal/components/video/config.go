package video

import "fmt"

const defaultClockHz uint64 = 1_000_000

type Backend string

const (
	BackendNull   Backend = "null"
	BackendVulkan Backend = "vulkan"
)

type CRTConfig struct {
	Width             int
	Height            int
	RefreshHz         int
	Curvature         float32
	ScanlineStrength  float32
	MaskStrength      float32
	GlowPersistence   float32
	HorizontalBlurTap float32
}

type Config struct {
	Backend Backend
	ClockHz uint64
	CRT     CRTConfig
	Trace   *TraceOverlay
	TraceOn bool
}

func DefaultCRTConfig() CRTConfig {
	return CRTConfig{
		Width:             320,
		Height:            240,
		RefreshHz:         60,
		Curvature:         0.18,
		ScanlineStrength:  0.35,
		MaskStrength:      0.22,
		GlowPersistence:   0.80,
		HorizontalBlurTap: 1.15,
	}
}

func DefaultConfig() Config {
	return Config{
		Backend: BackendNull,
		ClockHz: defaultClockHz,
		CRT:     DefaultCRTConfig(),
	}
}

func (cfg Config) normalized() Config {
	defaults := DefaultConfig()
	if cfg.Backend == "" {
		cfg.Backend = defaults.Backend
	}
	if cfg.ClockHz == 0 {
		cfg.ClockHz = defaults.ClockHz
	}
	if cfg.CRT.Width == 0 {
		cfg.CRT.Width = defaults.CRT.Width
	}
	if cfg.CRT.Height == 0 {
		cfg.CRT.Height = defaults.CRT.Height
	}
	if cfg.CRT.RefreshHz == 0 {
		cfg.CRT.RefreshHz = defaults.CRT.RefreshHz
	}
	if cfg.CRT.Curvature == 0 {
		cfg.CRT.Curvature = defaults.CRT.Curvature
	}
	if cfg.CRT.ScanlineStrength == 0 {
		cfg.CRT.ScanlineStrength = defaults.CRT.ScanlineStrength
	}
	if cfg.CRT.MaskStrength == 0 {
		cfg.CRT.MaskStrength = defaults.CRT.MaskStrength
	}
	if cfg.CRT.GlowPersistence == 0 {
		cfg.CRT.GlowPersistence = defaults.CRT.GlowPersistence
	}
	if cfg.CRT.HorizontalBlurTap == 0 {
		cfg.CRT.HorizontalBlurTap = defaults.CRT.HorizontalBlurTap
	}
	return cfg
}

func (cfg Config) validate() error {
	if cfg.ClockHz == 0 {
		return fmt.Errorf("video: clock frequency must be > 0")
	}
	if cfg.CRT.Width <= 0 || cfg.CRT.Height <= 0 {
		return fmt.Errorf("video: invalid framebuffer size %dx%d", cfg.CRT.Width, cfg.CRT.Height)
	}
	if cfg.CRT.RefreshHz <= 0 {
		return fmt.Errorf("video: refresh rate must be > 0")
	}
	if cfg.Backend != BackendNull && cfg.Backend != BackendVulkan {
		return fmt.Errorf("video: unsupported backend %q", cfg.Backend)
	}
	return nil
}
