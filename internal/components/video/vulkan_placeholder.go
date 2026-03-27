//go:build vulkan

package video

/*
#cgo CFLAGS: -DVK_NO_PROTOTYPES
#include <vulkan/vulkan.h>
#include <string.h>
#include <stdlib.h>

static PFN_vkCreateInstance pfn_vkCreateInstance = NULL;

void cSetCreateInstanceProc(void* proc) {
    pfn_vkCreateInstance = (PFN_vkCreateInstance)proc;
}

// Resolve vkCreateInstance via a provided vkGetInstanceProcAddr.
static void* cResolveCreateInstance(void* getInstProcAddr, const char* name) {
    typedef PFN_vkVoidFunction (*PFN_vkGetInstanceProcAddr_t)(VkInstance, const char*);
    PFN_vkGetInstanceProcAddr_t fn = (PFN_vkGetInstanceProcAddr_t)getInstProcAddr;
    return (void*)fn(NULL, name);
}

VkResult cCreateInstanceWithPortability(
    const char** extensions, uint32_t extensionCount,
    VkInstanceCreateFlags flags,
    VkInstance* instance
) {
    if (!pfn_vkCreateInstance) {
        return VK_ERROR_INITIALIZATION_FAILED;
    }

    VkApplicationInfo appInfo;
    memset(&appInfo, 0, sizeof(appInfo));
    appInfo.sType = VK_STRUCTURE_TYPE_APPLICATION_INFO;
    appInfo.pApplicationName = "emuai";
    appInfo.applicationVersion = VK_MAKE_VERSION(0, 1, 0);
    appInfo.pEngineName = "emuai";
    appInfo.engineVersion = VK_MAKE_VERSION(0, 1, 0);
    appInfo.apiVersion = VK_MAKE_VERSION(1, 0, 0);

    VkInstanceCreateInfo createInfo;
    memset(&createInfo, 0, sizeof(createInfo));
    createInfo.sType = VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO;
    createInfo.flags = flags;
    createInfo.pApplicationInfo = &appInfo;
    createInfo.enabledExtensionCount = extensionCount;
    createInfo.ppEnabledExtensionNames = extensions;

    return pfn_vkCreateInstance(&createInfo, NULL, instance);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

const (
	vkKhrPortabilityEnumerationExtensionName = "VK_KHR_portability_enumeration"
	vkInstanceCreateEnumeratePortabilityBit  = vk.InstanceCreateFlags(0x00000001)
)

// mainThreadFunc is a function to execute on the GLFW main thread together with
// a channel to signal completion.
type mainThreadFunc struct {
	fn   func()
	done chan struct{}
}

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

	// mainCh dispatches work to the goroutine that owns the OS main thread.
	// Closed by Close to terminate the dispatch loop.
	mainCh chan mainThreadFunc
}

func newVulkanRenderer(cfg Config) (Renderer, error) {
	return &VulkanRenderer{cfg: cfg, mainCh: make(chan mainThreadFunc, 4)}, nil
}

func (r *VulkanRenderer) Name() string { return string(BackendVulkan) }

// doOnMainThread sends fn to the OS-thread-locked dispatch goroutine and blocks
// until fn has finished executing.
func (r *VulkanRenderer) doOnMainThread(fn func()) {
	done := make(chan struct{})
	r.mainCh <- mainThreadFunc{fn: fn, done: done}
	<-done
}

func (r *VulkanRenderer) Init(cfg Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cfg = cfg
	if r.window != nil {
		return nil
	}

	// Run the full GLFW/Vulkan initialization on the main thread goroutine
	// and keep that goroutine alive to service future GLFW calls.
	initErr := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		// This goroutine now owns an OS thread permanently; it will serve
		// as the GLFW main thread until the channel is closed.

		err := r.initOnMainThread(cfg)
		initErr <- err
		if err != nil {
			return
		}

		// Dispatch loop: execute functions sent from other goroutines.
		for work := range r.mainCh {
			work.fn()
			close(work.done)
		}
	}()

	return <-initErr
}

// initOnMainThread performs all GLFW/Vulkan setup. Must be called on the
// OS-thread-locked goroutine.
func (r *VulkanRenderer) initOnMainThread(cfg Config) error {
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("video: glfw init failed: %w", err)
	}
	r.initGLFW = true

	if !glfw.VulkanSupported() {
		r.closeLocked()
		return fmt.Errorf("video: glfw reports that Vulkan is not available on this system")
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.True)

	windowWidth, windowHeight := scaledWindowSize(cfg)
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "emuai CRT [Vulkan]", nil, nil)
	if err != nil {
		r.closeLocked()
		return fmt.Errorf("video: could not create Vulkan window: %w", err)
	}
	r.window = window

	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	if err := vk.Init(); err != nil {
		r.closeLocked()
		return fmt.Errorf("video: vulkan init failed: %w", err)
	}

	instance, err := createVulkanInstance(window)
	if err != nil {
		r.closeLocked()
		return err
	}
	r.instance = instance

	if err := vk.InitInstance(instance); err != nil {
		r.closeLocked()
		return fmt.Errorf("video: vulkan instance init failed: %w", err)
	}

	surfacePtr, err := window.CreateWindowSurface(instance, nil)
	if err != nil {
		r.closeLocked()
		return fmt.Errorf("video: could not create Vulkan surface for GLFW window: %w", err)
	}
	r.surface = vk.SurfaceFromPointer(surfacePtr)

	deviceName, err := firstPhysicalDeviceName(instance)
	if err != nil {
		r.closeLocked()
		return err
	}
	r.deviceName = deviceName
	r.titlePrefix = fmt.Sprintf("emuai CRT [Vulkan] %s", deviceName)
	r.window.SetTitle(fmt.Sprintf("%s - %dx%d @ %d Hz", r.titlePrefix, cfg.CRT.Width, cfg.CRT.Height, cfg.CRT.RefreshHz))
	return nil
}

func (r *VulkanRenderer) Present(frame Frame) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.window == nil {
		return fmt.Errorf("video: vulkan renderer is not initialized")
	}

	var presentErr error
	r.doOnMainThread(func() {
		r.frameCount = frame.Sequence
		traceText := ""
		traceStatus := TraceStatus{}
		if r.cfg.Trace != nil {
			traceText = r.cfg.Trace.Text(vulkanTraceVisibleLines)
			traceStatus = r.cfg.Trace.Status()
		}
		presentedFrame := composeFrameWithTrace(frame, traceText, traceStatus, r.cfg.TraceOn)
		presentFrameInWindow(r.window, presentedFrame)
		if frame.Sequence == 1 || frame.Sequence%30 == 0 {
			w := presentedFrame.Width
			h := presentedFrame.Height
			r.window.SetTitle(fmt.Sprintf("%s - frame %d - %dx%d", r.titlePrefix, frame.Sequence, w, h))
		}
		glfw.PollEvents()
		if r.window.ShouldClose() {
			presentErr = fmt.Errorf("video: vulkan window was closed")
		}
	})
	return presentErr
}

func (r *VulkanRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.mainCh == nil {
		return nil
	}

	// Send cleanup work to the main thread, then close the channel to
	// terminate the dispatch goroutine.
	r.doOnMainThread(func() {
		r.closeLocked()
	})
	close(r.mainCh)
	r.mainCh = nil
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
	width += vulkanTracePanelWidth
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

	extensions, flags := requiredInstanceConfiguration(window)
	if len(extensions) == 0 {
		return zeroInstance, fmt.Errorf("video: GLFW did not return any Vulkan instance extensions for window surface creation")
	}

	// Obtain the vkCreateInstance function pointer from GLFW's loader and
	// pass it into our C helper so that vkCreateInstance is called directly
	// from C, bypassing vulkan-go's Go→C marshaling which may drop the
	// InstanceCreateFlags on this binding version.
	procAddr := glfw.GetVulkanGetInstanceProcAddress()
	if procAddr == nil {
		return zeroInstance, fmt.Errorf("video: GLFW returned nil vkGetInstanceProcAddr")
	}

	// Resolve vkCreateInstance via the global vkGetInstanceProcAddr(NULL, "vkCreateInstance").
	cName := C.CString("vkCreateInstance")
	defer C.free(unsafe.Pointer(cName))
	createInstProc := C.cResolveCreateInstance(unsafe.Pointer(procAddr), cName)
	if createInstProc == nil {
		return zeroInstance, fmt.Errorf("video: vkGetInstanceProcAddr(NULL, \"vkCreateInstance\") returned NULL")
	}
	C.cSetCreateInstanceProc(createInstProc)

	// Convert Go extension strings to C strings.
	cExtensions := make([]*C.char, len(extensions))
	for i, ext := range extensions {
		cExtensions[i] = C.CString(ext)
	}
	defer func() {
		for _, p := range cExtensions {
			C.free(unsafe.Pointer(p))
		}
	}()

	var cInstance C.VkInstance
	result := C.cCreateInstanceWithPortability(
		(**C.char)(unsafe.Pointer(&cExtensions[0])),
		C.uint32_t(len(extensions)),
		C.VkInstanceCreateFlags(flags),
		&cInstance,
	)
	if result != C.VK_SUCCESS {
		return zeroInstance, fmt.Errorf("video: vkCreateInstance failed: %w", vk.Error(vk.Result(result)))
	}

	// Convert the C VkInstance handle into a vulkan-go vk.Instance.
	// Both are typedef'd pointers (dispatchable handles), same size.
	instance := *(*vk.Instance)(unsafe.Pointer(&cInstance))
	return instance, nil
}

func requiredInstanceConfiguration(window *glfw.Window) ([]string, vk.InstanceCreateFlags) {
	extensions := append([]string(nil), window.GetRequiredInstanceExtensions()...)
	var flags vk.InstanceCreateFlags

	if runtime.GOOS == "darwin" {
		extensions = appendExtensionIfMissing(extensions, vk.KhrGetPhysicalDeviceProperties2ExtensionName)
		extensions = appendExtensionIfMissing(extensions, vkKhrPortabilityEnumerationExtensionName)
		flags |= vkInstanceCreateEnumeratePortabilityBit
	}

	return extensions, flags
}

func appendExtensionIfMissing(extensions []string, extension string) []string {
	for _, existing := range extensions {
		if existing == extension {
			return extensions
		}
	}
	return append(extensions, extension)
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
