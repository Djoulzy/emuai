//go:build vulkan && !darwin

package video

import "github.com/go-gl/glfw/v3.3/glfw"

func presentFrameInWindow(_ *glfw.Window, _ Frame) {
}