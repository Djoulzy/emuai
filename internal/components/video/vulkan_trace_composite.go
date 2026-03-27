package video

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	vulkanTraceVisibleLines = 24
	vulkanTracePanelWidth   = 220
	tracePanelPadding       = 6
	tracePanelBlockGap      = 3
	tracePanelLineGap       = 0
	tracePanelHeaderGap     = 4
)

const (
	traceCompositeBackground  = 0xFF000000
	tracePanelBackground      = 0xFF0D1217
	tracePanelBlockBackground = 0xFF152029
	tracePanelDivider         = 0xFF2A3640
	tracePanelText            = 0xFF99F2AD
	tracePanelHeader          = 0xFFE7F0A3
	tracePanelMuted           = 0xFF7B8B97
)

var (
	tracePanelFace  font.Face
	traceFontHeight int
	traceFontAscent int
)

func init() {
	f, err := opentype.Parse(gomono.TTF)
	if err != nil {
		setFallbackFont()
		return
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    9,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		setFallbackFont()
		return
	}
	tracePanelFace = face
	m := face.Metrics()
	traceFontHeight = m.Height.Ceil()
	traceFontAscent = m.Ascent.Ceil()
}

func setFallbackFont() {
	tracePanelFace = basicfont.Face7x13
	traceFontHeight = basicfont.Face7x13.Height
	traceFontAscent = basicfont.Face7x13.Ascent
}

func composeFrameWithTrace(frame Frame, traceText string, traceStatus TraceStatus, traceOn bool) Frame {
	if frame.Width <= 0 || frame.Height <= 0 {
		return frame
	}

	compositeWidth := frame.Width + vulkanTracePanelWidth
	img := image.NewRGBA(image.Rect(0, 0, compositeWidth, frame.Height))
	draw.Draw(img, img.Bounds(), image.NewUniform(argbColor(traceCompositeBackground)), image.Point{}, draw.Src)

	copyCRTToRGBA(img, frame)
	drawTracePanel(img, frame.Width, traceText, defaultTraceStatus(traceOn, traceStatus), traceOn)

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

func drawTracePanel(dst *image.RGBA, xStart int, traceText string, traceStatus TraceStatus, traceOn bool) {
	panelRect := image.Rect(xStart, 0, dst.Bounds().Dx(), dst.Bounds().Dy())
	draw.Draw(dst, panelRect, image.NewUniform(argbColor(tracePanelBackground)), image.Point{}, draw.Src)

	dividerRect := image.Rect(xStart, 0, xStart+2, dst.Bounds().Dy())
	draw.Draw(dst, dividerRect, image.NewUniform(argbColor(tracePanelDivider)), image.Point{}, draw.Src)

	textLeft := xStart + tracePanelPadding
	lineHeight := traceFontHeight + tracePanelLineGap

	drawer := &font.Drawer{
		Dst:  dst,
		Face: tracePanelFace,
	}

	statusBottom := drawTraceStatusBlocks(dst, drawer, textLeft, tracePanelPadding, traceStatus, traceOn)
	dividerY := statusBottom + tracePanelHeaderGap/2
	draw.Draw(
		dst,
		image.Rect(textLeft, dividerY, dst.Bounds().Dx()-tracePanelPadding, dividerY+1),
		image.NewUniform(argbColor(tracePanelDivider)),
		image.Point{},
		draw.Src,
	)

	linesTop := dividerY + tracePanelHeaderGap/2 + traceFontAscent
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

func drawTraceStatusBlocks(dst *image.RGBA, drawer *font.Drawer, textLeft int, top int, traceStatus TraceStatus, traceOn bool) int {
	panelRight := dst.Bounds().Dx() - tracePanelPadding
	blockHeight := traceFontHeight*2 + 2
	flagsWidth := 36
	if !traceOn {
		flagsWidth = 42
	}

	flagsRect := image.Rect(panelRight-flagsWidth, top, panelRight, top+blockHeight)
	registerLeft := textLeft
	registerRight := flagsRect.Min.X - tracePanelBlockGap
	if registerRight < registerLeft {
		registerRight = registerLeft
	}
	drawRegisterBlocks(dst, drawer, image.Rect(registerLeft, top, registerRight, top+blockHeight), splitRegisterValues(traceStatus.Registers))
	drawStatusBlock(dst, drawer, flagsRect, "P", traceStatus.Flags)

	return flagsRect.Max.Y
}

func drawRegisterBlocks(dst *image.RGBA, drawer *font.Drawer, rect image.Rectangle, registers []registerValue) {
	if rect.Dx() <= 0 {
		return
	}

	draw.Draw(dst, rect, image.NewUniform(argbColor(tracePanelBlockBackground)), image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(rect.Min.X, rect.Max.Y-1, rect.Max.X, rect.Max.Y), image.NewUniform(argbColor(tracePanelDivider)), image.Point{}, draw.Src)

	if len(registers) == 0 {
		return
	}

	columnWidth := rect.Dx() / len(registers)
	for idx, register := range registers {
		colLeft := rect.Min.X + idx*columnWidth
		if idx > 0 {
			draw.Draw(dst, image.Rect(colLeft, rect.Min.Y+2, colLeft+1, rect.Max.Y-2), image.NewUniform(argbColor(tracePanelDivider)), image.Point{}, draw.Src)
		}
		drawStatusText(drawer, colLeft+2, rect.Min.Y, register.label, register.value)
	}
}

func drawStatusBlock(dst *image.RGBA, drawer *font.Drawer, rect image.Rectangle, label string, value string) {
	draw.Draw(dst, rect, image.NewUniform(argbColor(tracePanelBlockBackground)), image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(rect.Min.X, rect.Max.Y-1, rect.Max.X, rect.Max.Y), image.NewUniform(argbColor(tracePanelDivider)), image.Point{}, draw.Src)
	drawStatusText(drawer, rect.Min.X+2, rect.Min.Y, label, value)
}

func drawStatusText(drawer *font.Drawer, textLeft int, top int, label string, value string) {

	labelBaseline := top + traceFontAscent
	valueBaseline := labelBaseline + traceFontHeight

	drawer.Src = image.NewUniform(argbColor(tracePanelHeader))
	drawer.Dot = fixed.P(textLeft, labelBaseline)
	drawer.DrawString(label)

	drawer.Src = image.NewUniform(argbColor(tracePanelText))
	drawer.Dot = fixed.P(textLeft, valueBaseline)
	drawer.DrawString(value)
}

type registerValue struct {
	label string
	value string
}

func splitRegisterValues(registers string) []registerValue {
	fields := strings.Fields(registers)
	if len(fields) == 0 {
		return []registerValue{{label: "A", value: "--"}, {label: "X", value: "--"}, {label: "Y", value: "--"}, {label: "SP", value: "--"}, {label: "P", value: "--"}}
	}

	values := make([]registerValue, 0, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, ":", 2)
		if len(parts) != 2 {
			continue
		}
		values = append(values, registerValue{label: parts[0], value: parts[1]})
	}
	if len(values) == 0 {
		return []registerValue{{label: "A", value: "--"}, {label: "X", value: "--"}, {label: "Y", value: "--"}, {label: "SP", value: "--"}, {label: "P", value: "--"}}
	}
	return values
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
