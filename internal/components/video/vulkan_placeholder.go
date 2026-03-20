//go:build vulkan

package video

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/go-gl/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

type VulkanRenderer struct {
	cfg         Config
	window      *glfw.Window
	instance    vk.Instance
	surface     vk.Surface
	deviceName  string
	frameCount  uint64
	initGLFW    bool
	titlePrefix string
	mu          sync.Mutex
}

func newVulkanRenderer(cfg Config) (Renderer, error) {
	return &VulkanRenderer{cfg: cfg}, nil
}

func (r *VulkanRenderer) Name() string { return string(BackendVulkan) }

func (r *VulkanRenderer) Init(cfg Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cfg = cfg
	if r.window != nil {
		return nil
	}

	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		runtime.UnlockOSThread()
		return fmt.Errorf("video: glfw init failed: %w", err)
	}
	r.initGLFW = true

	if !glfw.VulkanSupported() {
		r.closeLocked()
		runtime.UnlockOSThread()
		return fmt.Errorf("video: glfw reports that Vulkan is not available on this system")
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.True)

	windowWidth, windowHeight := scaledWindowSize(cfg)
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "emuai CRT [Vulkan]", nil, nil)
	if err != nil {
		r.closeLocked()
		runtime.UnlockOSThread()
		return fmt.Errorf("video: could not create Vulkan window: %w", err)
	}
	r.window = window

	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	if err := vk.Init(); err != nil {
		r.closeLocked()
		runtime.UnlockOSThread()
		return fmt.Errorf("video: vulkan init failed: %w", err)
	}

	instance, err := createVulkanInstance(window)
	if err != nil {
		r.closeLocked()
		runtime.UnlockOSThread()
		return err
	}
	r.instance = instance

	if err := vk.InitInstance(instance); err != nil {
		r.closeLocked()
		runtime.UnlockOSThread()
		return fmt.Errorf("video: vulkan instance init failed: %w", err)
	}

	surfacePtr, err := window.CreateWindowSurface(&instance, nil)
	if err != nil {
		r.closeLocked()
		runtime.UnlockOSThread()
		return fmt.Errorf("video: could not create Vulkan surface for GLFW window: %w", err)
	}
	r.surface = vk.SurfaceFromPointer(surfacePtr)

	deviceName, err := firstPhysicalDeviceName(instance)
	if err != nil {
		r.closeLocked()
		runtime.UnlockOSThread()
		return err
	}
	r.deviceName = deviceName
	r.titlePrefix = fmt.Sprintf("emuai CRT [Vulkan] %s", deviceName)
	r.window.SetTitle(fmt.Sprintf("%s - %dx%d @ %d Hz", r.titlePrefix, cfg.CRT.Width, cfg.CRT.Height, cfg.CRT.RefreshHz))
	runtime.UnlockOSThread()
	return nil
}

func (r *VulkanRenderer) Present(frame Frame) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.window == nil {
		return fmt.Errorf("video: vulkan renderer is not initialized")
	}

	r.frameCount = frame.Sequence
	if frame.Sequence == 1 || frame.Sequence%30 == 0 {
		r.window.SetTitle(fmt.Sprintf("%s - frame %d - %dx%d", r.titlePrefix, frame.Sequence, frame.Width, frame.Height))
	}
	glfw.PollEvents()
	if r.window.ShouldClose() {
		return fmt.Errorf("video: vulkan window was closed")
	}
	return nil
}

func (r *VulkanRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeLocked()
	return nil
}

func (r *VulkanRenderer) closeLocked() {
	var zeroInstance vk.Instance
	var zeroSurface vk.Surface

	if r.surface != zeroSurface && r.instance != zeroInstance {
		vk.DestroySurface(r.instance, r.surface, nil)
		r.surface = zeroSurface
	}
	if r.window != nil {
		r.window.Destroy()
		r.window = nil
	}
	if r.instance != zeroInstance {
		vk.DestroyInstance(r.instance, nil)
		r.instance = zeroInstance
	}
	if r.initGLFW {
		glfw.Terminate()
		r.initGLFW = false
	}
}

func scaledWindowSize(cfg Config) (int, int) {
	width := cfg.CRT.Width * 3
	height := cfg.CRT.Height * 3
	if width < 640 {
		width = 640
	}
	if height < 480 {
		height = 480
	}
	return width, height
}

func createVulkanInstance(window *glfw.Window) (vk.Instance, error) {
	var zeroInstance vk.Instance

	extensions := window.GetRequiredInstanceExtensions()
	if len(extensions) == 0 {
		return zeroInstance, fmt.Errorf("video: GLFW did not return any Vulkan instance extensions for window surface creation")
	}

	appInfo := vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		PApplicationName:   "emuai",
		ApplicationVersion: vk.MakeVersion(0, 1, 0),
		PEngineName:        "emuai",
		EngineVersion:      vk.MakeVersion(0, 1, 0),
		ApiVersion:         vk.MakeVersion(1, 0, 0),
	}
	createInfo := vk.InstanceCreateInfo{
		SType:                   vk.StructureTypeInstanceCreateInfo,
		PApplicationInfo:        &appInfo,
		EnabledExtensionCount:   uint32(len(extensions)),
		PpEnabledExtensionNames: extensions,
	}

	var instance vk.Instance
	if result := vk.CreateInstance(&createInfo, nil, &instance); result != vk.Success {
		return zeroInstance, fmt.Errorf("video: vkCreateInstance failed: %w", vk.Error(result))
	}
	return instance, nil
}

func firstPhysicalDeviceName(instance vk.Instance) (string, error) {
	var deviceCount uint32
	if result := vk.EnumeratePhysicalDevices(instance, &deviceCount, nil); result != vk.Success {
		return "", fmt.Errorf("video: could not enumerate Vulkan physical devices: %w", vk.Error(result))
	}
	if deviceCount == 0 {
		return "", fmt.Errorf("video: no Vulkan physical device is available")
	}

	devices := make([]vk.PhysicalDevice, deviceCount)
	if result := vk.EnumeratePhysicalDevices(instance, &deviceCount, devices); result != vk.Success {
		return "", fmt.Errorf("video: could not read Vulkan physical devices: %w", vk.Error(result))
	}

	var properties vk.PhysicalDeviceProperties
	vk.GetPhysicalDeviceProperties(devices[0], &properties)
	name := strings.TrimRight(string(properties.DeviceName[:]), "\x00")
	if name == "" {
		name = "unknown GPU"
	}
	return name, nil
}
