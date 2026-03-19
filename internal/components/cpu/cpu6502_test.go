package cpu

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/Djoulzy/emuai/internal/components/memory"
	"github.com/Djoulzy/emuai/internal/emulator"
)

type traceRAM struct {
	data   [65536]byte
	reads  map[uint16]int
	writes map[uint16]int
	log    []traceAccess
}

type traceAccess struct {
	kind  byte
	addr  uint16
	value byte
}

func newTraceRAM() *traceRAM {
	return &traceRAM{
		reads:  make(map[uint16]int),
		writes: make(map[uint16]int),
	}
}

func (r *traceRAM) Read(addr uint16) (byte, error) {
	r.reads[addr]++
	v := r.data[addr]
	r.log = append(r.log, traceAccess{kind: 'R', addr: addr, value: v})
	return v, nil
}

func (r *traceRAM) Write(addr uint16, value byte) error {
	r.writes[addr]++
	r.data[addr] = value
	r.log = append(r.log, traceAccess{kind: 'W', addr: addr, value: value})
	return nil
}

func (r *traceRAM) clearTrace() {
	r.reads = make(map[uint16]int)
	r.writes = make(map[uint16]int)
	r.log = nil
}

func newTestCPU(t *testing.T, program []byte) (*CPU6502, *emulator.Bus) {
	t.Helper()
	return newTestCPUAt(t, 0x0200, program)
}

func newTestCPUAt(t *testing.T, resetVector uint16, program []byte) (*CPU6502, *emulator.Bus) {
	t.Helper()

	bus := emulator.NewBus()
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	for i, b := range program {
		if err := bus.Write(resetVector+uint16(i), b); err != nil {
			t.Fatalf("seed RAM: %v", err)
		}
	}

	c := NewCPU6502("cpu", resetVector)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	return c, bus
}

func runUntilHalt(t *testing.T, c *CPU6502, bus *emulator.Bus, maxCycles int) {
	t.Helper()

	for i := 0; i < maxCycles && !c.Halted(); i++ {
		if err := c.Tick(context.Background(), emulator.Tick{Cycle: uint64(i)}, bus); err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
	}
	if !c.Halted() {
		t.Fatalf("expected CPU to halt within %d cycles", maxCycles)
	}
}

func tickOnce(t *testing.T, c *CPU6502, bus *emulator.Bus, cycle uint64) {
	t.Helper()

	if err := c.Tick(context.Background(), emulator.Tick{Cycle: cycle}, bus); err != nil {
		t.Fatalf("tick %d: %v", cycle, err)
	}
}

func newTraceCPUAt(t *testing.T, resetVector uint16, program []byte) (*CPU6502, *emulator.Bus, *traceRAM) {
	t.Helper()

	trace := newTraceRAM()
	bus := emulator.NewBus()
	if err := bus.MapDevice(0x0000, 0xFFFF, "trace", trace); err != nil {
		t.Fatalf("map trace RAM: %v", err)
	}
	for i, b := range program {
		if err := bus.Write(resetVector+uint16(i), b); err != nil {
			t.Fatalf("seed trace RAM: %v", err)
		}
	}

	c := NewCPU6502("cpu", resetVector)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}
	trace.clearTrace()

	return c, bus, trace
}

func assertTrace(t *testing.T, got []traceAccess, want []traceAccess) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d bus accesses, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected access %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func writeBytes(t *testing.T, bus *emulator.Bus, start uint16, data []byte) {
	t.Helper()

	for i, b := range data {
		if err := bus.Write(start+uint16(i), b); err != nil {
			t.Fatalf("write 0x%04X: %v", start+uint16(i), err)
		}
	}
}

func TestCPU6502_ProgramFlow(t *testing.T) {
	bus := emulator.NewBus()
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	program := []byte{
		0xA9, 0x05, // LDA #$05
		0x69, 0x03, // ADC #$03 => A = 8
		0x85, 0x10, // STA $10
		0xA5, 0x10, // LDA $10
		0xAA, // TAX
		0xE8, // INX => X = 9
		0x00, // BRK
	}
	for i, b := range program {
		if err := bus.Write(0x0200+uint16(i), b); err != nil {
			t.Fatalf("seed RAM: %v", err)
		}
	}

	c := NewCPU6502("cpu", 0x0200)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	for i := 0; i < 64 && !c.Halted(); i++ {
		if err := c.Tick(context.Background(), emulator.Tick{Cycle: uint64(i)}, bus); err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
	}
	if !c.Halted() {
		t.Fatalf("expected CPU to halt on BRK")
	}

	if c.A != 0x08 {
		t.Fatalf("expected A=0x08, got 0x%02X", c.A)
	}
	if c.X != 0x09 {
		t.Fatalf("expected X=0x09, got 0x%02X", c.X)
	}
	v, err := bus.Read(0x0010)
	if err != nil {
		t.Fatalf("read ram: %v", err)
	}
	if v != 0x08 {
		t.Fatalf("expected RAM[0x0010]=0x08, got 0x%02X", v)
	}
}

func TestCPU6502_InstructionResumesOnNextCycle(t *testing.T) {
	bus := emulator.NewBus()
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	program := []byte{0xA9, 0x7F, 0x00} // LDA #$7F ; BRK
	for i, b := range program {
		if err := bus.Write(0x0200+uint16(i), b); err != nil {
			t.Fatalf("seed RAM: %v", err)
		}
	}

	c := NewCPU6502("cpu", 0x0200)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	// Cycle 1: opcode fetch only (LDA), instruction not completed yet.
	if err := c.Tick(context.Background(), emulator.Tick{Cycle: 0}, bus); err != nil {
		t.Fatalf("tick 0: %v", err)
	}
	if c.A != 0x00 {
		t.Fatalf("expected A unchanged after fetch cycle, got 0x%02X", c.A)
	}
	if c.PC != 0x0201 {
		t.Fatalf("expected PC=0x0201 after fetch, got 0x%04X", c.PC)
	}

	// Cycle 2: LDA immediate data cycle, now A is updated.
	if err := c.Tick(context.Background(), emulator.Tick{Cycle: 1}, bus); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	if c.A != 0x7F {
		t.Fatalf("expected A=0x7F after second cycle, got 0x%02X", c.A)
	}
	if c.PC != 0x0202 {
		t.Fatalf("expected PC=0x0202 after operand read, got 0x%04X", c.PC)
	}
}

func TestCPU6502_TraceWriter(t *testing.T) {
	program := []byte{0xA9, 0x42, 0xAA, 0x00}
	c, bus := newTestCPU(t, program)
	writeBytes(t, bus, 0xFFFE, []byte{0x00, 0x40})

	var trace bytes.Buffer
	c.SetTraceWriter(&trace)

	runUntilHalt(t, c, bus, 32)

	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	got := strings.TrimSpace(ansiPattern.ReplaceAllString(trace.String(), ""))
	want := strings.Join([]string{
		"CYC     PC     BYTES     ASM                 REGS FLAGS",
		"0       $0200  A9 42     LDA  #$42           A:00 X:00 Y:00 SP:FD P:24 ..U..I..",
		"2       $0202  AA        TAX                 A:42 X:00 Y:00 SP:FD P:24 ..U..I..",
		"4       $0203  00        BRK                 A:42 X:42 Y:00 SP:FD P:24 ..U..I..",
	}, "\n")

	if got != want {
		t.Fatalf("unexpected trace output:\n%s\nwant:\n%s", got, want)
	}
}

func TestCPU6502_UnsupportedOpcode(t *testing.T) {
	bus := emulator.NewBus()
	ram, err := memory.NewRAM("ram", 0x0000, 0xFFFF)
	if err != nil {
		t.Fatalf("new RAM: %v", err)
	}
	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}

	if err := bus.Write(0x0200, 0x02); err != nil {
		t.Fatalf("seed RAM: %v", err)
	}

	c := NewCPU6502("cpu", 0x0200)
	if err := c.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}

	err = c.Tick(context.Background(), emulator.Tick{Cycle: 0}, bus)
	if err == nil {
		t.Fatalf("expected unsupported opcode error")
	}
}

func TestCPU6502_StackAndSubroutineInstructions(t *testing.T) {
	program := []byte{
		0xA9, 0x11,
		0x48,
		0xA9, 0x22,
		0x20, 0x0A, 0x02,
		0x68,
		0x00,
		0x08,
		0xA9, 0x33,
		0x28,
		0x60,
	}

	c, bus := newTestCPU(t, program)
	writeBytes(t, bus, 0xFFFE, []byte{0x00, 0x40})
	runUntilHalt(t, c, bus, 128)

	if c.A != 0x11 {
		t.Fatalf("expected A=0x11 after PLA, got 0x%02X", c.A)
	}
	if c.SP != 0xFA {
		t.Fatalf("expected SP=0xFA after BRK stack push, got 0x%02X", c.SP)
	}
	if c.PC != 0x4000 {
		t.Fatalf("expected PC=0x4000 after BRK vector load, got 0x%04X", c.PC)
	}
}

func TestCPU6502_IncrementAndDecrementUpdateFlags(t *testing.T) {
	t.Run("INC zero page sets zero flag", func(t *testing.T) {
		program := []byte{0xE6, 0x10, 0x00}
		c, bus := newTestCPU(t, program)
		writeBytes(t, bus, 0x0010, []byte{0xFF})

		runUntilHalt(t, c, bus, 32)

		value, err := bus.Read(0x0010)
		if err != nil {
			t.Fatalf("read zero page: %v", err)
		}
		if value != 0x00 {
			t.Fatalf("expected incremented value 0x00, got 0x%02X", value)
		}
		if c.getFlag(flagZ) == 0 {
			t.Fatalf("expected zero flag set after INC wraps to zero")
		}
		if c.getFlag(flagN) != 0 {
			t.Fatalf("expected negative flag clear after INC wraps to zero")
		}
	})

	t.Run("DEC zero page sets negative flag", func(t *testing.T) {
		program := []byte{0xC6, 0x10, 0x00}
		c, bus := newTestCPU(t, program)
		writeBytes(t, bus, 0x0010, []byte{0x00})

		runUntilHalt(t, c, bus, 32)

		value, err := bus.Read(0x0010)
		if err != nil {
			t.Fatalf("read zero page: %v", err)
		}
		if value != 0xFF {
			t.Fatalf("expected decremented value 0xFF, got 0x%02X", value)
		}
		if c.getFlag(flagN) == 0 {
			t.Fatalf("expected negative flag set after DEC underflows to 0xFF")
		}
		if c.getFlag(flagZ) != 0 {
			t.Fatalf("expected zero flag clear after DEC underflows to 0xFF")
		}
	})
}

func TestCPU6502_BranchesAndComparisons(t *testing.T) {
	program := []byte{
		0xA2, 0x03,
		0xCA,
		0xE0, 0x02,
		0xF0, 0x02,
		0xA9, 0x00,
		0xA9, 0x7F,
		0xC9, 0x7F,
		0xD0, 0x02,
		0xA9, 0x55,
		0xA9, 0x80,
		0x30, 0x02,
		0xA9, 0x00,
		0xA9, 0x01,
		0x10, 0x02,
		0xA9, 0x00,
		0x00,
	}

	c, bus := newTestCPU(t, program)
	runUntilHalt(t, c, bus, 128)

	if c.X != 0x02 {
		t.Fatalf("expected X=0x02, got 0x%02X", c.X)
	}
	if c.A != 0x01 {
		t.Fatalf("expected A=0x01 after taken branches, got 0x%02X", c.A)
	}
	if c.getFlag(flagZ) != 0 {
		t.Fatalf("expected zero flag cleared by final LDA")
	}
	if c.getFlag(flagN) != 0 {
		t.Fatalf("expected negative flag cleared by final LDA")
	}
}

func TestCPU6502_IndirectAddressingAndStores(t *testing.T) {
	program := []byte{
		0xA2, 0x04,
		0xA0, 0x02,
		0xA9, 0x5A,
		0x81, 0x20,
		0xA9, 0x00,
		0xA1, 0x20,
		0x91, 0x30,
		0xB1, 0x30,
		0x86, 0x40,
		0x84, 0x41,
		0xA6, 0x40,
		0xA4, 0x41,
		0x8E, 0x00, 0x07,
		0x8C, 0x01, 0x07,
		0x00,
	}

	c, bus := newTestCPU(t, program)
	writeBytes(t, bus, 0x0024, []byte{0x00, 0x06})
	writeBytes(t, bus, 0x0030, []byte{0x10, 0x06})
	runUntilHalt(t, c, bus, 256)

	checks := []struct {
		addr uint16
		want byte
	}{
		{addr: 0x0600, want: 0x5A},
		{addr: 0x0612, want: 0x5A},
		{addr: 0x0700, want: 0x04},
		{addr: 0x0701, want: 0x02},
	}
	for _, check := range checks {
		got, err := bus.Read(check.addr)
		if err != nil {
			t.Fatalf("read 0x%04X: %v", check.addr, err)
		}
		if got != check.want {
			t.Fatalf("expected RAM[0x%04X]=0x%02X, got 0x%02X", check.addr, check.want, got)
		}
	}

	if c.A != 0x5A {
		t.Fatalf("expected A=0x5A, got 0x%02X", c.A)
	}
	if c.X != 0x04 {
		t.Fatalf("expected X=0x04, got 0x%02X", c.X)
	}
	if c.Y != 0x02 {
		t.Fatalf("expected Y=0x02, got 0x%02X", c.Y)
	}
}

func TestCPU6502_ShiftsRotatesAndLogic(t *testing.T) {
	program := []byte{
		0xA9, 0x81,
		0x0A,
		0x2A,
		0x4A,
		0x6A,
		0x85, 0x10,
		0xE6, 0x10,
		0xC6, 0x10,
		0x46, 0x10,
		0x26, 0x10,
		0xA9, 0xF0,
		0x29, 0x0F,
		0x09, 0x80,
		0x49, 0xFF,
		0x18,
		0xA9, 0x10,
		0x69, 0x05,
		0x38,
		0xE9, 0x03,
		0x00,
	}

	c, bus := newTestCPU(t, program)
	runUntilHalt(t, c, bus, 256)

	mem, err := bus.Read(0x0010)
	if err != nil {
		t.Fatalf("read 0x0010: %v", err)
	}
	if mem != 0x81 {
		t.Fatalf("expected RAM[0x0010]=0x81, got 0x%02X", mem)
	}
	if c.A != 0x12 {
		t.Fatalf("expected A=0x12 after ADC/SBC sequence, got 0x%02X", c.A)
	}
	if c.getFlag(flagC) == 0 {
		t.Fatalf("expected carry flag set after final SBC")
	}
}

func TestCPU6502_AbsoluteIndexedReadTiming(t *testing.T) {
	t.Run("no page crossing", func(t *testing.T) {
		c, bus := newTestCPU(t, []byte{0xBD, 0xFE, 0x02, 0x00})
		c.X = 0x01
		writeBytes(t, bus, 0x02FF, []byte{0x44})

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)
		if c.A != 0x00 {
			t.Fatalf("expected A unchanged before final read, got 0x%02X", c.A)
		}

		tickOnce(t, c, bus, 3)
		if c.A != 0x44 {
			t.Fatalf("expected A updated on 4th cycle without page crossing, got 0x%02X", c.A)
		}
	})

	t.Run("with page crossing", func(t *testing.T) {
		c, bus := newTestCPU(t, []byte{0xBD, 0xFF, 0x02, 0x00})
		c.X = 0x01
		writeBytes(t, bus, 0x0300, []byte{0x55})

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)
		tickOnce(t, c, bus, 3)
		if c.A != 0x00 {
			t.Fatalf("expected extra page-cross cycle before load, got A=0x%02X", c.A)
		}

		tickOnce(t, c, bus, 4)
		if c.A != 0x55 {
			t.Fatalf("expected A updated one cycle later on page crossing, got 0x%02X", c.A)
		}
	})
}

func TestCPU6502_IndirectYReadTiming(t *testing.T) {
	c, bus := newTestCPU(t, []byte{0xB1, 0x10, 0x00})
	c.Y = 0x01
	writeBytes(t, bus, 0x0010, []byte{0xFF, 0x02})
	writeBytes(t, bus, 0x0300, []byte{0x66})

	tickOnce(t, c, bus, 0)
	tickOnce(t, c, bus, 1)
	tickOnce(t, c, bus, 2)
	tickOnce(t, c, bus, 3)
	tickOnce(t, c, bus, 4)
	if c.A != 0x00 {
		t.Fatalf("expected indirect,Y page crossing to delay the load, got A=0x%02X", c.A)
	}

	tickOnce(t, c, bus, 5)
	if c.A != 0x66 {
		t.Fatalf("expected A updated after the extra indirect,Y cycle, got 0x%02X", c.A)
	}
}

func TestCPU6502_BranchTiming(t *testing.T) {
	t.Run("branch not taken", func(t *testing.T) {
		c, bus := newTestCPU(t, []byte{0xD0, 0x02, 0xEA, 0x00})
		c.setFlag(flagZ, true)

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		if c.PC != 0x0202 {
			t.Fatalf("expected PC=0x0202 after branch not taken, got 0x%04X", c.PC)
		}

		tickOnce(t, c, bus, 2)
		if c.PC != 0x0203 {
			t.Fatalf("expected next opcode fetched immediately after untaken branch, got PC=0x%04X", c.PC)
		}
	})

	t.Run("branch taken same page", func(t *testing.T) {
		c, bus := newTestCPU(t, []byte{0xD0, 0x02, 0xEA, 0xEA, 0x00})
		c.setFlag(flagZ, false)

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		if c.PC != 0x0202 {
			t.Fatalf("expected operand fetch to leave PC at 0x0202, got 0x%04X", c.PC)
		}

		tickOnce(t, c, bus, 2)
		if c.PC != 0x0204 {
			t.Fatalf("expected taken branch target on third cycle, got 0x%04X", c.PC)
		}

		tickOnce(t, c, bus, 3)
		if c.PC != 0x0205 {
			t.Fatalf("expected next fetch right after same-page taken branch, got PC=0x%04X", c.PC)
		}
	})

	t.Run("branch taken with page crossing", func(t *testing.T) {
		c, bus := newTestCPUAt(t, 0x02FD, []byte{0xD0, 0x02, 0xEA, 0xEA, 0x00})
		c.setFlag(flagZ, false)

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)
		if c.PC != 0x0301 {
			t.Fatalf("expected crossed-page target selected before final timing cycle, got 0x%04X", c.PC)
		}

		tickOnce(t, c, bus, 3)
		if c.PC != 0x0301 {
			t.Fatalf("expected extra cycle to preserve PC before fetch, got 0x%04X", c.PC)
		}

		tickOnce(t, c, bus, 4)
		if c.PC != 0x0302 {
			t.Fatalf("expected fetch only after crossed-page branch extra cycle, got PC=0x%04X", c.PC)
		}
	})
}

func TestCPU6502_ReadModifyWriteBusCycles(t *testing.T) {
	t.Run("zero page writes old then new value", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0x06, 0x10, 0x00})
		trace.data[0x0010] = 0x81

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)
		if trace.writes[0x0010] != 0 {
			t.Fatalf("expected no write before dummy write cycle, got %d", trace.writes[0x0010])
		}

		tickOnce(t, c, bus, 3)
		if trace.writes[0x0010] != 1 {
			t.Fatalf("expected one dummy write on cycle 4, got %d", trace.writes[0x0010])
		}
		if trace.data[0x0010] != 0x81 {
			t.Fatalf("expected dummy write to preserve old value 0x81, got 0x%02X", trace.data[0x0010])
		}

		tickOnce(t, c, bus, 4)
		if trace.writes[0x0010] != 2 {
			t.Fatalf("expected final write after dummy write, got %d total writes", trace.writes[0x0010])
		}
		if trace.data[0x0010] != 0x02 {
			t.Fatalf("expected final shifted value 0x02, got 0x%02X", trace.data[0x0010])
		}
	})

	t.Run("absolute X performs dummy indexed read and two writes", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0x1E, 0xFF, 0x12, 0x00})
		c.X = 0x01
		trace.data[0x1300] = 0x40

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)
		tickOnce(t, c, bus, 3)
		if trace.reads[0x1200] != 1 {
			t.Fatalf("expected one dummy indexed read at 0x1200, got %d", trace.reads[0x1200])
		}
		if trace.reads[0x1300] != 0 {
			t.Fatalf("expected no real operand read before next cycle, got %d", trace.reads[0x1300])
		}

		tickOnce(t, c, bus, 4)
		if trace.reads[0x1300] != 1 {
			t.Fatalf("expected operand read on following cycle, got %d", trace.reads[0x1300])
		}

		tickOnce(t, c, bus, 5)
		if trace.writes[0x1300] != 1 {
			t.Fatalf("expected dummy write of old value, got %d", trace.writes[0x1300])
		}
		if trace.data[0x1300] != 0x40 {
			t.Fatalf("expected dummy write to keep 0x40, got 0x%02X", trace.data[0x1300])
		}

		tickOnce(t, c, bus, 6)
		if trace.writes[0x1300] != 2 {
			t.Fatalf("expected final write after dummy write, got %d", trace.writes[0x1300])
		}
		if trace.data[0x1300] != 0x80 {
			t.Fatalf("expected final shifted value 0x80, got 0x%02X", trace.data[0x1300])
		}
	})
}

func TestCPU6502_StackAndSubroutineBusSequences(t *testing.T) {
	t.Run("PHA", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0x48, 0x00})
		c.A = 0x42

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0200, value: 0x48},
			{kind: 'R', addr: 0x0201, value: 0x00},
			{kind: 'W', addr: 0x01FD, value: 0x42},
		})
	})

	t.Run("PHP", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0x08, 0x00})
		c.P = flagI | flagU

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0200, value: 0x08},
			{kind: 'R', addr: 0x0201, value: 0x00},
			{kind: 'W', addr: 0x01FD, value: flagI | flagU | flagB},
		})
	})

	t.Run("PLA", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0x68, 0x00})
		c.SP = 0xFC
		trace.data[0x01FD] = 0x37

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)
		tickOnce(t, c, bus, 3)

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0200, value: 0x68},
			{kind: 'R', addr: 0x0201, value: 0x00},
			{kind: 'R', addr: 0x01FC, value: 0x00},
			{kind: 'R', addr: 0x01FD, value: 0x37},
		})
	})

	t.Run("PLP", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0x28, 0x00})
		c.SP = 0xFC
		trace.data[0x01FD] = flagN

		tickOnce(t, c, bus, 0)
		tickOnce(t, c, bus, 1)
		tickOnce(t, c, bus, 2)
		tickOnce(t, c, bus, 3)

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0200, value: 0x28},
			{kind: 'R', addr: 0x0201, value: 0x00},
			{kind: 'R', addr: 0x01FC, value: 0x00},
			{kind: 'R', addr: 0x01FD, value: flagN},
		})
	})

	t.Run("JSR", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0x20, 0x34, 0x12})

		for cycle := uint64(0); cycle < 6; cycle++ {
			tickOnce(t, c, bus, cycle)
		}

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0200, value: 0x20},
			{kind: 'R', addr: 0x0201, value: 0x34},
			{kind: 'R', addr: 0x01FD, value: 0x00},
			{kind: 'W', addr: 0x01FD, value: 0x02},
			{kind: 'W', addr: 0x01FC, value: 0x02},
			{kind: 'R', addr: 0x0202, value: 0x12},
		})
	})

	t.Run("RTI", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0300, []byte{0x40, 0x00})
		c.SP = 0xFA
		trace.data[0x01FB] = flagZ
		trace.data[0x01FC] = 0x34
		trace.data[0x01FD] = 0x12

		for cycle := uint64(0); cycle < 6; cycle++ {
			tickOnce(t, c, bus, cycle)
		}

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0300, value: 0x40},
			{kind: 'R', addr: 0x0301, value: 0x00},
			{kind: 'R', addr: 0x01FA, value: 0x00},
			{kind: 'R', addr: 0x01FB, value: flagZ},
			{kind: 'R', addr: 0x01FC, value: 0x34},
			{kind: 'R', addr: 0x01FD, value: 0x12},
		})
	})

	t.Run("RTS", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0300, []byte{0x60, 0x00})
		c.SP = 0xFB
		trace.data[0x01FC] = 0x02
		trace.data[0x01FD] = 0x02
		trace.data[0x0203] = 0xEA

		for cycle := uint64(0); cycle < 6; cycle++ {
			tickOnce(t, c, bus, cycle)
		}

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0300, value: 0x60},
			{kind: 'R', addr: 0x0301, value: 0x00},
			{kind: 'R', addr: 0x01FB, value: 0x00},
			{kind: 'R', addr: 0x01FC, value: 0x02},
			{kind: 'R', addr: 0x01FD, value: 0x02},
			{kind: 'R', addr: 0x0203, value: 0xEA},
		})

		countBefore := len(trace.log)
		tickOnce(t, c, bus, 6)
		if len(trace.log) != countBefore {
			t.Fatalf("expected prefetched RTS opcode decode without extra bus access")
		}
		if c.PC != 0x0204 {
			t.Fatalf("expected PC=0x0204 after RTS prefetch, got 0x%04X", c.PC)
		}
	})
}

func TestCPU6502_InterruptBusSequences(t *testing.T) {
	t.Run("BRK performs full interrupt sequence then halts", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0x00, 0xEA})
		trace.data[0xFFFE] = 0x34
		trace.data[0xFFFF] = 0x12

		for cycle := uint64(0); cycle < 7; cycle++ {
			tickOnce(t, c, bus, cycle)
		}

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0200, value: 0x00},
			{kind: 'R', addr: 0x0201, value: 0xEA},
			{kind: 'W', addr: 0x01FD, value: 0x02},
			{kind: 'W', addr: 0x01FC, value: 0x02},
			{kind: 'W', addr: 0x01FB, value: flagI | flagU | flagB},
			{kind: 'R', addr: 0xFFFE, value: 0x34},
			{kind: 'R', addr: 0xFFFF, value: 0x12},
		})
		if !c.Halted() {
			t.Fatalf("expected BRK to halt in compatibility mode")
		}
		if c.PC != 0x1234 {
			t.Fatalf("expected BRK to load vector PC=0x1234 before halting, got 0x%04X", c.PC)
		}
	})

	t.Run("IRQ uses dummy opcode fetch then vector", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0xEA})
		c.setFlag(flagI, false)
		c.RequestIRQ()
		trace.data[0xFFFE] = 0x78
		trace.data[0xFFFF] = 0x56

		for cycle := uint64(0); cycle < 6; cycle++ {
			tickOnce(t, c, bus, cycle)
		}

		assertTrace(t, trace.log, []traceAccess{
			{kind: 'R', addr: 0x0200, value: 0xEA},
			{kind: 'W', addr: 0x01FD, value: 0x02},
			{kind: 'W', addr: 0x01FC, value: 0x00},
			{kind: 'W', addr: 0x01FB, value: flagU},
			{kind: 'R', addr: 0xFFFE, value: 0x78},
			{kind: 'R', addr: 0xFFFF, value: 0x56},
		})
		if c.PC != 0x5678 {
			t.Fatalf("expected IRQ vector PC=0x5678, got 0x%04X", c.PC)
		}
		if c.getFlag(flagI) == 0 {
			t.Fatalf("expected IRQ sequence to set I flag")
		}
	})

	t.Run("NMI has priority over IRQ", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0200, []byte{0xEA})
		c.setFlag(flagI, false)
		c.RequestIRQ()
		c.RequestNMI()
		trace.data[0xFFFA] = 0xCD
		trace.data[0xFFFB] = 0xAB

		for cycle := uint64(0); cycle < 6; cycle++ {
			tickOnce(t, c, bus, cycle)
		}

		if trace.reads[0xFFFA] != 1 || trace.reads[0xFFFB] != 1 {
			t.Fatalf("expected NMI vector reads, got low=%d high=%d", trace.reads[0xFFFA], trace.reads[0xFFFB])
		}
		if trace.reads[0xFFFE] != 0 || trace.reads[0xFFFF] != 0 {
			t.Fatalf("expected IRQ vector not to be used while NMI is pending")
		}
		if c.PC != 0xABCD {
			t.Fatalf("expected NMI vector PC=0xABCD, got 0x%04X", c.PC)
		}
	})

	t.Run("IRQ consumes RTS prefetch without extra bus read", func(t *testing.T) {
		c, bus, trace := newTraceCPUAt(t, 0x0300, []byte{0x60, 0x00})
		c.SP = 0xFB
		trace.data[0x01FC] = 0x02
		trace.data[0x01FD] = 0x02
		trace.data[0x0203] = 0xEA
		trace.data[0xFFFE] = 0x00
		trace.data[0xFFFF] = 0x40

		for cycle := uint64(0); cycle < 6; cycle++ {
			tickOnce(t, c, bus, cycle)
		}
		countBefore := len(trace.log)
		c.setFlag(flagI, false)
		c.RequestIRQ()

		tickOnce(t, c, bus, 6)
		if len(trace.log) != countBefore {
			t.Fatalf("expected pending IRQ to consume RTS prefetch without new bus access")
		}

		tickOnce(t, c, bus, 7)
		assertTrace(t, trace.log[countBefore-1:], []traceAccess{
			{kind: 'R', addr: 0x0203, value: 0xEA},
			{kind: 'W', addr: 0x01FD, value: 0x02},
		})
	})
}
