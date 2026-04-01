package memory

import (
	"context"
	"fmt"
	"os"

	"github.com/Djoulzy/emuai/internal/emulator"
)

const (
	appleIIeMMUSwitch80StoreOff    uint16 = 0xC000
	appleIIeMMUSwitch80StoreOn     uint16 = 0xC001
	appleIIeMMUSwitchRAMReadOff    uint16 = 0xC002
	appleIIeMMUSwitchRAMReadOn     uint16 = 0xC003
	appleIIeMMUSwitchRAMWriteOff   uint16 = 0xC004
	appleIIeMMUSwitchRAMWriteOn    uint16 = 0xC005
	appleIIeMMUSwitchIntCxROMOff   uint16 = 0xC006
	appleIIeMMUSwitchIntCxROMOn    uint16 = 0xC007
	appleIIeMMUSwitchAltZPOff      uint16 = 0xC008
	appleIIeMMUSwitchAltZPOn       uint16 = 0xC009
	appleIIeMMUSwitchSlotC3ROMOff  uint16 = 0xC00A
	appleIIeMMUSwitchSlotC3ROMOn   uint16 = 0xC00B
	appleIIeMMUSwitchReadRAMRead   uint16 = 0xC013
	appleIIeMMUSwitchReadRAMWrt    uint16 = 0xC014
	appleIIeMMUSwitchReadIntCxROM  uint16 = 0xC015
	appleIIeMMUSwitchReadAltZP     uint16 = 0xC016
	appleIIeMMUSwitchReadSlotC3ROM uint16 = 0xC017
	appleIIeMMUSwitchRead80Store   uint16 = 0xC018
	appleIIeMMUSwitchPage1         uint16 = 0xC054
	appleIIeMMUSwitchPage2         uint16 = 0xC055
	appleIIeMMUSwitchLoRes         uint16 = 0xC056
	appleIIeMMUSwitchHiRes         uint16 = 0xC057

	appleIIeAddressSpaceSize = 0x10000
	appleIIeSlotROMMaxSize   = 0x0900
)

type AppleIIeMemoryMode struct {
	RAMReadAux    bool
	RAMWriteAux   bool
	AltZeroPage   bool
	InternalCxROM bool
	SlotC3ROM     bool
	Store80       bool
	Page2         bool
	HiRes         bool
}

type AppleIIeMMU struct {
	name string

	main       []byte
	aux        []byte
	internal   []byte
	romLoaded  []bool
	slotROMs   [8][]byte
	activeSlot int

	mode AppleIIeMemoryMode
}

func NewAppleIIeMMU(name string) (*AppleIIeMMU, error) {
	if name == "" {
		return nil, fmt.Errorf("apple2e-mmu: device name is required")
	}

	return &AppleIIeMMU{
		name:      name,
		main:      make([]byte, appleIIeAddressSpaceSize),
		aux:       make([]byte, appleIIeAddressSpaceSize),
		internal:  make([]byte, appleIIeAddressSpaceSize),
		romLoaded: make([]bool, appleIIeAddressSpaceSize),
	}, nil
}

func (m *AppleIIeMMU) Name() string {
	return m.name
}

func (m *AppleIIeMMU) Reset(_ context.Context, _ *emulator.Bus) error {
	for idx := range m.main {
		m.main[idx] = 0
		m.aux[idx] = 0
	}
	m.mode = AppleIIeMemoryMode{}
	m.activeSlot = 0
	return nil
}

func (m *AppleIIeMMU) Tick(_ context.Context, _ emulator.Tick, _ *emulator.Bus) error {
	return nil
}

func (m *AppleIIeMMU) Close() error {
	return nil
}

func (m *AppleIIeMMU) Mode() AppleIIeMemoryMode {
	return m.mode
}

func (m *AppleIIeMMU) Read(addr uint16) (byte, error) {
	switch addr {
	case appleIIeMMUSwitchReadRAMRead:
		return softSwitchValue(m.mode.RAMReadAux), nil
	case appleIIeMMUSwitchReadRAMWrt:
		return softSwitchValue(m.mode.RAMWriteAux), nil
	case appleIIeMMUSwitchReadIntCxROM:
		return softSwitchValue(m.mode.InternalCxROM), nil
	case appleIIeMMUSwitchReadAltZP:
		return softSwitchValue(m.mode.AltZeroPage), nil
	case appleIIeMMUSwitchReadSlotC3ROM:
		return softSwitchValue(m.mode.SlotC3ROM), nil
	case appleIIeMMUSwitchRead80Store:
		return softSwitchValue(m.mode.Store80), nil
	}

	if value, ok := m.readInternalROM(addr); ok {
		return value, nil
	}

	if m.useAuxRead(addr) {
		return m.aux[addr], nil
	}
	return m.main[addr], nil
}

func (m *AppleIIeMMU) ReadMain(addr uint16) (byte, error) {
	return m.main[addr], nil
}

func (m *AppleIIeMMU) ReadAux(addr uint16) (byte, error) {
	return m.aux[addr], nil
}

func (m *AppleIIeMMU) Write(addr uint16, value byte) error {
	switch addr {
	case appleIIeMMUSwitch80StoreOff:
		m.mode.Store80 = false
		return nil
	case appleIIeMMUSwitch80StoreOn:
		m.mode.Store80 = true
		return nil
	case appleIIeMMUSwitchRAMReadOff:
		m.mode.RAMReadAux = false
		return nil
	case appleIIeMMUSwitchRAMReadOn:
		m.mode.RAMReadAux = true
		return nil
	case appleIIeMMUSwitchRAMWriteOff:
		m.mode.RAMWriteAux = false
		return nil
	case appleIIeMMUSwitchRAMWriteOn:
		m.mode.RAMWriteAux = true
		return nil
	case appleIIeMMUSwitchIntCxROMOff:
		m.mode.InternalCxROM = false
		return nil
	case appleIIeMMUSwitchIntCxROMOn:
		m.mode.InternalCxROM = true
		return nil
	case appleIIeMMUSwitchAltZPOff:
		m.mode.AltZeroPage = false
		return nil
	case appleIIeMMUSwitchAltZPOn:
		m.mode.AltZeroPage = true
		return nil
	case appleIIeMMUSwitchSlotC3ROMOff:
		m.mode.SlotC3ROM = false
		return nil
	case appleIIeMMUSwitchSlotC3ROMOn:
		m.mode.SlotC3ROM = true
		return nil
	case appleIIeMMUSwitchPage1:
		m.mode.Page2 = false
		return nil
	case appleIIeMMUSwitchPage2:
		m.mode.Page2 = true
		return nil
	case appleIIeMMUSwitchLoRes:
		m.mode.HiRes = false
		return nil
	case appleIIeMMUSwitchHiRes:
		m.mode.HiRes = true
		return nil
	}

	if m.useAuxWrite(addr) {
		m.aux[addr] = value
		return nil
	}
	m.main[addr] = value
	return nil
}

func (m *AppleIIeMMU) Load(addr uint16, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	start := int(addr)
	end := start + len(data)
	if end > appleIIeAddressSpaceSize {
		return fmt.Errorf(
			"apple2e-mmu: binary load out of range start=0x%04X len=%d end=0x%04X limit=0x%04X",
			addr,
			len(data),
			addr+uint16(len(data))-1,
			0xFFFF,
		)
	}

	for idx, value := range data {
		target := start + idx
		if target >= 0xC000 {
			m.internal[target] = value
			m.romLoaded[target] = true
			continue
		}
		m.main[target] = value
	}

	return nil
}

func (m *AppleIIeMMU) LoadFile(path string, addr uint16) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("apple2e-mmu: read binary %q: %w", path, err)
	}

	if err := m.Load(addr, data); err != nil {
		return fmt.Errorf("apple2e-mmu: load binary %q: %w", path, err)
	}

	return nil
}

func (m *AppleIIeMMU) LoadAux(addr uint16, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	start := int(addr)
	end := start + len(data)
	if end > appleIIeAddressSpaceSize {
		return fmt.Errorf(
			"apple2e-mmu: aux load out of range start=0x%04X len=%d end=0x%04X limit=0x%04X",
			addr,
			len(data),
			addr+uint16(len(data))-1,
			0xFFFF,
		)
	}

	copy(m.aux[start:end], data)
	return nil
}

func (m *AppleIIeMMU) LoadSlotROM(slot int, data []byte) error {
	if slot < 1 || slot > 7 {
		return fmt.Errorf("apple2e-mmu: invalid slot %d", slot)
	}
	if len(data) == 0 {
		return fmt.Errorf("apple2e-mmu: slot %d ROM is empty", slot)
	}
	if len(data) > appleIIeSlotROMMaxSize {
		return fmt.Errorf("apple2e-mmu: slot %d ROM too large: got %d bytes, max %d", slot, len(data), appleIIeSlotROMMaxSize)
	}

	rom := make([]byte, len(data))
	copy(rom, data)
	m.slotROMs[slot] = rom
	return nil
}

func (m *AppleIIeMMU) LoadSlotROMFile(slot int, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("apple2e-mmu: read slot %d ROM %q: %w", slot, path, err)
	}

	if err := m.LoadSlotROM(slot, data); err != nil {
		return fmt.Errorf("apple2e-mmu: load slot %d ROM %q: %w", slot, path, err)
	}

	return nil
}

func (m *AppleIIeMMU) readInternalROM(addr uint16) (byte, bool) {
	if addr == 0xCFFF {
		m.activeSlot = 0
	}

	if !m.romLoaded[addr] {
		if value, ok := m.readSlotROM(addr); ok {
			return value, true
		}
		return 0, false
	}

	if addr >= 0xD000 {
		return m.internal[addr], true
	}

	if addr >= 0xC300 && addr <= 0xC3FF && (m.mode.InternalCxROM || !m.mode.SlotC3ROM) {
		m.activeSlot = 0
	}

	if value, ok := m.readSlotROM(addr); ok {
		return value, true
	}

	if addr >= 0xC100 {
		if m.mode.InternalCxROM {
			return m.internal[addr], true
		}

		return m.internal[addr], true
	}

	if addr >= 0xC000 {
		return m.internal[addr], true
	}

	return 0, false
}

func (m *AppleIIeMMU) readSlotROM(addr uint16) (byte, bool) {
	if m.mode.InternalCxROM {
		return 0, false
	}

	if addr >= 0xC100 && addr <= 0xC7FF {
		slot := int((addr >> 8) & 0x000F)
		if slot == 3 && !m.mode.SlotC3ROM {
			return 0, false
		}
		rom := m.slotROMs[slot]
		if len(rom) == 0 {
			return 0, false
		}

		m.activeSlot = slot
		offset := int(addr & 0x00FF)
		if offset >= len(rom) {
			return 0, false
		}
		return rom[offset], true
	}

	if addr >= 0xC800 && addr <= 0xCFFF && m.activeSlot >= 1 && m.activeSlot <= 7 {
		rom := m.slotROMs[m.activeSlot]
		if len(rom) <= 0x0100 {
			return 0, false
		}

		offset := 0x0100 + int(addr-0xC800)
		if offset >= len(rom) {
			return 0, false
		}
		return rom[offset], true
	}

	return 0, false
}

func (m *AppleIIeMMU) useAuxRead(addr uint16) bool {
	if addr < 0x0200 {
		return m.mode.AltZeroPage
	}
	if addr >= 0xC000 {
		return false
	}
	if m.mode.Store80 {
		if addr >= 0x0400 && addr <= 0x07FF {
			return m.mode.Page2
		}
		if m.mode.HiRes && addr >= 0x2000 && addr <= 0x3FFF {
			return m.mode.Page2
		}
	}
	return m.mode.RAMReadAux
}

func (m *AppleIIeMMU) useAuxWrite(addr uint16) bool {
	if addr < 0x0200 {
		return m.mode.AltZeroPage
	}
	if addr >= 0xC000 {
		return false
	}
	if m.mode.Store80 {
		if addr >= 0x0400 && addr <= 0x07FF {
			return m.mode.Page2
		}
		if m.mode.HiRes && addr >= 0x2000 && addr <= 0x3FFF {
			return m.mode.Page2
		}
	}
	return m.mode.RAMWriteAux
}

func softSwitchValue(enabled bool) byte {
	if enabled {
		return 0x80
	}
	return 0x00
}
