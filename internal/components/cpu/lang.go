package cpu

import "github.com/Djoulzy/emuai/internal/emulator"

type Instruction struct {
	Name   string
	Bytes  byte
	Cycles int
	action []microOp
}

var Opcode_6502 = make(map[byte]Instruction)

type byteConsumer func(byte)
type byteProducer func() byte
type byteTransformer func(byte) byte

func noopMicroOp(_ *emulator.Bus) error {
	return nil
}

func (c *CPU6502) implied(fn func()) []microOp {
	return []microOp{
		func(_ *emulator.Bus) error {
			fn()
			return nil
		},
	}
}

func (c *CPU6502) immediateRead(fn byteConsumer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			v, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) zeroPageRead(fn byteConsumer) []microOp {
	return []microOp{
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
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) zeroPageXRead(fn byteConsumer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			v, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = v
			return nil
		},
		func(_ *emulator.Bus) error {
			c.tmpAddr = uint16(byte(c.tmp8 + c.X))
			return nil
		},
		func(bus *emulator.Bus) error {
			v, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) zeroPageYRead(fn byteConsumer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			v, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = v
			return nil
		},
		func(_ *emulator.Bus) error {
			c.tmpAddr = uint16(byte(c.tmp8 + c.Y))
			return nil
		},
		func(bus *emulator.Bus) error {
			v, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) absoluteRead(fn byteConsumer) []microOp {
	return []microOp{
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
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) absoluteXRead(fn byteConsumer) []microOp {
	return []microOp{
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
			c.tmpBase = (uint16(hi) << 8) | uint16(c.tmp8)
			c.tmpAddr = c.tmpBase + uint16(c.X)
			return nil
		},
		func(bus *emulator.Bus) error {
			if !c.pageCrossed(c.tmpBase, c.tmpAddr) {
				v, err := bus.Read(c.tmpAddr)
				if err != nil {
					return err
				}
				fn(v)
				c.skipMicroOps(1)
				return nil
			}
			dummyAddr := (c.tmpBase & 0xFF00) | (c.tmpAddr & 0x00FF)
			_, err := bus.Read(dummyAddr)
			return err
		},
		func(bus *emulator.Bus) error {
			v, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) absoluteYRead(fn byteConsumer) []microOp {
	return []microOp{
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
			c.tmpBase = (uint16(hi) << 8) | uint16(c.tmp8)
			c.tmpAddr = c.tmpBase + uint16(c.Y)
			return nil
		},
		func(bus *emulator.Bus) error {
			if !c.pageCrossed(c.tmpBase, c.tmpAddr) {
				v, err := bus.Read(c.tmpAddr)
				if err != nil {
					return err
				}
				fn(v)
				c.skipMicroOps(1)
				return nil
			}
			dummyAddr := (c.tmpBase & 0xFF00) | (c.tmpAddr & 0x00FF)
			_, err := bus.Read(dummyAddr)
			return err
		},
		func(bus *emulator.Bus) error {
			v, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) indirectXRead(fn byteConsumer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			base, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = base
			return nil
		},
		func(_ *emulator.Bus) error {
			c.tmpAddr = uint16(byte(c.tmp8 + c.X))
			return nil
		},
		func(bus *emulator.Bus) error {
			lo, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hi, err := bus.Read(uint16(byte(byte(c.tmpAddr) + 1)))
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
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) indirectYRead(fn byteConsumer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			base, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmpAddr = uint16(base)
			return nil
		},
		func(bus *emulator.Bus) error {
			lo, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hi, err := bus.Read(uint16(byte(byte(c.tmpAddr) + 1)))
			if err != nil {
				return err
			}
			c.tmpBase = (uint16(hi) << 8) | uint16(c.tmp8)
			c.tmpAddr = c.tmpBase + uint16(c.Y)
			return nil
		},
		func(bus *emulator.Bus) error {
			if !c.pageCrossed(c.tmpBase, c.tmpAddr) {
				v, err := bus.Read(c.tmpAddr)
				if err != nil {
					return err
				}
				fn(v)
				c.skipMicroOps(1)
				return nil
			}
			dummyAddr := (c.tmpBase & 0xFF00) | (c.tmpAddr & 0x00FF)
			_, err := bus.Read(dummyAddr)
			return err
		},
		func(bus *emulator.Bus) error {
			v, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			fn(v)
			return nil
		},
	}
}

func (c *CPU6502) zeroPageWrite(value byteProducer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			v, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmpAddr = uint16(v)
			return nil
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, value())
		},
	}
}

func (c *CPU6502) zeroPageXWrite(value byteProducer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			v, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = v
			return nil
		},
		func(_ *emulator.Bus) error {
			c.tmpAddr = uint16(byte(c.tmp8 + c.X))
			return nil
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, value())
		},
	}
}

func (c *CPU6502) zeroPageYWrite(value byteProducer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			v, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = v
			return nil
		},
		func(_ *emulator.Bus) error {
			c.tmpAddr = uint16(byte(c.tmp8 + c.Y))
			return nil
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, value())
		},
	}
}

func (c *CPU6502) absoluteWrite(value byteProducer) []microOp {
	return []microOp{
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
			return bus.Write(c.tmpAddr, value())
		},
	}
}

func (c *CPU6502) absoluteXWrite(value byteProducer) []microOp {
	return []microOp{
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
			c.tmpBase = (uint16(hi) << 8) | uint16(c.tmp8)
			c.tmpAddr = c.tmpBase + uint16(c.X)
			return nil
		},
		func(bus *emulator.Bus) error {
			dummyAddr := (c.tmpBase & 0xFF00) | (c.tmpAddr & 0x00FF)
			_, err := bus.Read(dummyAddr)
			return err
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, value())
		},
	}
}

func (c *CPU6502) absoluteYWrite(value byteProducer) []microOp {
	return []microOp{
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
			c.tmpBase = (uint16(hi) << 8) | uint16(c.tmp8)
			c.tmpAddr = c.tmpBase + uint16(c.Y)
			return nil
		},
		func(bus *emulator.Bus) error {
			dummyAddr := (c.tmpBase & 0xFF00) | (c.tmpAddr & 0x00FF)
			_, err := bus.Read(dummyAddr)
			return err
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, value())
		},
	}
}

func (c *CPU6502) indirectXWrite(value byteProducer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			base, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = base
			return nil
		},
		func(_ *emulator.Bus) error {
			c.tmpAddr = uint16(byte(c.tmp8 + c.X))
			return nil
		},
		func(bus *emulator.Bus) error {
			lo, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hi, err := bus.Read(uint16(byte(byte(c.tmpAddr) + 1)))
			if err != nil {
				return err
			}
			c.tmpAddr = uint16(hi)<<8 | uint16(c.tmp8)
			return nil
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, value())
		},
	}
}

func (c *CPU6502) indirectYWrite(value byteProducer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			base, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmpAddr = uint16(base)
			return nil
		},
		func(bus *emulator.Bus) error {
			lo, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hi, err := bus.Read(uint16(byte(byte(c.tmpAddr) + 1)))
			if err != nil {
				return err
			}
			c.tmpBase = (uint16(hi) << 8) | uint16(c.tmp8)
			c.tmpAddr = c.tmpBase + uint16(c.Y)
			return nil
		},
		func(bus *emulator.Bus) error {
			dummyAddr := (c.tmpBase & 0xFF00) | (c.tmpAddr & 0x00FF)
			_, err := bus.Read(dummyAddr)
			return err
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, value())
		},
	}
}

func (c *CPU6502) zeroPageModify(transform byteTransformer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			addr, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmpAddr = uint16(addr)
			return nil
		},
		func(bus *emulator.Bus) error {
			v, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			c.tmp8 = v
			return nil
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, c.tmp8)
		},
		func(bus *emulator.Bus) error {
			c.tmp8 = transform(c.tmp8)
			return bus.Write(c.tmpAddr, c.tmp8)
		},
	}
}

func (c *CPU6502) zeroPageXModify(transform byteTransformer) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			addr, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = addr
			return nil
		},
		func(_ *emulator.Bus) error {
			c.tmpAddr = uint16(byte(c.tmp8 + c.X))
			return nil
		},
		func(bus *emulator.Bus) error {
			v, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			c.tmp8 = v
			return nil
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, c.tmp8)
		},
		func(bus *emulator.Bus) error {
			c.tmp8 = transform(c.tmp8)
			return bus.Write(c.tmpAddr, c.tmp8)
		},
	}
}

func (c *CPU6502) absoluteModify(transform byteTransformer) []microOp {
	return []microOp{
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
			c.tmp8 = v
			return nil
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, c.tmp8)
		},
		func(bus *emulator.Bus) error {
			c.tmp8 = transform(c.tmp8)
			return bus.Write(c.tmpAddr, c.tmp8)
		},
	}
}

func (c *CPU6502) absoluteXModify(transform byteTransformer) []microOp {
	return []microOp{
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
			c.tmpBase = (uint16(hi) << 8) | uint16(c.tmp8)
			c.tmpAddr = c.tmpBase + uint16(c.X)
			return nil
		},
		func(bus *emulator.Bus) error {
			dummyAddr := (c.tmpBase & 0xFF00) | (c.tmpAddr & 0x00FF)
			_, err := bus.Read(dummyAddr)
			return err
		},
		func(bus *emulator.Bus) error {
			v, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			c.tmp8 = v
			return nil
		},
		func(bus *emulator.Bus) error {
			return bus.Write(c.tmpAddr, c.tmp8)
		},
		func(bus *emulator.Bus) error {
			c.tmp8 = transform(c.tmp8)
			return bus.Write(c.tmpAddr, c.tmp8)
		},
	}
}

func (c *CPU6502) accumulatorModify(transform byteTransformer) []microOp {
	return []microOp{
		func(_ *emulator.Bus) error {
			c.A = transform(c.A)
			return nil
		},
	}
}

func (c *CPU6502) relativeBranch(condition func() bool) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			offset, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = offset
			if !condition() {
				c.skipMicroOps(2)
			}
			return nil
		},
		func(_ *emulator.Bus) error {
			base := c.PC
			c.branch(c.tmp8)
			if !c.pageCrossed(base, c.PC) {
				c.skipMicroOps(1)
			}
			return nil
		},
		noopMicroOp,
	}
}

func (c *CPU6502) interruptSequence(vector uint16) []microOp {
	return []microOp{
		func(bus *emulator.Bus) error { return c.pushByte(bus, byte(c.PC>>8)) },
		func(bus *emulator.Bus) error { return c.pushByte(bus, byte(c.PC)) },
		func(bus *emulator.Bus) error {
			if err := c.pushByte(bus, c.status(false)); err != nil {
				return err
			}
			c.setFlag(flagI, true)
			return nil
		},
		func(bus *emulator.Bus) error {
			lo, err := bus.Read(vector)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hi, err := bus.Read(vector + 1)
			if err != nil {
				return err
			}
			c.PC = uint16(hi)<<8 | uint16(c.tmp8)
			return nil
		},
	}
}

func (c *CPU6502) brkSequence() []microOp {
	return []microOp{
		func(bus *emulator.Bus) error {
			_, err := c.fetchByte(bus)
			return err
		},
		func(bus *emulator.Bus) error { return c.pushByte(bus, byte(c.PC>>8)) },
		func(bus *emulator.Bus) error { return c.pushByte(bus, byte(c.PC)) },
		func(bus *emulator.Bus) error {
			if err := c.pushByte(bus, c.status(true)); err != nil {
				return err
			}
			c.setFlag(flagI, true)
			return nil
		},
		func(bus *emulator.Bus) error {
			lo, err := bus.Read(vectorIRQ)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hi, err := bus.Read(vectorIRQ + 1)
			if err != nil {
				return err
			}
			c.PC = uint16(hi)<<8 | uint16(c.tmp8)
			if c.haltOnBRK {
				c.halted = true
			}
			return nil
		},
	}
}

func (c *CPU6502) initLanguage() {
	Opcode_6502 = make(map[byte]Instruction, 151)

	register := func(opcode byte, name string, bytes byte, cycles int, action []microOp) {
		Opcode_6502[opcode] = Instruction{Name: name, Bytes: bytes, Cycles: cycles, action: action}
	}

	loadA := func(v byte) {
		c.A = v
		c.updateZN(c.A)
	}
	loadX := func(v byte) {
		c.X = v
		c.updateZN(c.X)
	}
	loadY := func(v byte) {
		c.Y = v
		c.updateZN(c.Y)
	}

	register(0x00, "BRK", 1, 7, c.brkSequence())
	register(0xEA, "NOP", 1, 2, []microOp{noopMicroOp})

	register(0x69, "ADC #imm", 2, 2, c.immediateRead(c.adc))
	register(0x65, "ADC zp", 2, 3, c.zeroPageRead(c.adc))
	register(0x75, "ADC zp,X", 2, 4, c.zeroPageXRead(c.adc))
	register(0x6D, "ADC abs", 3, 4, c.absoluteRead(c.adc))
	register(0x7D, "ADC abs,X", 3, 4, c.absoluteXRead(c.adc))
	register(0x79, "ADC abs,Y", 3, 4, c.absoluteYRead(c.adc))
	register(0x61, "ADC (zp,X)", 2, 6, c.indirectXRead(c.adc))
	register(0x71, "ADC (zp),Y", 2, 5, c.indirectYRead(c.adc))

	register(0x29, "AND #imm", 2, 2, c.immediateRead(func(v byte) { loadA(c.A & v) }))
	register(0x25, "AND zp", 2, 3, c.zeroPageRead(func(v byte) { loadA(c.A & v) }))
	register(0x35, "AND zp,X", 2, 4, c.zeroPageXRead(func(v byte) { loadA(c.A & v) }))
	register(0x2D, "AND abs", 3, 4, c.absoluteRead(func(v byte) { loadA(c.A & v) }))
	register(0x3D, "AND abs,X", 3, 4, c.absoluteXRead(func(v byte) { loadA(c.A & v) }))
	register(0x39, "AND abs,Y", 3, 4, c.absoluteYRead(func(v byte) { loadA(c.A & v) }))
	register(0x21, "AND (zp,X)", 2, 6, c.indirectXRead(func(v byte) { loadA(c.A & v) }))
	register(0x31, "AND (zp),Y", 2, 5, c.indirectYRead(func(v byte) { loadA(c.A & v) }))

	register(0x0A, "ASL A", 1, 2, c.accumulatorModify(c.asl))
	register(0x06, "ASL zp", 2, 5, c.zeroPageModify(c.asl))
	register(0x16, "ASL zp,X", 2, 6, c.zeroPageXModify(c.asl))
	register(0x0E, "ASL abs", 3, 6, c.absoluteModify(c.asl))
	register(0x1E, "ASL abs,X", 3, 7, c.absoluteXModify(c.asl))

	register(0x90, "BCC", 2, 2, c.relativeBranch(func() bool { return c.getFlag(flagC) == 0 }))
	register(0xB0, "BCS", 2, 2, c.relativeBranch(func() bool { return c.getFlag(flagC) != 0 }))
	register(0xF0, "BEQ", 2, 2, c.relativeBranch(func() bool { return c.getFlag(flagZ) != 0 }))
	register(0x30, "BMI", 2, 2, c.relativeBranch(func() bool { return c.getFlag(flagN) != 0 }))
	register(0xD0, "BNE", 2, 2, c.relativeBranch(func() bool { return c.getFlag(flagZ) == 0 }))
	register(0x10, "BPL", 2, 2, c.relativeBranch(func() bool { return c.getFlag(flagN) == 0 }))
	register(0x50, "BVC", 2, 2, c.relativeBranch(func() bool { return c.getFlag(flagV) == 0 }))
	register(0x70, "BVS", 2, 2, c.relativeBranch(func() bool { return c.getFlag(flagV) != 0 }))

	register(0x24, "BIT zp", 2, 3, c.zeroPageRead(c.bit))
	register(0x2C, "BIT abs", 3, 4, c.absoluteRead(c.bit))

	register(0x18, "CLC", 1, 2, c.implied(func() { c.setFlag(flagC, false) }))
	register(0xD8, "CLD", 1, 2, c.implied(func() { c.setFlag(flagD, false) }))
	register(0x58, "CLI", 1, 2, c.implied(func() { c.setFlag(flagI, false) }))
	register(0xB8, "CLV", 1, 2, c.implied(func() { c.setFlag(flagV, false) }))

	register(0xC9, "CMP #imm", 2, 2, c.immediateRead(func(v byte) { c.compare(c.A, v) }))
	register(0xC5, "CMP zp", 2, 3, c.zeroPageRead(func(v byte) { c.compare(c.A, v) }))
	register(0xD5, "CMP zp,X", 2, 4, c.zeroPageXRead(func(v byte) { c.compare(c.A, v) }))
	register(0xCD, "CMP abs", 3, 4, c.absoluteRead(func(v byte) { c.compare(c.A, v) }))
	register(0xDD, "CMP abs,X", 3, 4, c.absoluteXRead(func(v byte) { c.compare(c.A, v) }))
	register(0xD9, "CMP abs,Y", 3, 4, c.absoluteYRead(func(v byte) { c.compare(c.A, v) }))
	register(0xC1, "CMP (zp,X)", 2, 6, c.indirectXRead(func(v byte) { c.compare(c.A, v) }))
	register(0xD1, "CMP (zp),Y", 2, 5, c.indirectYRead(func(v byte) { c.compare(c.A, v) }))

	register(0xE0, "CPX #imm", 2, 2, c.immediateRead(func(v byte) { c.compare(c.X, v) }))
	register(0xE4, "CPX zp", 2, 3, c.zeroPageRead(func(v byte) { c.compare(c.X, v) }))
	register(0xEC, "CPX abs", 3, 4, c.absoluteRead(func(v byte) { c.compare(c.X, v) }))

	register(0xC0, "CPY #imm", 2, 2, c.immediateRead(func(v byte) { c.compare(c.Y, v) }))
	register(0xC4, "CPY zp", 2, 3, c.zeroPageRead(func(v byte) { c.compare(c.Y, v) }))
	register(0xCC, "CPY abs", 3, 4, c.absoluteRead(func(v byte) { c.compare(c.Y, v) }))

	register(0xC6, "DEC zp", 2, 5, c.zeroPageModify(c.dec))
	register(0xD6, "DEC zp,X", 2, 6, c.zeroPageXModify(c.dec))
	register(0xCE, "DEC abs", 3, 6, c.absoluteModify(c.dec))
	register(0xDE, "DEC abs,X", 3, 7, c.absoluteXModify(c.dec))

	register(0xCA, "DEX", 1, 2, c.implied(func() { c.X--; c.updateZN(c.X) }))
	register(0x88, "DEY", 1, 2, c.implied(func() { c.Y--; c.updateZN(c.Y) }))

	register(0x49, "EOR #imm", 2, 2, c.immediateRead(func(v byte) { loadA(c.A ^ v) }))
	register(0x45, "EOR zp", 2, 3, c.zeroPageRead(func(v byte) { loadA(c.A ^ v) }))
	register(0x55, "EOR zp,X", 2, 4, c.zeroPageXRead(func(v byte) { loadA(c.A ^ v) }))
	register(0x4D, "EOR abs", 3, 4, c.absoluteRead(func(v byte) { loadA(c.A ^ v) }))
	register(0x5D, "EOR abs,X", 3, 4, c.absoluteXRead(func(v byte) { loadA(c.A ^ v) }))
	register(0x59, "EOR abs,Y", 3, 4, c.absoluteYRead(func(v byte) { loadA(c.A ^ v) }))
	register(0x41, "EOR (zp,X)", 2, 6, c.indirectXRead(func(v byte) { loadA(c.A ^ v) }))
	register(0x51, "EOR (zp),Y", 2, 5, c.indirectYRead(func(v byte) { loadA(c.A ^ v) }))

	register(0xE6, "INC zp", 2, 5, c.zeroPageModify(c.inc))
	register(0xF6, "INC zp,X", 2, 6, c.zeroPageXModify(c.inc))
	register(0xEE, "INC abs", 3, 6, c.absoluteModify(c.inc))
	register(0xFE, "INC abs,X", 3, 7, c.absoluteXModify(c.inc))

	register(0xE8, "INX", 1, 2, c.implied(func() { c.X++; c.updateZN(c.X) }))
	register(0xC8, "INY", 1, 2, c.implied(func() { c.Y++; c.updateZN(c.Y) }))

	register(0x4C, "JMP abs", 3, 3, []microOp{
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
	})
	register(0x6C, "JMP ind", 3, 5, []microOp{
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
			lo, err := bus.Read(c.tmpAddr)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hiAddr := (c.tmpAddr & 0xFF00) | uint16(byte(c.tmpAddr+1))
			hi, err := bus.Read(hiAddr)
			if err != nil {
				return err
			}
			c.PC = uint16(hi)<<8 | uint16(c.tmp8)
			return nil
		},
	})
	register(0x20, "JSR abs", 3, 6, []microOp{
		func(bus *emulator.Bus) error {
			lo, err := c.fetchByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error { return c.dummyReadStack(bus) },
		func(bus *emulator.Bus) error { return c.pushByte(bus, byte(c.PC>>8)) },
		func(bus *emulator.Bus) error {
			if err := c.pushByte(bus, byte(c.PC)); err != nil {
				return err
			}
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
	})

	register(0xA9, "LDA #imm", 2, 2, c.immediateRead(loadA))
	register(0xA5, "LDA zp", 2, 3, c.zeroPageRead(loadA))
	register(0xB5, "LDA zp,X", 2, 4, c.zeroPageXRead(loadA))
	register(0xAD, "LDA abs", 3, 4, c.absoluteRead(loadA))
	register(0xBD, "LDA abs,X", 3, 4, c.absoluteXRead(loadA))
	register(0xB9, "LDA abs,Y", 3, 4, c.absoluteYRead(loadA))
	register(0xA1, "LDA (zp,X)", 2, 6, c.indirectXRead(loadA))
	register(0xB1, "LDA (zp),Y", 2, 5, c.indirectYRead(loadA))

	register(0xA2, "LDX #imm", 2, 2, c.immediateRead(loadX))
	register(0xA6, "LDX zp", 2, 3, c.zeroPageRead(loadX))
	register(0xB6, "LDX zp,Y", 2, 4, c.zeroPageYRead(loadX))
	register(0xAE, "LDX abs", 3, 4, c.absoluteRead(loadX))
	register(0xBE, "LDX abs,Y", 3, 4, c.absoluteYRead(loadX))

	register(0xA0, "LDY #imm", 2, 2, c.immediateRead(loadY))
	register(0xA4, "LDY zp", 2, 3, c.zeroPageRead(loadY))
	register(0xB4, "LDY zp,X", 2, 4, c.zeroPageXRead(loadY))
	register(0xAC, "LDY abs", 3, 4, c.absoluteRead(loadY))
	register(0xBC, "LDY abs,X", 3, 4, c.absoluteXRead(loadY))

	register(0x4A, "LSR A", 1, 2, c.accumulatorModify(c.lsr))
	register(0x46, "LSR zp", 2, 5, c.zeroPageModify(c.lsr))
	register(0x56, "LSR zp,X", 2, 6, c.zeroPageXModify(c.lsr))
	register(0x4E, "LSR abs", 3, 6, c.absoluteModify(c.lsr))
	register(0x5E, "LSR abs,X", 3, 7, c.absoluteXModify(c.lsr))

	register(0x09, "ORA #imm", 2, 2, c.immediateRead(func(v byte) { loadA(c.A | v) }))
	register(0x05, "ORA zp", 2, 3, c.zeroPageRead(func(v byte) { loadA(c.A | v) }))
	register(0x15, "ORA zp,X", 2, 4, c.zeroPageXRead(func(v byte) { loadA(c.A | v) }))
	register(0x0D, "ORA abs", 3, 4, c.absoluteRead(func(v byte) { loadA(c.A | v) }))
	register(0x1D, "ORA abs,X", 3, 4, c.absoluteXRead(func(v byte) { loadA(c.A | v) }))
	register(0x19, "ORA abs,Y", 3, 4, c.absoluteYRead(func(v byte) { loadA(c.A | v) }))
	register(0x01, "ORA (zp,X)", 2, 6, c.indirectXRead(func(v byte) { loadA(c.A | v) }))
	register(0x11, "ORA (zp),Y", 2, 5, c.indirectYRead(func(v byte) { loadA(c.A | v) }))

	register(0x48, "PHA", 1, 3, []microOp{func(bus *emulator.Bus) error { return c.dummyReadPC(bus) }, func(bus *emulator.Bus) error { return c.pushByte(bus, c.A) }})
	register(0x08, "PHP", 1, 3, []microOp{func(bus *emulator.Bus) error { return c.dummyReadPC(bus) }, func(bus *emulator.Bus) error { return c.pushByte(bus, c.status(true)) }})
	register(0x68, "PLA", 1, 4, []microOp{func(bus *emulator.Bus) error { return c.dummyReadPC(bus) }, func(bus *emulator.Bus) error { return c.dummyReadStack(bus) }, func(bus *emulator.Bus) error {
		v, err := c.pullByte(bus)
		if err != nil {
			return err
		}
		c.A = v
		c.updateZN(c.A)
		return nil
	}})
	register(0x28, "PLP", 1, 4, []microOp{func(bus *emulator.Bus) error { return c.dummyReadPC(bus) }, func(bus *emulator.Bus) error { return c.dummyReadStack(bus) }, func(bus *emulator.Bus) error {
		v, err := c.pullByte(bus)
		if err != nil {
			return err
		}
		c.setStatus(v &^ flagB)
		return nil
	}})

	register(0x2A, "ROL A", 1, 2, c.accumulatorModify(c.rol))
	register(0x26, "ROL zp", 2, 5, c.zeroPageModify(c.rol))
	register(0x36, "ROL zp,X", 2, 6, c.zeroPageXModify(c.rol))
	register(0x2E, "ROL abs", 3, 6, c.absoluteModify(c.rol))
	register(0x3E, "ROL abs,X", 3, 7, c.absoluteXModify(c.rol))

	register(0x6A, "ROR A", 1, 2, c.accumulatorModify(c.ror))
	register(0x66, "ROR zp", 2, 5, c.zeroPageModify(c.ror))
	register(0x76, "ROR zp,X", 2, 6, c.zeroPageXModify(c.ror))
	register(0x6E, "ROR abs", 3, 6, c.absoluteModify(c.ror))
	register(0x7E, "ROR abs,X", 3, 7, c.absoluteXModify(c.ror))

	register(0x40, "RTI", 1, 6, []microOp{
		func(bus *emulator.Bus) error { return c.dummyReadPC(bus) },
		func(bus *emulator.Bus) error { return c.dummyReadStack(bus) },
		func(bus *emulator.Bus) error {
			status, err := c.pullByte(bus)
			if err != nil {
				return err
			}
			c.setStatus(status &^ flagB)
			return nil
		},
		func(bus *emulator.Bus) error {
			lo, err := c.pullByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hi, err := c.pullByte(bus)
			if err != nil {
				return err
			}
			c.PC = uint16(hi)<<8 | uint16(c.tmp8)
			return nil
		},
	})
	register(0x60, "RTS", 1, 6, []microOp{
		func(bus *emulator.Bus) error { return c.dummyReadPC(bus) },
		func(bus *emulator.Bus) error { return c.dummyReadStack(bus) },
		func(bus *emulator.Bus) error {
			lo, err := c.pullByte(bus)
			if err != nil {
				return err
			}
			c.tmp8 = lo
			return nil
		},
		func(bus *emulator.Bus) error {
			hi, err := c.pullByte(bus)
			if err != nil {
				return err
			}
			c.tmpAddr = uint16(hi)<<8 | uint16(c.tmp8)
			return nil
		},
		func(bus *emulator.Bus) error { return c.prefetch(bus, c.tmpAddr+1) },
	})

	register(0xE9, "SBC #imm", 2, 2, c.immediateRead(c.sbc))
	register(0xE5, "SBC zp", 2, 3, c.zeroPageRead(c.sbc))
	register(0xF5, "SBC zp,X", 2, 4, c.zeroPageXRead(c.sbc))
	register(0xED, "SBC abs", 3, 4, c.absoluteRead(c.sbc))
	register(0xFD, "SBC abs,X", 3, 4, c.absoluteXRead(c.sbc))
	register(0xF9, "SBC abs,Y", 3, 4, c.absoluteYRead(c.sbc))
	register(0xE1, "SBC (zp,X)", 2, 6, c.indirectXRead(c.sbc))
	register(0xF1, "SBC (zp),Y", 2, 5, c.indirectYRead(c.sbc))

	register(0x38, "SEC", 1, 2, c.implied(func() { c.setFlag(flagC, true) }))
	register(0xF8, "SED", 1, 2, c.implied(func() { c.setFlag(flagD, true) }))
	register(0x78, "SEI", 1, 2, c.implied(func() { c.setFlag(flagI, true) }))

	register(0x85, "STA zp", 2, 3, c.zeroPageWrite(func() byte { return c.A }))
	register(0x95, "STA zp,X", 2, 4, c.zeroPageXWrite(func() byte { return c.A }))
	register(0x8D, "STA abs", 3, 4, c.absoluteWrite(func() byte { return c.A }))
	register(0x9D, "STA abs,X", 3, 5, c.absoluteXWrite(func() byte { return c.A }))
	register(0x99, "STA abs,Y", 3, 5, c.absoluteYWrite(func() byte { return c.A }))
	register(0x81, "STA (zp,X)", 2, 6, c.indirectXWrite(func() byte { return c.A }))
	register(0x91, "STA (zp),Y", 2, 6, c.indirectYWrite(func() byte { return c.A }))

	register(0x86, "STX zp", 2, 3, c.zeroPageWrite(func() byte { return c.X }))
	register(0x96, "STX zp,Y", 2, 4, c.zeroPageYWrite(func() byte { return c.X }))
	register(0x8E, "STX abs", 3, 4, c.absoluteWrite(func() byte { return c.X }))

	register(0x84, "STY zp", 2, 3, c.zeroPageWrite(func() byte { return c.Y }))
	register(0x94, "STY zp,X", 2, 4, c.zeroPageXWrite(func() byte { return c.Y }))
	register(0x8C, "STY abs", 3, 4, c.absoluteWrite(func() byte { return c.Y }))

	register(0xAA, "TAX", 1, 2, c.implied(func() { loadX(c.A) }))
	register(0xA8, "TAY", 1, 2, c.implied(func() { loadY(c.A) }))
	register(0xBA, "TSX", 1, 2, c.implied(func() { loadX(c.SP) }))
	register(0x8A, "TXA", 1, 2, c.implied(func() { loadA(c.X) }))
	register(0x9A, "TXS", 1, 2, c.implied(func() { c.SP = c.X }))
	register(0x98, "TYA", 1, 2, c.implied(func() { loadA(c.Y) }))
}
