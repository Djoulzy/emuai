package cpu

import (
	"context"
	"fmt"

	"github.com/Djoulzy/emuAI/internal/emulator"
)

const (
	flagC byte = 1 << 0 // Carry
	flagZ byte = 1 << 1 // Zero
	flagI byte = 1 << 2 // Interrupt disable
	flagD byte = 1 << 3 // Decimal mode
	flagB byte = 1 << 4 // Break
	flagV byte = 1 << 6 // Overflow
	flagN byte = 1 << 7 // Negative
)

type microOp func(bus *emulator.Bus) error

type decodedInstruction struct {
	opcode byte
	name   string
	step   int
	steps  []microOp
}

// CPU6502 is a cycle-sliced skeleton for a MOS 6502-like core.
// Each motherboard tick executes exactly one CPU cycle:
// opcode fetch, then one micro-op per subsequent cycle.
type CPU6502 struct {
	name        string
	resetVector uint16

	A  byte
	X  byte
	Y  byte
	SP byte
	P  byte
	PC uint16

	halted  bool
	current *decodedInstruction
	tmp8    byte
	tmpAddr uint16
}

func NewCPU6502(name string, resetVector uint16) *CPU6502 {
	return &CPU6502{name: name, resetVector: resetVector}
}

// Kept for backward compatibility with initial scaffold.
func NewSimpleCPU(name string, resetVector uint16) *CPU6502 {
	return NewCPU6502(name, resetVector)
}

func (c *CPU6502) Name() string {
	return c.name
}

func (c *CPU6502) Reset(_ context.Context) error {
	c.A = 0
	c.X = 0
	c.Y = 0
	c.SP = 0xFD
	c.P = flagI
	c.PC = c.resetVector
	c.halted = false
	c.current = nil
	c.tmp8 = 0
	c.tmpAddr = 0
	return nil
}

func (c *CPU6502) Tick(_ context.Context, _ emulator.Tick, bus *emulator.Bus) error {
	if c.halted {
		return nil
	}

	// No current instruction means this cycle is the opcode fetch cycle.
	if c.current == nil {
		opcode, err := c.fetchByte(bus)
		if err != nil {
			return err
		}

		instr, err := c.decode(opcode)
		if err != nil {
			return err
		}
		c.current = instr
		return nil
	}

	if c.current.step >= len(c.current.steps) {
		c.current = nil
		return nil
	}

	op := c.current.steps[c.current.step]
	if err := op(bus); err != nil {
		return err
	}

	c.current.step++
	if c.current.step >= len(c.current.steps) {
		c.current = nil
	}

	return nil
}

func (c *CPU6502) Close() error {
	return nil
}

func (c *CPU6502) fetchByte(bus *emulator.Bus) (byte, error) {
	v, err := bus.Read(c.PC)
	if err != nil {
		return 0, err
	}
	c.PC++
	return v, nil
}

func (c *CPU6502) updateZN(v byte) {
	c.setFlag(flagZ, v == 0)
	c.setFlag(flagN, (v&0x80) != 0)
}

func (c *CPU6502) setFlag(flag byte, enabled bool) {
	if enabled {
		c.P |= flag
		return
	}
	c.P &^= flag
}

func (c *CPU6502) getFlag(flag byte) byte {
	if c.P&flag != 0 {
		return 1
	}
	return 0
}

func (c *CPU6502) adc(v byte) {
	a := c.A
	sum := uint16(a) + uint16(v) + uint16(c.getFlag(flagC))
	result := byte(sum)

	c.setFlag(flagC, sum > 0xFF)
	c.setFlag(flagV, ((^(a ^ v))&(a^result)&0x80) != 0)
	c.A = result
	c.updateZN(c.A)
}

func (c *CPU6502) Halted() bool {
	return c.halted
}

func (c *CPU6502) decode(opcode byte) (*decodedInstruction, error) {
	switch opcode {
	case 0xEA: // NOP (2 cycles total)
		return &decodedInstruction{opcode: opcode, name: "NOP", steps: []microOp{
			func(_ *emulator.Bus) error { return nil },
		}}, nil

	case 0xA9: // LDA #imm (2 cycles total)
		return &decodedInstruction{opcode: opcode, name: "LDA #imm", steps: []microOp{
			func(bus *emulator.Bus) error {
				v, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.A = v
				c.updateZN(c.A)
				return nil
			},
		}}, nil

	case 0xA5: // LDA zp (3 cycles total)
		return &decodedInstruction{opcode: opcode, name: "LDA zp", steps: []microOp{
			func(bus *emulator.Bus) error {
				v, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.tmpAddr = uint16(v)
				return nil
			},
			func(bus *emulator.Bus) error {
				v, err := bus.Read(c.tmpAddr)
				if err != nil {
					return err
				}
				c.A = v
				c.updateZN(c.A)
				return nil
			},
		}}, nil

	case 0xAD: // LDA abs (4 cycles total)
		return &decodedInstruction{opcode: opcode, name: "LDA abs", steps: []microOp{
			func(bus *emulator.Bus) error {
				lo, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.tmp8 = lo
				return nil
			},
			func(bus *emulator.Bus) error {
				hi, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.tmpAddr = uint16(hi)<<8 | uint16(c.tmp8)
				return nil
			},
			func(bus *emulator.Bus) error {
				v, err := bus.Read(c.tmpAddr)
				if err != nil {
					return err
				}
				c.A = v
				c.updateZN(c.A)
				return nil
			},
		}}, nil

	case 0x85: // STA zp (3 cycles total)
		return &decodedInstruction{opcode: opcode, name: "STA zp", steps: []microOp{
			func(bus *emulator.Bus) error {
				v, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.tmpAddr = uint16(v)
				return nil
			},
			func(bus *emulator.Bus) error {
				return bus.Write(c.tmpAddr, c.A)
			},
		}}, nil

	case 0x8D: // STA abs (4 cycles total)
		return &decodedInstruction{opcode: opcode, name: "STA abs", steps: []microOp{
			func(bus *emulator.Bus) error {
				lo, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.tmp8 = lo
				return nil
			},
			func(bus *emulator.Bus) error {
				hi, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.tmpAddr = uint16(hi)<<8 | uint16(c.tmp8)
				return nil
			},
			func(bus *emulator.Bus) error {
				return bus.Write(c.tmpAddr, c.A)
			},
		}}, nil

	case 0xAA: // TAX (2 cycles total)
		return &decodedInstruction{opcode: opcode, name: "TAX", steps: []microOp{
			func(_ *emulator.Bus) error {
				c.X = c.A
				c.updateZN(c.X)
				return nil
			},
		}}, nil

	case 0xE8: // INX (2 cycles total)
		return &decodedInstruction{opcode: opcode, name: "INX", steps: []microOp{
			func(_ *emulator.Bus) error {
				c.X++
				c.updateZN(c.X)
				return nil
			},
		}}, nil

	case 0x4C: // JMP abs (3 cycles total)
		return &decodedInstruction{opcode: opcode, name: "JMP abs", steps: []microOp{
			func(bus *emulator.Bus) error {
				lo, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.tmp8 = lo
				return nil
			},
			func(bus *emulator.Bus) error {
				hi, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.PC = uint16(hi)<<8 | uint16(c.tmp8)
				return nil
			},
		}}, nil

	case 0x69: // ADC #imm (2 cycles total)
		return &decodedInstruction{opcode: opcode, name: "ADC #imm", steps: []microOp{
			func(bus *emulator.Bus) error {
				v, err := c.fetchByte(bus)
				if err != nil {
					return err
				}
				c.adc(v)
				return nil
			},
		}}, nil

	case 0x00: // BRK (simplified 7 cycles total)
		return &decodedInstruction{opcode: opcode, name: "BRK", steps: []microOp{
			func(_ *emulator.Bus) error { return nil },
			func(_ *emulator.Bus) error { return nil },
			func(_ *emulator.Bus) error { return nil },
			func(_ *emulator.Bus) error { return nil },
			func(_ *emulator.Bus) error { return nil },
			func(_ *emulator.Bus) error {
				c.setFlag(flagB, true)
				c.halted = true
				return nil
			},
		}}, nil

	default:
		return nil, fmt.Errorf("cpu6502: unsupported opcode 0x%02X at PC=0x%04X", opcode, c.PC-1)
	}
}
