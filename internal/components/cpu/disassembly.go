package cpu

import (
	"fmt"
	"strings"

	"github.com/Djoulzy/emuai/internal/emulator"
)

type DisassemblyLine struct {
	Address  uint16
	RawBytes string
	Mnemonic string
	Operand  string
	Size     byte
}

func DisassembleBlock(bus *emulator.Bus, start uint16, instructionCount int) ([]DisassemblyLine, error) {
	if bus == nil {
		return nil, fmt.Errorf("cpu6502: bus is nil")
	}
	if instructionCount <= 0 {
		return nil, fmt.Errorf("cpu6502: instruction count must be > 0")
	}

	disassembler := NewCPU6502("disassembler")
	disassembler.SetPC(start)
	lines := make([]DisassemblyLine, 0, instructionCount)
	pc := start

	for range instructionCount {
		opcode, err := bus.Read(pc)
		if err != nil {
			return nil, fmt.Errorf("cpu6502: read opcode at 0x%04X: %w", pc, err)
		}

		instr, err := disassembler.decode(opcode)
		if err != nil {
			lines = append(lines, DisassemblyLine{
				Address:  pc,
				RawBytes: fmt.Sprintf("%02X", opcode),
				Mnemonic: ".DB",
				Operand:  fmt.Sprintf("$%02X", opcode),
				Size:     1,
			})
			pc++
			continue
		}

		rawBytes := disassembler.traceRawBytes(opcode, disassembler.traceOperandBytes(bus, pc, instr.bytes))
		mnemonic, operand := disassembler.traceAssembly(instr, pc, bus)
		lines = append(lines, DisassemblyLine{
			Address:  pc,
			RawBytes: rawBytes,
			Mnemonic: mnemonic,
			Operand:  operand,
			Size:     max(1, instr.bytes),
		})
		pc += uint16(max(1, instr.bytes))
	}

	return lines, nil
}

func FormatDisassembly(lines []DisassemblyLine) string {
	if len(lines) == 0 {
		return "PC     BYTES     ASM"
	}

	var builder strings.Builder
	builder.WriteString("PC     BYTES     ASM\n")
	for idx, line := range lines {
		builder.WriteString(fmt.Sprintf("$%04X  %-8s  %-4s %s", line.Address, line.RawBytes, line.Mnemonic, line.Operand))
		if idx < len(lines)-1 {
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

func max(a, b byte) byte {
	if a > b {
		return a
	}
	return b
}
