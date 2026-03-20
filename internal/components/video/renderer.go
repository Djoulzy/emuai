package video

import "fmt"

type Renderer interface {
	Name() string
	Init(cfg Config) error
	Present(frame Frame) error
	Close() error
}

type NullRenderer struct{}

func NewNullRenderer() *NullRenderer {
	return &NullRenderer{}
}

func (r *NullRenderer) Name() string { return string(BackendNull) }

func (r *NullRenderer) Init(_ Config) error { return nil }

func (r *NullRenderer) Present(_ Frame) error { return nil }

func (r *NullRenderer) Close() error { return nil }

func newRenderer(cfg Config) (Renderer, error) {
	switch cfg.Backend {
	case BackendNull:
		return NewNullRenderer(), nil
	case BackendVulkan:
		return newVulkanRenderer(cfg)
	default:
		return nil, fmt.Errorf("video: unsupported backend %q", cfg.Backend)
	}
}
