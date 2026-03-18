package cpu

import (
	"context"
	"fmt"

	"github.com/Djoulzy/emuai/internal/emulator"
)

const (
	flagC byte = 1 << 0 // Carry
	flagZ byte = 1 << 1 // Zero
	flagI byte = 1 << 2 // Interrupt disable
	flagD byte = 1 << 3 // Decimal mode
	flagB byte = 1 << 4 // Break
	flagU byte = 1 << 5 // Unused, always set when pushed/restored
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

	halted         bool
	current        *decodedInstruction
	tmp8           byte
	tmpAddr        uint16
	tmpBase        uint16
	prefetchOpcode byte
	hasPrefetch    bool
}

func NewCPU6502(name string, resetVector uint16) *CPU6502 {
	newCPU := &CPU6502{name: name, resetVector: resetVector}
	newCPU.initLanguage()
	return newCPU
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
	c.P = flagI | flagU
	c.PC = c.resetVector
	c.halted = false
	c.current = nil
	c.tmp8 = 0
	c.tmpAddr = 0
	c.tmpBase = 0
	c.prefetchOpcode = 0
	c.hasPrefetch = false
	return nil
}

func (c *CPU6502) Tick(_ context.Context, _ emulator.Tick, bus *emulator.Bus) error {
	if c.halted {
		return nil
	}

	// No current instruction means this cycle is the opcode fetch cycle.
	if c.current == nil {
		opcode := c.prefetchOpcode
		if c.hasPrefetch {
			c.hasPrefetch = false
		} else {
			var err error
			opcode, err = c.fetchByte(bus)
			if err != nil {
				return err
			}
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

func (c *CPU6502) fetchWord(bus *emulator.Bus) (uint16, error) {
	lo, err := c.fetchByte(bus)
	if err != nil {
		return 0, err
	}
	hi, err := c.fetchByte(bus)
	if err != nil {
		return 0, err
	}
	return uint16(hi)<<8 | uint16(lo), nil
}

func (c *CPU6502) pushByte(bus *emulator.Bus, v byte) error {
	if err := bus.Write(0x0100|uint16(c.SP), v); err != nil {
		return err
	}
	c.SP--
	return nil
}

func (c *CPU6502) pullByte(bus *emulator.Bus) (byte, error) {
	c.SP++
	return bus.Read(0x0100 | uint16(c.SP))
}

func (c *CPU6502) readZeroPageWord(bus *emulator.Bus, addr byte) (uint16, error) {
	lo, err := bus.Read(uint16(addr))
	if err != nil {
		return 0, err
	}
	hi, err := bus.Read(uint16(byte(addr + 1)))
	if err != nil {
		return 0, err
	}
	return uint16(hi)<<8 | uint16(lo), nil
}

func (c *CPU6502) dummyReadPC(bus *emulator.Bus) error {
	_, err := bus.Read(c.PC)
	return err
}

func (c *CPU6502) dummyReadStack(bus *emulator.Bus) error {
	_, err := bus.Read(0x0100 | uint16(c.SP))
	return err
}

func (c *CPU6502) prefetch(bus *emulator.Bus, addr uint16) error {
	opcode, err := bus.Read(addr)
	if err != nil {
		return err
	}
	c.prefetchOpcode = opcode
	c.hasPrefetch = true
	c.PC = addr + 1
	return nil
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

func (c *CPU6502) setStatus(v byte) {
	c.P = v | flagU
}

func (c *CPU6502) status(withBreak bool) byte {
	p := c.P | flagU
	if withBreak {
		return p | flagB
	}
	return p &^ flagB
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

func (c *CPU6502) sbc(v byte) {
	c.adc(^v)
}

func (c *CPU6502) compare(reg, v byte) {
	result := reg - v
	c.setFlag(flagC, reg >= v)
	c.updateZN(result)
}

func (c *CPU6502) bit(v byte) {
	c.setFlag(flagZ, c.A&v == 0)
	c.setFlag(flagV, v&flagV != 0)
	c.setFlag(flagN, v&flagN != 0)
}

func (c *CPU6502) branch(offset byte) {
	c.PC = uint16(int32(c.PC) + int32(int8(offset)))
}

func (c *CPU6502) skipMicroOps(count int) {
	if c.current == nil || count <= 0 {
		return
	}
	c.current.step += count
}

func (c *CPU6502) pageCrossed(base, target uint16) bool {
	return base&0xFF00 != target&0xFF00
}

func (c *CPU6502) asl(v byte) byte {
	c.setFlag(flagC, v&0x80 != 0)
	out := v << 1
	c.updateZN(out)
	return out
}

func (c *CPU6502) lsr(v byte) byte {
	c.setFlag(flagC, v&0x01 != 0)
	out := v >> 1
	c.updateZN(out)
	return out
}

func (c *CPU6502) rol(v byte) byte {
	carryIn := c.getFlag(flagC)
	c.setFlag(flagC, v&0x80 != 0)
	out := (v << 1) | carryIn
	c.updateZN(out)
	return out
}

func (c *CPU6502) ror(v byte) byte {
	carryIn := c.getFlag(flagC) << 7
	c.setFlag(flagC, v&0x01 != 0)
	out := (v >> 1) | carryIn
	c.updateZN(out)
	return out
}

func (c *CPU6502) Halted() bool {
	return c.halted
}

func (c *CPU6502) decode(opcode byte) (*decodedInstruction, error) {
	if instr, ok := Opcode_6502[opcode]; ok {
		return &decodedInstruction{opcode: opcode, name: instr.Name, steps: instr.action}, nil
	}
	return nil, fmt.Errorf("cpu6502: unsupported opcode 0x%02X at PC=0x%04X", opcode, c.PC-1)
}

// default:
// 	return nil, fmt.Errorf("cpu6502: unsupported opcode 0x%02X at PC=0x%04X", opcode, c.PC-1)
