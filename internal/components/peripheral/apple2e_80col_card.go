package peripheral

import "github.com/Djoulzy/emuai/internal/emulator"

const (
	appleIIe80ColSlot3BaseAddress    uint16 = 0xC0B0
	appleIIe80ColSlot3ReadHandshake  uint16 = 0xC0B2
	appleIIe80ColSlot3BeginTransfer  uint16 = 0xC0B4
	appleIIe80ColSlot3FinishTransfer uint16 = 0xC0B6
)

type appleIIe80ColumnAuxMemory interface {
	ArmAuxSlotAccess()
	ClearAuxSlotAccess()
}

type AppleIIe80ColumnCard struct {
	name        string
	memory      appleIIe80ColumnAuxMemory
	handshakeHi bool
}

func NewAppleIIe80ColumnCard(name string, memory appleIIe80ColumnAuxMemory) *AppleIIe80ColumnCard {
	if name == "" {
		name = "apple2e-80col-slot3"
	}
	return &AppleIIe80ColumnCard{name: name, memory: memory}
}

func (c *AppleIIe80ColumnCard) Name() string {
	return c.name
}

func (c *AppleIIe80ColumnCard) Read(addr uint16) (byte, error) {
	switch addr {
	case appleIIe80ColSlot3BeginTransfer:
		c.handshakeHi = false
		if c.memory != nil {
			c.memory.ClearAuxSlotAccess()
		}
		return 0x00, nil
	case appleIIe80ColSlot3ReadHandshake:
		if c.handshakeHi {
			c.handshakeHi = false
			if c.memory != nil {
				c.memory.ArmAuxSlotAccess()
			}
			return 0x80, nil
		}
		c.handshakeHi = true
		return 0x00, nil
	case appleIIe80ColSlot3FinishTransfer:
		if c.memory != nil {
			c.memory.ClearAuxSlotAccess()
		}
		return 0x00, nil
	default:
		return 0x00, nil
	}
}

func (c *AppleIIe80ColumnCard) Write(addr uint16, _ byte) error {
	_, err := c.Read(addr)
	return err
}

var _ emulator.AddressableDevice = (*AppleIIe80ColumnCard)(nil)
