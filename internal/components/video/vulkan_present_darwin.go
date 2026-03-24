//go:build vulkan && darwin

package video

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore -framework CoreGraphics -framework Foundation
#include <stdint.h>
#include <stdlib.h>
#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import <CoreGraphics/CoreGraphics.h>

static void cPresentSoftwareFrame(void *nsWindow, const uint32_t *pixels, int width, int height) {
	if (nsWindow == NULL || pixels == NULL || width <= 0 || height <= 0) {
		return;
	}

	size_t pixelCount = (size_t)width * (size_t)height;
	size_t byteCount = pixelCount * 4;
	uint8_t *rgba = (uint8_t *)malloc(byteCount);
	if (rgba == NULL) {
		return;
	}

	for (size_t i = 0; i < pixelCount; i++) {
		uint32_t pixel = pixels[i];
		rgba[(i * 4) + 0] = (uint8_t)((pixel >> 16) & 0xFF);
		rgba[(i * 4) + 1] = (uint8_t)((pixel >> 8) & 0xFF);
		rgba[(i * 4) + 2] = (uint8_t)(pixel & 0xFF);
		rgba[(i * 4) + 3] = (uint8_t)((pixel >> 24) & 0xFF);
	}

	CGColorSpaceRef colorSpace = CGColorSpaceCreateDeviceRGB();
	if (colorSpace == NULL) {
		free(rgba);
		return;
	}

	CGContextRef bitmap = CGBitmapContextCreate(
		rgba,
		(size_t)width,
		(size_t)height,
		8,
		(size_t)width * 4,
		colorSpace,
		kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big
	);
	if (bitmap == NULL) {
		CGColorSpaceRelease(colorSpace);
		free(rgba);
		return;
	}

	CGImageRef image = CGBitmapContextCreateImage(bitmap);
	if (image == NULL) {
		CGContextRelease(bitmap);
		CGColorSpaceRelease(colorSpace);
		free(rgba);
		return;
	}

	NSWindow *window = (__bridge NSWindow *)nsWindow;
	NSView *view = [window contentView];
	[view setWantsLayer:YES];
	if (view.layer == nil) {
		view.layer = [CALayer layer];
	}
	view.layer.contentsGravity = kCAGravityResizeAspect;
	view.layer.backgroundColor = CGColorGetConstantColor(kCGColorBlack);
	view.layer.contents = (__bridge id)image;
	[view.layer setNeedsDisplay];
	[view setNeedsDisplay:YES];

	CGImageRelease(image);
	CGContextRelease(bitmap);
	CGColorSpaceRelease(colorSpace);
	free(rgba);
}
*/
import "C"

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func presentFrameInWindow(window *glfw.Window, frame Frame) {
	if window == nil || len(frame.Pixels) == 0 {
		return
	}

	nativeWindow := window.GetCocoaWindow()
	if nativeWindow == nil {
		return
	}

	C.cPresentSoftwareFrame(nativeWindow, (*C.uint32_t)(unsafe.Pointer(&frame.Pixels[0])), C.int(frame.Width), C.int(frame.Height))
}