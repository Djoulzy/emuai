//go:build !vulkan

package video

import "fmt"

func newVulkanRenderer(_ Config) (Renderer, error) {
	return nil, fmt.Errorf("video: vulkan backend scaffold is present but not enabled in this build; rebuild with -tags vulkan once the concrete Vulkan renderer is wired")
}
