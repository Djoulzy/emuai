package video

import (
	"context"
	"fmt"
	"strings"

	"github.com/Djoulzy/emuai/internal/emulator"
)

const (
	appleIIeTextPageSize  = 0x0400
	appleIIeHiResPageSize = 0x2000
	appleIIeCharROMSize   = 256 * 8

	appleIIeSwitch80ColOff uint16 = 0xC00C
	appleIIeSwitch80ColOn  uint16 = 0xC00D
	appleIIeSwitchGraphics uint16 = 0xC050
	appleIIeSwitchText     uint16 = 0xC051
	appleIIeSwitchFull     uint16 = 0xC052
	appleIIeSwitchMixed    uint16 = 0xC053
	appleIIeSwitchPage1    uint16 = 0xC054
	appleIIeSwitchPage2    uint16 = 0xC055
	appleIIeSwitchLoRes    uint16 = 0xC056
	appleIIeSwitchHiRes    uint16 = 0xC057
	appleIIeSwitchVBlank   uint16 = 0xC019
)

const (
	appleIIeRegHorizontalTotal = iota
	appleIIeRegHorizontalDisplayed
	appleIIeRegHorizontalSyncPosition
	appleIIeRegSyncWidths
	appleIIeRegVerticalTotal
	appleIIeRegVerticalAdjust
	appleIIeRegVerticalDisplayed
	appleIIeRegVerticalSyncPosition
	appleIIeRegInterlaceAndSkew
	appleIIeRegMaxRasterAddress
	appleIIeRegCursorStart
	appleIIeRegCursorEnd
	appleIIeRegStartAddressHigh
	appleIIeRegStartAddressLow
	appleIIeRegCursorHigh
	appleIIeRegCursorLow
	appleIIeRegLightPenHigh
	appleIIeRegLightPenLow
)

type appleIIeRenderMode int

const (
	appleIIeRenderText40 appleIIeRenderMode = iota
	appleIIeRenderText80
	appleIIeRenderLoRes
	appleIIeRenderLoRes80
	appleIIeRenderHiRes
	appleIIeRenderDoubleHiRes
)

type AppleIIeDisplayMode struct {
	Text        bool
	HiRes       bool
	Mixed       bool
	Page2       bool
	Columns80   bool
	DoubleWidth bool
}

type AppleIIeMemory struct {
	MainText  [2][]byte
	AuxText   [2][]byte
	MainHiRes [2][]byte
	AuxHiRes  [2][]byte
}

type AppleIIeOptions struct {
	ColorDisplay bool
	CharacterROM []byte
	Memory       AppleIIeMemory
}

type AppleIIeCRTC struct {
	*Device

	Reg [18]byte

	bus *emulator.Bus

	mode         AppleIIeDisplayMode
	renderMode   appleIIeRenderMode
	memory       AppleIIeMemory
	characterROM []byte

	videoMainMem []byte
	videoAuxMem  []byte

	textColor   uint32
	blinkOn     bool
	blinkFrames uint64

	pixelSize    byte
	screenWidth  int
	screenHeight int

	BeamX       int
	BeamY       int
	RasterLine  int
	RasterCount int
	CCLK        byte
	VBL         byte
}

func DefaultAppleIIeConfig() Config {
	crt := DefaultCRTConfig()
	crt.Width = 560
	crt.Height = 384
	crt.RefreshHz = 60
	return Config{
		Backend: BackendNull,
		ClockHz: defaultClockHz,
		CRT:     crt,
	}
}

func NewAppleIIeCRTC(name string, cfg Config, options AppleIIeOptions) (*AppleIIeCRTC, error) {
	return newAppleIIeCRTC(name, cfg, options, nil)
}

func newAppleIIeCRTC(name string, cfg Config, options AppleIIeOptions, renderer Renderer) (*AppleIIeCRTC, error) {
	if name == "" {
		return nil, fmt.Errorf("video: apple IIe CRTC requires a device name")
	}

	base := DefaultAppleIIeConfig()
	if cfg.Backend != "" {
		base.Backend = cfg.Backend
	}
	if cfg.ClockHz != 0 {
		base.ClockHz = cfg.ClockHz
	}
	if cfg.CRT.Width != 0 {
		base.CRT.Width = cfg.CRT.Width
	}
	if cfg.CRT.Height != 0 {
		base.CRT.Height = cfg.CRT.Height
	}
	if cfg.CRT.RefreshHz != 0 {
		base.CRT.RefreshHz = cfg.CRT.RefreshHz
	}
	if cfg.CRT.Curvature != 0 {
		base.CRT.Curvature = cfg.CRT.Curvature
	}
	if cfg.CRT.ScanlineStrength != 0 {
		base.CRT.ScanlineStrength = cfg.CRT.ScanlineStrength
	}
	if cfg.CRT.MaskStrength != 0 {
		base.CRT.MaskStrength = cfg.CRT.MaskStrength
	}
	if cfg.CRT.GlowPersistence != 0 {
		base.CRT.GlowPersistence = cfg.CRT.GlowPersistence
	}
	if cfg.CRT.HorizontalBlurTap != 0 {
		base.CRT.HorizontalBlurTap = cfg.CRT.HorizontalBlurTap
	}

	base = base.normalized()
	if err := base.validate(); err != nil {
		return nil, err
	}

	if renderer == nil {
		var err error
		renderer, err = newRenderer(base)
		if err != nil {
			return nil, err
		}
	}
	if err := renderer.Init(base); err != nil {
		return nil, err
	}

	cyclesPerFrame := base.ClockHz / uint64(base.CRT.RefreshHz)
	if cyclesPerFrame == 0 {
		cyclesPerFrame = 1
	}

	crtc := &AppleIIeCRTC{
		Device: &Device{
			name:           name,
			cfg:            base,
			renderer:       renderer,
			framebuffer:    NewFramebuffer(base.CRT.Width, base.CRT.Height),
			cyclesPerFrame: cyclesPerFrame,
		},
		memory:       normalizedAppleIIeMemory(options.Memory),
		characterROM: normalizedCharacterROM(options.CharacterROM),
		textColor:    appleIIeMonochromeColor,
		pixelSize:    1,
	}

	if options.ColorDisplay {
		crtc.textColor = appleIIeWhiteColor
	}

	crtc.initRegisters()
	crtc.SetTextMode()
	if err := crtc.Reset(context.Background(), nil); err != nil {
		return nil, err
	}

	return crtc, nil
}

func (c *AppleIIeCRTC) Reset(_ context.Context, bus *emulator.Bus) error {
	c.bus = bus
	c.frameSequence = 0
	c.nextPresentTick = c.cyclesPerFrame
	c.BeamX = 0
	c.BeamY = 0
	c.RasterLine = 0
	c.RasterCount = 0
	c.CCLK = 0
	c.VBL = 0x00
	c.blinkOn = false
	c.blinkFrames = 0
	c.framebuffer.Fill(appleIIeBlackColor)
	c.redrawFrame()
	return nil
}

func (c *AppleIIeCRTC) Tick(_ context.Context, tick emulator.Tick, bus *emulator.Bus) error {
	if bus != nil && c.bus == nil {
		c.bus = bus
	}
	c.advanceBeam()

	if tick.Cycle < c.nextPresentTick {
		return nil
	}

	c.redrawFrame()
	c.frameSequence++
	c.blinkFrames++
	if c.blinkFrames%12 == 0 {
		c.blinkOn = !c.blinkOn
	}
	if err := c.renderer.Present(c.framebuffer.Snapshot(c.frameSequence)); err != nil {
		return err
	}
	c.nextPresentTick += c.cyclesPerFrame
	return nil
}

func (c *AppleIIeCRTC) Read(addr uint16) (byte, error) {
	switch addr {
	case appleIIeSwitchVBlank:
		return c.VBL, nil
	case appleIIeSwitchText:
		if c.mode.Text {
			return 0x80, nil
		}
		return 0x00, nil
	case appleIIeSwitchGraphics:
		if !c.mode.Text {
			return 0x80, nil
		}
		return 0x00, nil
	case appleIIeSwitchHiRes:
		if c.mode.HiRes {
			return 0x80, nil
		}
		return 0x00, nil
	case appleIIeSwitchLoRes:
		if !c.mode.HiRes {
			return 0x80, nil
		}
		return 0x00, nil
	case appleIIeSwitchPage2:
		if c.mode.Page2 {
			return 0x80, nil
		}
		return 0x00, nil
	case appleIIeSwitchPage1:
		if !c.mode.Page2 {
			return 0x80, nil
		}
		return 0x00, nil
	case appleIIeSwitchMixed:
		if c.mode.Mixed {
			return 0x80, nil
		}
		return 0x00, nil
	case appleIIeSwitch80ColOn:
		if c.mode.Columns80 {
			return 0x80, nil
		}
		return 0x00, nil
	case appleIIeSwitch80ColOff:
		if !c.mode.Columns80 {
			return 0x80, nil
		}
		return 0x00, nil
	default:
		return 0x00, nil
	}
}

func (c *AppleIIeCRTC) Write(addr uint16, _ byte) error {
	switch addr {
	case appleIIeSwitchGraphics:
		c.SetGraphicsMode()
	case appleIIeSwitchText:
		c.SetTextMode()
	case appleIIeSwitchFull:
		c.SetFullMode()
	case appleIIeSwitchMixed:
		c.SetMixedMode()
	case appleIIeSwitchPage1:
		c.SetPage1()
	case appleIIeSwitchPage2:
		c.SetPage2()
	case appleIIeSwitchLoRes:
		c.SetLoResMode()
	case appleIIeSwitchHiRes:
		c.SetHiResMode()
	case appleIIeSwitch80ColOff:
		c.Set40Cols()
	case appleIIeSwitch80ColOn:
		c.Set80Cols()
	}
	return nil
}

func (c *AppleIIeCRTC) Mode() AppleIIeDisplayMode {
	return c.mode
}

func (c *AppleIIeCRTC) ModeString() string {
	parts := make([]string, 0, 5)
	if c.mode.Text {
		parts = append(parts, "TEXT")
	} else if c.mode.HiRes {
		parts = append(parts, "HIRES")
	} else {
		parts = append(parts, "LORES")
	}
	if c.mode.Mixed {
		parts = append(parts, "MIXED")
	}
	if c.mode.Columns80 {
		parts = append(parts, "80COL")
	} else {
		parts = append(parts, "40COL")
	}
	if c.mode.Page2 {
		parts = append(parts, "PAGE2")
	} else {
		parts = append(parts, "PAGE1")
	}
	if c.mode.DoubleWidth {
		parts = append(parts, "DBLWIDTH")
	}
	return strings.Join(parts, " ")
}

func (c *AppleIIeCRTC) SetTextMode() {
	c.mode.Text = true
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) SetGraphicsMode() {
	c.mode.Text = false
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) Set40Cols() {
	c.mode.Columns80 = false
	c.disableDoubleWidth()
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) Set80Cols() {
	c.mode.Columns80 = true
	c.enableDoubleWidth()
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) SetMixedMode() {
	c.mode.Mixed = true
}

func (c *AppleIIeCRTC) SetFullMode() {
	c.mode.Mixed = false
}

func (c *AppleIIeCRTC) SetLoResMode() {
	c.mode.HiRes = false
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) SetHiResMode() {
	c.mode.HiRes = true
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) SetPage1() {
	c.mode.Page2 = false
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) SetPage2() {
	c.mode.Page2 = true
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) SetDoubleWidth() {
	c.mode.DoubleWidth = true
	c.enableDoubleWidth()
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) SetNormalWidth() {
	c.mode.DoubleWidth = false
	c.disableDoubleWidth()
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) ToggleMonitorColor() {
	if c.textColor == appleIIeWhiteColor {
		c.textColor = appleIIeMonochromeColor
		return
	}
	c.textColor = appleIIeWhiteColor
}

func (c *AppleIIeCRTC) initRegisters() {
	c.Reg[appleIIeRegHorizontalTotal] = 63
	c.Reg[appleIIeRegHorizontalDisplayed] = 40
	c.Reg[appleIIeRegHorizontalSyncPosition] = 50
	c.Reg[appleIIeRegSyncWidths] = 0b10001000
	c.Reg[appleIIeRegVerticalTotal] = 32
	c.Reg[appleIIeRegVerticalAdjust] = 16
	c.Reg[appleIIeRegVerticalDisplayed] = 24
	c.Reg[appleIIeRegVerticalSyncPosition] = 29
	c.Reg[appleIIeRegMaxRasterAddress] = 8
	c.Reg[appleIIeRegStartAddressHigh] = 0
	c.Reg[appleIIeRegStartAddressLow] = 0
	c.pixelSize = 1
	c.screenWidth = int(c.Reg[appleIIeRegHorizontalDisplayed]) * 7
	c.screenHeight = int(c.Reg[appleIIeRegVerticalDisplayed]) * 8
	c.updateDisplayMode()
}

func (c *AppleIIeCRTC) enableDoubleWidth() {
	c.Reg[appleIIeRegHorizontalTotal] = 126
	c.Reg[appleIIeRegHorizontalDisplayed] = 80
	c.pixelSize = 2
	c.screenWidth = int(c.Reg[appleIIeRegHorizontalDisplayed]) * 7
}

func (c *AppleIIeCRTC) disableDoubleWidth() {
	c.Reg[appleIIeRegHorizontalTotal] = 63
	c.Reg[appleIIeRegHorizontalDisplayed] = 40
	c.pixelSize = 1
	c.screenWidth = int(c.Reg[appleIIeRegHorizontalDisplayed]) * 7
}

func (c *AppleIIeCRTC) updateDisplayMode() {
	page := 0
	if c.mode.Page2 {
		page = 1
	}

	if c.mode.Text {
		c.videoMainMem = c.memory.MainText[page]
		if c.mode.Columns80 {
			c.videoAuxMem = c.memory.AuxText[page]
			c.renderMode = appleIIeRenderText80
		} else {
			c.videoAuxMem = nil
			c.renderMode = appleIIeRenderText40
		}
		return
	}

	if c.mode.HiRes {
		c.videoMainMem = c.memory.MainHiRes[page]
		if c.mode.Columns80 {
			c.videoAuxMem = c.memory.AuxHiRes[page]
			c.renderMode = appleIIeRenderDoubleHiRes
		} else {
			c.videoAuxMem = nil
			c.renderMode = appleIIeRenderHiRes
		}
		return
	}

	c.videoMainMem = c.memory.MainText[page]
	if c.mode.Columns80 && c.mode.DoubleWidth {
		c.videoAuxMem = c.memory.AuxText[page]
		c.renderMode = appleIIeRenderLoRes80
		return
	}
	c.videoAuxMem = nil
	c.renderMode = appleIIeRenderLoRes
}

func (c *AppleIIeCRTC) advanceBeam() {
	if c.VBL == 0 {
		c.VBL = 0x80
	}

	c.BeamX = int(c.CCLK) * 7
	c.CCLK += c.pixelSize
	if c.CCLK < c.Reg[appleIIeRegHorizontalTotal] {
		return
	}

	c.CCLK = 0
	c.BeamY++
	if c.BeamY >= c.screenHeight {
		c.BeamY = 0
		c.RasterCount = 0
		c.RasterLine = 0
		c.VBL = 0x00
		return
	}

	c.RasterCount++
	if c.RasterCount == int(c.Reg[appleIIeRegMaxRasterAddress]) {
		c.RasterLine++
		c.RasterCount = 0
	}
	c.VBL = 0x80
}

func (c *AppleIIeCRTC) redrawFrame() {
	c.framebuffer.Fill(appleIIeBlackColor)
	switch c.renderMode {
	case appleIIeRenderText40:
		c.renderText(false)
	case appleIIeRenderText80:
		c.renderText(true)
	case appleIIeRenderLoRes:
		c.renderLoRes(false)
	case appleIIeRenderLoRes80:
		c.renderLoRes(true)
	case appleIIeRenderHiRes:
		c.renderHiRes(false)
	case appleIIeRenderDoubleHiRes:
		c.renderHiRes(true)
	}
}

func (c *AppleIIeCRTC) renderText(columns80 bool) {
	cellWidth := c.cfg.CRT.Width / 40
	if columns80 {
		cellWidth = c.cfg.CRT.Width / 80
	}
	cellHeight := c.cfg.CRT.Height / 24
	for row := 0; row < 24; row++ {
		if !c.mode.Text && c.mode.Mixed && row < 20 {
			continue
		}
		columns := 40
		if columns80 {
			columns = 80
		}
		for col := 0; col < columns; col++ {
			value := c.textByte(row, col, columns80)
			c.drawCharacter(col*cellWidth, row*cellHeight, cellWidth, cellHeight, value)
		}
	}
}

func (c *AppleIIeCRTC) renderLoRes(columns80 bool) {
	columns := 40
	if columns80 {
		columns = 80
	}
	cellWidth := c.cfg.CRT.Width / columns
	blockHeight := c.cfg.CRT.Height / 48
	for row := 0; row < 24; row++ {
		if c.mode.Mixed && row >= 20 {
			continue
		}
		for col := 0; col < columns; col++ {
			value := c.textByte(row, col, columns80)
			topColor := appleIIePalette[value&0x0F]
			bottomColor := appleIIePalette[(value>>4)&0x0F]
			c.fillRect(col*cellWidth, row*2*blockHeight, cellWidth, blockHeight, topColor)
			c.fillRect(col*cellWidth, (row*2+1)*blockHeight, cellWidth, blockHeight, bottomColor)
		}
	}
	if c.mode.Mixed {
		c.renderText(columns80)
	}
}

func (c *AppleIIeCRTC) renderHiRes(doubleHiRes bool) {
	scaleX := c.cfg.CRT.Width / 280
	if scaleX == 0 {
		scaleX = 1
	}
	scaleY := c.cfg.CRT.Height / 192
	if scaleY == 0 {
		scaleY = 1
	}
	for y := 0; y < 192; y++ {
		if c.mode.Mixed && y >= 160 {
			continue
		}
		for col := 0; col < 40; col++ {
			mainByte := c.readHiResByte(y, col, false)
			for bit := 0; bit < 7; bit++ {
				pixelOn := mainByte&(1<<bit) != 0
				x := col*7 + bit
				color := appleIIeBlackColor
				if pixelOn {
					color = c.textColor
				}
				c.fillRect(x*scaleX, y*scaleY, scaleX, scaleY, color)
			}

			if !doubleHiRes || c.videoAuxMem == nil {
				continue
			}
			auxByte := c.readHiResByte(y, col, true)
			for bit := 0; bit < 7; bit++ {
				if auxByte&(1<<bit) == 0 {
					continue
				}
				x := col*7 + bit
				c.fillRect(x*scaleX, y*scaleY, scaleX/2+1, scaleY, appleIIePalette[(bit+col)%len(appleIIePalette)])
			}
		}
	}
	if c.mode.Mixed {
		c.renderText(c.mode.Columns80)
	}
}

func (c *AppleIIeCRTC) textByte(row, col int, columns80 bool) byte {
	index := col
	mem := c.videoMainMem
	aux := false
	if columns80 {
		index = col / 2
		if col%2 == 0 {
			mem = c.videoAuxMem
			aux = true
		}
	}
	offset := appleIIeTextOffset(row, index)
	if value, ok := c.readActiveVideoByte(offset, aux); ok {
		return value
	}
	if offset >= len(mem) {
		return 0xA0
	}
	return mem[offset]
}

func (c *AppleIIeCRTC) readHiResByte(row, col int, aux bool) byte {
	offset := appleIIeHiResOffset(row, col)
	if value, ok := c.readActiveVideoByte(offset, aux); ok {
		return value
	}

	bank := c.videoMainMem
	if aux {
		bank = c.videoAuxMem
	}
	if offset >= len(bank) {
		return 0
	}
	return bank[offset]
}

func (c *AppleIIeCRTC) readActiveVideoByte(offset int, aux bool) (byte, bool) {
	if c.bus == nil {
		return 0, false
	}

	addr, ok := c.activeVideoAddress(offset, aux)
	if !ok {
		return 0, false
	}

	value, err := c.bus.Read(addr)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (c *AppleIIeCRTC) activeVideoAddress(offset int, aux bool) (uint16, bool) {
	if offset < 0 || aux {
		return 0, false
	}

	var base uint16
	switch c.renderMode {
	case appleIIeRenderText40, appleIIeRenderText80, appleIIeRenderLoRes, appleIIeRenderLoRes80:
		if offset >= appleIIeTextPageSize {
			return 0, false
		}
		base = appleIIeTextBaseAddress(c.mode.Page2)
	case appleIIeRenderHiRes, appleIIeRenderDoubleHiRes:
		if offset >= appleIIeHiResPageSize {
			return 0, false
		}
		base = appleIIeHiResBaseAddress(c.mode.Page2)
	default:
		return 0, false
	}

	return base + uint16(offset), true
}

func (c *AppleIIeCRTC) drawCharacter(x, y, width, height int, value byte) {
	glyphWidth := 7
	glyphHeight := 8
	scaleX := width / glyphWidth
	if scaleX == 0 {
		scaleX = 1
	}
	scaleY := height / glyphHeight
	if scaleY == 0 {
		scaleY = 1
	}

	glyphIndex := int(value & 0x7F)
	inverse := value < 0x40
	flashing := value >= 0x40 && value < 0x80
	if flashing && c.blinkOn {
		inverse = !inverse
	}

	for row := 0; row < glyphHeight; row++ {
		glyphRow := c.characterROM[glyphIndex*glyphHeight+row]
		for col := 0; col < glyphWidth; col++ {
			bitOn := glyphRow&(1<<(glyphWidth-col-1)) != 0
			if inverse {
				bitOn = !bitOn
			}
			color := appleIIeBlackColor
			if bitOn {
				color = c.textColor
			}
			c.fillRect(x+col*scaleX, y+row*scaleY, scaleX, scaleY, color)
		}
	}
}

func (c *AppleIIeCRTC) fillRect(x, y, width, height int, color uint32) {
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			c.framebuffer.SetPixel(x+px, y+py, color)
		}
	}
}

func appleIIeTextOffset(row, col int) int {
	return ((row & 0x07) << 7) + ((row >> 3) * 0x28) + col
}

func appleIIeHiResOffset(row, col int) int {
	return ((row & 0x07) << 10) + (((row >> 3) & 0x07) << 7) + ((row >> 6) * 0x28) + col
}

func normalizedAppleIIeMemory(memory AppleIIeMemory) AppleIIeMemory {
	for page := 0; page < 2; page++ {
		memory.MainText[page] = normalizedBank(memory.MainText[page], appleIIeTextPageSize)
		memory.AuxText[page] = normalizedBank(memory.AuxText[page], appleIIeTextPageSize)
		memory.MainHiRes[page] = normalizedBank(memory.MainHiRes[page], appleIIeHiResPageSize)
		memory.AuxHiRes[page] = normalizedBank(memory.AuxHiRes[page], appleIIeHiResPageSize)
	}
	return memory
}

func normalizedBank(bank []byte, size int) []byte {
	if len(bank) >= size {
		return bank
	}
	normalized := make([]byte, size)
	copy(normalized, bank)
	return normalized
}

func normalizedCharacterROM(charROM []byte) []byte {
	if len(charROM) >= appleIIeCharROMSize {
		return charROM
	}
	normalized := make([]byte, appleIIeCharROMSize)
	copy(normalized, charROM)
	return normalized
}

func appleIIeTextBaseAddress(page2 bool) uint16 {
	if page2 {
		return 0x0800
	}
	return 0x0400
}

func appleIIeHiResBaseAddress(page2 bool) uint16 {
	if page2 {
		return 0x4000
	}
	return 0x2000
}

const (
	appleIIeBlackColor      uint32 = 0xFF000000
	appleIIeWhiteColor      uint32 = 0xFFF2F2F2
	appleIIeMonochromeColor uint32 = 0xFF7CFF7C
)

var appleIIePalette = [16]uint32{
	0xFF000000,
	0xFF9D0966,
	0xFF2E2CD8,
	0xFFB03BFF,
	0xFF007A27,
	0xFF808080,
	0xFF00AEEF,
	0xFF83D0FF,
	0xFF5E3C00,
	0xFFFF5D12,
	0xFFAAAAAA,
	0xFFFF89D1,
	0xFF31D843,
	0xFFFFFF65,
	0xFF5EF1A4,
	0xFFFFFFFF,
}
