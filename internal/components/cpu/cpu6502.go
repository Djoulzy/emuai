package cpu

import (
	"context"
	"fmt"
	"io"
	"strings"

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

	vectorNMI uint16 = 0xFFFA
	vectorIRQ uint16 = 0xFFFE
)

type microOp func(bus *emulator.Bus) error

type decodedInstruction struct {
	opcode byte
	name   string
	bytes  byte
	step   int
	steps  []microOp
}

const (
	traceAnsiReset     = "\x1b[0m"
	traceAnsiDim       = "\x1b[38;5;244m"
	traceAnsiPC        = "\x1b[38;5;81m"
	traceAnsiVisit     = "\x1b[1;30;103m"
	traceAnsiBytes     = "\x1b[38;5;214m"
	traceAnsiMnemonic  = "\x1b[1;38;5;120m"
	traceAnsiOperand   = "\x1b[38;5;223m"
	traceAnsiRegister  = "\x1b[38;5;45m"
	traceAnsiFlags     = "\x1b[38;5;178m"
	traceAnsiInterrupt = "\x1b[1;38;5;203m"
)

type traceByte struct {
	value byte
	valid bool
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
	pendingIRQ     bool
	pendingNMI     bool
	haltOnBRK      bool
	traceWriter    io.Writer
	traceHeaderOut bool
	traceVisitedPC map[uint16]uint32
}

func NewCPU6502(name string, resetVector uint16) *CPU6502 {
	newCPU := &CPU6502{name: name, resetVector: resetVector, haltOnBRK: true}
	newCPU.initLanguage()
	return newCPU
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
	c.pendingIRQ = false
	c.pendingNMI = false
	c.traceHeaderOut = false
	c.traceVisitedPC = make(map[uint16]uint32)
	return nil
}

func (c *CPU6502) Tick(_ context.Context, tick emulator.Tick, bus *emulator.Bus) error {
	if c.halted {
		return nil
	}

	// No current instruction means this cycle is the opcode fetch cycle.
	if c.current == nil {
		if instr := c.pendingInterruptInstruction(); instr != nil {
			c.traceInterrupt(tick.Cycle, c.PC, instr.name)
			if c.hasPrefetch {
				c.hasPrefetch = false
			} else if err := c.dummyReadPC(bus); err != nil {
				return err
			}
			c.current = instr
			return nil
		}

		instructionPC := c.PC
		opcode := c.prefetchOpcode
		if c.hasPrefetch {
			instructionPC = c.PC - 1
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
		c.traceInstruction(tick.Cycle, instructionPC, opcode, instr, bus)
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

func (c *CPU6502) RequestIRQ() {
	c.pendingIRQ = true
}

func (c *CPU6502) RequestNMI() {
	c.pendingNMI = true
}

func (c *CPU6502) SetHaltOnBRK(enabled bool) {
	c.haltOnBRK = enabled
}

func (c *CPU6502) SetTraceWriter(w io.Writer) {
	c.traceWriter = w
	c.traceHeaderOut = false
	c.traceVisitedPC = make(map[uint16]uint32)
}

func (c *CPU6502) pendingInterruptInstruction() *decodedInstruction {
	if c.pendingNMI {
		c.pendingNMI = false
		return &decodedInstruction{name: "NMI", steps: c.interruptSequence(vectorNMI)}
	}
	if c.pendingIRQ && c.getFlag(flagI) == 0 {
		c.pendingIRQ = false
		return &decodedInstruction{name: "IRQ", steps: c.interruptSequence(vectorIRQ)}
	}
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
	carryIn := uint16(c.getFlag(flagC))
	binarySum := uint16(a) + uint16(v) + carryIn
	binaryResult := byte(binarySum)

	if c.getFlag(flagD) != 0 {
		adjusted := binarySum
		if ((uint16(a) & 0x0F) + (uint16(v) & 0x0F) + carryIn) > 0x09 {
			adjusted += 0x06
		}
		if adjusted > 0x99 {
			adjusted += 0x60
		}

		c.setFlag(flagC, adjusted > 0xFF)
		c.setFlag(flagV, ((^(a ^ v))&(a^binaryResult)&0x80) != 0)
		c.A = byte(adjusted)
		c.updateZN(c.A)
		return
	}

	c.setFlag(flagC, binarySum > 0xFF)
	c.setFlag(flagV, ((^(a ^ v))&(a^binaryResult)&0x80) != 0)
	c.A = binaryResult
	c.updateZN(c.A)
}

func (c *CPU6502) sbc(v byte) {
	a := c.A
	carryIn := int16(c.getFlag(flagC))
	borrow := int16(1 - carryIn)
	binaryDiff := int16(a) - int16(v) - borrow
	binaryResult := byte(binaryDiff)

	if c.getFlag(flagD) != 0 {
		low := int16(a&0x0F) - int16(v&0x0F) - borrow
		high := int16(a>>4) - int16(v>>4)

		if low < 0 {
			low -= 0x06
			high--
		}
		if high < 0 {
			high -= 0x06
		}

		c.setFlag(flagC, binaryDiff >= 0)
		c.setFlag(flagV, ((a^v)&(a^binaryResult)&0x80) != 0)
		c.A = byte((high << 4) | (low & 0x0F))
		c.updateZN(c.A)
		return
	}

	c.setFlag(flagC, binaryDiff >= 0)
	c.setFlag(flagV, ((a^v)&(a^binaryResult)&0x80) != 0)
	c.A = binaryResult
	c.updateZN(c.A)
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

func (c *CPU6502) inc(v byte) byte {
	out := v + 1
	c.updateZN(out)
	return out
}

func (c *CPU6502) dec(v byte) byte {
	out := v - 1
	c.updateZN(out)
	return out
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

func (c *CPU6502) ReadyForInstruction() bool {
	return c.current == nil
}

func (c *CPU6502) flagStatusString() string {
	flags := []struct {
		mask  byte
		label byte
	}{
		{mask: flagN, label: 'N'},
		{mask: flagV, label: 'V'},
		{mask: flagU, label: 'U'},
		{mask: flagB, label: 'B'},
		{mask: flagD, label: 'D'},
		{mask: flagI, label: 'I'},
		{mask: flagZ, label: 'Z'},
		{mask: flagC, label: 'C'},
	}

	status := make([]byte, len(flags))
	for i, flag := range flags {
		if c.P&flag.mask != 0 {
			status[i] = flag.label
			continue
		}
		status[i] = '.'
	}

	return string(status)
}

func (c *CPU6502) traceInstruction(cycle uint64, pc uint16, opcode byte, instr *decodedInstruction, bus *emulator.Bus) {
	if c.traceWriter == nil || instr == nil {
		return
	}

	c.writeTraceHeader()
	visit := c.traceVisitMarker(pc)
	rawBytes := c.traceRawBytes(opcode, c.traceOperandBytes(bus, pc, instr.bytes))
	mnemonic, operand := c.traceAssembly(instr, pc, bus)
	_, _ = fmt.Fprintf(
		c.traceWriter,
		"%s  %s  %s  %s  %s %s  %s %s\n",
		c.traceColor(traceAnsiDim, fmt.Sprintf("%-6d", cycle)),
		c.traceColor(traceAnsiPC, fmt.Sprintf("%-5s", fmt.Sprintf("$%04X", pc))),
		visit,
		c.traceColor(traceAnsiBytes, fmt.Sprintf("%-8s", rawBytes)),
		c.traceColor(traceAnsiMnemonic, fmt.Sprintf("%-4s", mnemonic)),
		c.traceColor(traceAnsiOperand, fmt.Sprintf("%-13s", operand)),
		c.traceRegisterState(),
		c.traceColor(traceAnsiFlags, c.flagStatusString()),
	)
}

func (c *CPU6502) traceInterrupt(cycle uint64, pc uint16, name string) {
	if c.traceWriter == nil {
		return
	}

	c.writeTraceHeader()
	visit := c.traceVisitMarker(pc)
	_, _ = fmt.Fprintf(
		c.traceWriter,
		"%s  %s  %s  %s  %s  %s %s\n",
		c.traceColor(traceAnsiDim, fmt.Sprintf("%-6d", cycle)),
		c.traceColor(traceAnsiPC, fmt.Sprintf("%-5s", fmt.Sprintf("$%04X", pc))),
		visit,
		c.traceColor(traceAnsiBytes, fmt.Sprintf("%-8s", "")),
		c.traceColor(traceAnsiInterrupt, fmt.Sprintf("%-18s", name)),
		c.traceRegisterState(),
		c.traceColor(traceAnsiFlags, c.flagStatusString()),
	)
}

func (c *CPU6502) writeTraceHeader() {
	if c.traceWriter == nil || c.traceHeaderOut {
		return
	}

	_, _ = fmt.Fprintf(
		c.traceWriter,
		"%s  %s  %s  %s  %s  %s %s\n",
		c.traceColor(traceAnsiDim, fmt.Sprintf("%-6s", "CYC")),
		c.traceColor(traceAnsiPC, fmt.Sprintf("%-5s", "PC")),
		c.traceColor(traceAnsiVisit, fmt.Sprintf("%-6s", "FLOW")),
		c.traceColor(traceAnsiBytes, fmt.Sprintf("%-8s", "BYTES")),
		c.traceColor(traceAnsiMnemonic, fmt.Sprintf("%-18s", "ASM")),
		c.traceColor(traceAnsiRegister, "REGS"),
		c.traceColor(traceAnsiFlags, "FLAGS"),
	)
	c.traceHeaderOut = true
}

func (c *CPU6502) traceVisitMarker(pc uint16) string {
	if c.traceVisitedPC == nil {
		c.traceVisitedPC = make(map[uint16]uint32)
	}

	visits := c.traceVisitedPC[pc] + 1
	c.traceVisitedPC[pc] = visits

	label := fmt.Sprintf("SEEN#%d", visits)
	color := traceAnsiVisit
	if visits == 1 {
		label = "NEW"
		color = traceAnsiDim
	}

	return c.traceColor(color, fmt.Sprintf("%-6s", label))
}

func (c *CPU6502) traceRegisterState() string {
	return strings.Join([]string{
		c.traceColor(traceAnsiRegister, fmt.Sprintf("A:%02X", c.A)),
		c.traceColor(traceAnsiRegister, fmt.Sprintf("X:%02X", c.X)),
		c.traceColor(traceAnsiRegister, fmt.Sprintf("Y:%02X", c.Y)),
		c.traceColor(traceAnsiRegister, fmt.Sprintf("SP:%02X", c.SP)),
		c.traceColor(traceAnsiRegister, fmt.Sprintf("P:%02X", c.P)),
	}, " ")
}

func (c *CPU6502) traceOperandBytes(bus *emulator.Bus, pc uint16, instructionBytes byte) []traceByte {
	if bus == nil || instructionBytes <= 1 {
		return nil
	}

	operands := make([]traceByte, 0, instructionBytes-1)
	for offset := byte(1); offset < instructionBytes; offset++ {
		value, err := bus.Read(pc + uint16(offset))
		if err != nil {
			operands = append(operands, traceByte{})
			continue
		}
		operands = append(operands, traceByte{value: value, valid: true})
	}

	return operands
}

func (c *CPU6502) traceRawBytes(opcode byte, operands []traceByte) string {
	parts := []string{fmt.Sprintf("%02X", opcode)}
	for _, operand := range operands {
		if operand.valid {
			parts = append(parts, fmt.Sprintf("%02X", operand.value))
			continue
		}
		parts = append(parts, "??")
	}
	return strings.Join(parts, " ")
}

func (c *CPU6502) traceAssembly(instr *decodedInstruction, pc uint16, bus *emulator.Bus) (string, string) {
	if instr == nil {
		return "???", ""
	}

	parts := strings.SplitN(instr.name, " ", 2)
	mnemonic := parts[0]
	operands := c.traceOperandBytes(bus, pc, instr.bytes)
	if len(parts) == 1 {
		if c.isBranchMnemonic(mnemonic) {
			if len(operands) == 0 || !operands[0].valid {
				return mnemonic, "?"
			}
			target := uint16(int32(pc) + int32(instr.bytes) + int32(int8(operands[0].value)))
			return mnemonic, fmt.Sprintf("$%04X", target)
		}
		return mnemonic, ""
	}

	switch parts[1] {
	case "#imm":
		if len(operands) < 1 || !operands[0].valid {
			return mnemonic, "#$??"
		}
		return mnemonic, fmt.Sprintf("#$%02X", operands[0].value)
	case "zp":
		return mnemonic, c.traceFormatZeroPage("$%s", operands)
	case "zp,X":
		return mnemonic, c.traceFormatZeroPage("$%s,X", operands)
	case "zp,Y":
		return mnemonic, c.traceFormatZeroPage("$%s,Y", operands)
	case "abs":
		return mnemonic, c.traceFormatAbsolute("$%s", operands)
	case "abs,X":
		return mnemonic, c.traceFormatAbsolute("$%s,X", operands)
	case "abs,Y":
		return mnemonic, c.traceFormatAbsolute("$%s,Y", operands)
	case "(zp,X)":
		return mnemonic, c.traceFormatZeroPage("($%s,X)", operands)
	case "(zp),Y":
		return mnemonic, c.traceFormatZeroPage("($%s),Y", operands)
	case "ind":
		return mnemonic, c.traceFormatAbsolute("($%s)", operands)
	case "A":
		return mnemonic, "A"
	default:
		return mnemonic, parts[1]
	}
}

func (c *CPU6502) traceFormatZeroPage(pattern string, operands []traceByte) string {
	if len(operands) < 1 || !operands[0].valid {
		return fmt.Sprintf(pattern, "??")
	}
	return fmt.Sprintf(pattern, fmt.Sprintf("%02X", operands[0].value))
}

func (c *CPU6502) traceFormatAbsolute(pattern string, operands []traceByte) string {
	if len(operands) < 2 || !operands[0].valid || !operands[1].valid {
		return fmt.Sprintf(pattern, "????")
	}
	value := uint16(operands[1].value)<<8 | uint16(operands[0].value)
	return fmt.Sprintf(pattern, fmt.Sprintf("%04X", value))
}

func (c *CPU6502) isBranchMnemonic(mnemonic string) bool {
	switch mnemonic {
	case "BCC", "BCS", "BEQ", "BMI", "BNE", "BPL", "BVC", "BVS":
		return true
	default:
		return false
	}
}

func (c *CPU6502) traceColor(code, text string) string {
	return code + text + traceAnsiReset
}

func (c *CPU6502) decode(opcode byte) (*decodedInstruction, error) {
	if instr, ok := Opcode_6502[opcode]; ok {
		return &decodedInstruction{opcode: opcode, name: instr.Name, bytes: instr.Bytes, steps: instr.action}, nil
	}
	return nil, fmt.Errorf("cpu6502: unsupported opcode 0x%02X at PC=0x%04X", opcode, c.PC-1)
}

// default:
// 	return nil, fmt.Errorf("cpu6502: unsupported opcode 0x%02X at PC=0x%04X", opcode, c.PC-1)
