package video

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	vulkanTraceVisibleLines = 24
	vulkanTracePanelWidth   = 520
	tracePanelPadding       = 18
	tracePanelLineGap       = 3
	tracePanelHeaderGap     = 12
)

const (
	traceCompositeBackground = 0xFF000000
	tracePanelBackground     = 0xFF0D1217
	tracePanelDivider        = 0xFF2A3640
	tracePanelText           = 0xFF99F2AD
	tracePanelHeader         = 0xFFE7F0A3
)

var tracePanelFace = basicfont.Face7x13

func composeFrameWithTrace(frame Frame, traceText string, traceOn bool) Frame {
	if frame.Width <= 0 || frame.Height <= 0 {
		return frame
	}

	compositeWidth := frame.Width + vulkanTracePanelWidth
	img := image.NewRGBA(image.Rect(0, 0, compositeWidth, frame.Height))
	draw.Draw(img, img.Bounds(), image.NewUniform(argbColor(traceCompositeBackground)), image.Point{}, draw.Src)

	copyCRTToRGBA(img, frame)
	drawTracePanel(img, frame.Width, traceText, traceOn)

	pixels := make([]uint32, compositeWidth*frame.Height)
	for y := 0; y < frame.Height; y++ {
		rowOffset := y * img.Stride
		pixelOffset := y * compositeWidth
		for x := 0; x < compositeWidth; x++ {
			base := rowOffset + x*4
			pixels[pixelOffset+x] = rgbaBytesToARGB(img.Pix[base], img.Pix[base+1], img.Pix[base+2], img.Pix[base+3])
		}
	}

	return Frame{
		Sequence: frame.Sequence,
		Width:    compositeWidth,
		Height:   frame.Height,
		Pixels:   pixels,
	}
}

func copyCRTToRGBA(dst *image.RGBA, frame Frame) {
	for y := 0; y < frame.Height; y++ {
		pixelOffset := y * frame.Width
		for x := 0; x < frame.Width; x++ {
			dst.SetRGBA(x, y, argbColor(frame.Pixels[pixelOffset+x]))
		}
	}
}

func drawTracePanel(dst *image.RGBA, xStart int, traceText string, traceOn bool) {
	panelRect := image.Rect(xStart, 0, dst.Bounds().Dx(), dst.Bounds().Dy())
	draw.Draw(dst, panelRect, image.NewUniform(argbColor(tracePanelBackground)), image.Point{}, draw.Src)

	dividerRect := image.Rect(xStart, 0, xStart+2, dst.Bounds().Dy())
	draw.Draw(dst, dividerRect, image.NewUniform(argbColor(tracePanelDivider)), image.Point{}, draw.Src)

	textLeft := xStart + tracePanelPadding
	textTop := tracePanelPadding + tracePanelFace.Ascent
	lineHeight := tracePanelFace.Height + tracePanelLineGap

	drawer := &font.Drawer{
		Dst:  dst,
		Face: tracePanelFace,
	}

	drawer.Src = image.NewUniform(argbColor(tracePanelHeader))
	drawer.Dot = fixed.P(textLeft, textTop)
	drawer.DrawString("CPU TRACE")

	dividerY := textTop + tracePanelHeaderGap
	draw.Draw(
		dst,
		image.Rect(textLeft, dividerY, dst.Bounds().Dx()-tracePanelPadding, dividerY+1),
		image.NewUniform(argbColor(tracePanelDivider)),
		image.Point{},
		draw.Src,
	)

	linesTop := dividerY + tracePanelHeaderGap + tracePanelFace.Ascent
	availableHeight := dst.Bounds().Dy() - linesTop - tracePanelPadding
	if availableHeight <= 0 {
		return
	}
	maxLines := availableHeight / lineHeight
	if maxLines <= 0 {
		return
	}

	lines := visibleTraceLines(traceText, maxLines, traceOn)
	drawer.Src = image.NewUniform(argbColor(tracePanelText))
	for idx, line := range lines {
		drawer.Dot = fixed.P(textLeft, linesTop+idx*lineHeight)
		drawer.DrawString(line)
	}
}

func visibleTraceLines(traceText string, maxLines int, traceOn bool) []string {
	if !traceOn {
		traceText = "TRACE DISABLED\n\nRelance avec -trace"
	} else if strings.TrimSpace(traceText) == "" {
		traceText = "TRACE ACTIVE\n\nWAITING FOR CPU..."
	}

	lines := strings.Split(traceText, "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines
}

func argbColor(value uint32) color.RGBA {
	return color.RGBA{
		R: uint8(value >> 16),
		G: uint8(value >> 8),
		B: uint8(value),
		A: uint8(value >> 24),
	}
}

func rgbaBytesToARGB(r, g, b, a byte) uint32 {
	return uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}
