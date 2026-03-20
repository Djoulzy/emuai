package video

import "sync"

type Frame struct {
	Sequence uint64
	Width    int
	Height   int
	Pixels   []uint32
}

type Framebuffer struct {
	mu     sync.RWMutex
	width  int
	height int
	pixels []uint32
}

func NewFramebuffer(width, height int) *Framebuffer {
	return &Framebuffer{
		width:  width,
		height: height,
		pixels: make([]uint32, width*height),
	}
}

func (f *Framebuffer) Size() (int, int) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.width, f.height
}

func (f *Framebuffer) Fill(value uint32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.pixels {
		f.pixels[i] = value
	}
}

func (f *Framebuffer) SetPixel(x, y int, value uint32) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if x < 0 || y < 0 || x >= f.width || y >= f.height {
		return false
	}
	f.pixels[(y*f.width)+x] = value
	return true
}

func (f *Framebuffer) Snapshot(sequence uint64) Frame {
	f.mu.RLock()
	defer f.mu.RUnlock()
	pixels := make([]uint32, len(f.pixels))
	copy(pixels, f.pixels)
	return Frame{
		Sequence: sequence,
		Width:    f.width,
		Height:   f.height,
		Pixels:   pixels,
	}
}
