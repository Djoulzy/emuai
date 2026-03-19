package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Djoulzy/emuai/internal/components/cpu"
	"github.com/Djoulzy/emuai/internal/components/memory"
	"github.com/Djoulzy/emuai/internal/components/peripheral"
	"github.com/Djoulzy/emuai/internal/components/sound"
	"github.com/Djoulzy/emuai/internal/components/video"
	"github.com/Djoulzy/emuai/internal/emulator"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	motherboardFrequencyHz = 1_000_000
	pausePollInterval      = 25 * time.Millisecond
)

type uint16Flag struct {
	value uint16
	set   bool
}

type runControl struct {
	paused atomic.Bool
}

func (c *runControl) Paused() bool {
	if c == nil {
		return false
	}
	return c.paused.Load()
}

func (c *runControl) SetPaused(paused bool) {
	if c == nil {
		return
	}
	c.paused.Store(paused)
}

func (c *runControl) TogglePaused() bool {
	if c == nil {
		return false
	}

	paused := !c.paused.Load()
	c.paused.Store(paused)
	return paused
}

func processControlKey(control *runControl, quit func(), key byte) string {
	switch key {
	case ' ':
		if control.TogglePaused() {
			return "pause"
		}
		return "resume"
	case 'q', 'Q':
		if quit != nil {
			quit()
		}
		return "quit"
	default:
		return ""
	}
}

func (f *uint16Flag) String() string {
	return fmt.Sprintf("0x%04X", f.value)
}

func (f *uint16Flag) Set(raw string) error {
	v, err := strconv.ParseUint(raw, 0, 16)
	if err != nil {
		return fmt.Errorf("invalid 16-bit address %q: %w", raw, err)
	}

	f.value = uint16(v)
	f.set = true
	return nil
}

func main() {
	trace := flag.Bool("trace", false, "print each instruction as the CPU executes it")
	binaryPath := flag.String("bin", "", "path to a .bin file to load into RAM before execution")
	timeout := flag.Duration("timeout", 0, "maximum wall-clock run duration; 0 disables timeout")
	maxCycles := flag.Uint64("max-cycles", 0, "maximum number of motherboard cycles to execute; 0 disables the limit")
	realtime := flag.Bool("realtime", false, "run using the motherboard clock instead of stepping as fast as possible")
	loadAddr := &uint16Flag{value: 0x0200}
	pcAddr := &uint16Flag{}
	flag.Var(loadAddr, "load-addr", "RAM address where the binary is loaded")
	flag.Var(pcAddr, "pc", "CPU program counter start address; defaults to -load-addr")
	flag.Parse()

	entryPoint := loadAddr.value
	if pcAddr.set {
		entryPoint = pcAddr.value
	}

	board, err := emulator.NewMotherboard(emulator.Config{FrequencyHz: motherboardFrequencyHz})
	if err != nil {
		log.Fatalf("create motherboard: %v", err)
	}
	defer func() {
		if err := board.Close(); err != nil {
			log.Printf("close warning: %v", err)
		}
	}()

	ram, err := memory.NewRAM("main-ram", 0x0000, 0xFFFF)
	if err != nil {
		log.Fatalf("create RAM: %v", err)
	}

	if err := board.Bus().MapDevice(0x0000, 0xFFFF, "main-ram", ram); err != nil {
		log.Fatalf("map RAM: %v", err)
	}

	processor := cpu.NewCPU6502("cpu-main", entryPoint)
	processor.SetHaltOnBRK(*binaryPath == "")
	if *trace {
		processor.SetTraceWriter(os.Stdout)
	}

	components := []emulator.ClockedComponent{
		ram,
		processor,
		video.NewNullVideo("video-main"),
		sound.NewNullSound("sound-main"),
		peripheral.NewKeyboard("kbd-main"),
	}

	for _, c := range components {
		if err := board.AddComponent(c); err != nil {
			log.Fatalf("add component %s: %v", c.Name(), err)
		}
	}

	baseCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()

	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	if *timeout > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, *timeout)
		defer timeoutCancel()
	}

	control := &runControl{}
	stopPauseControls, err := startPauseControls(control, cancel)
	if err != nil {
		log.Printf("pause controls disabled: %v", err)
	} else {
		defer stopPauseControls()
		log.Printf("interactive controls: press space to pause/resume, q to quit, Ctrl+C to interrupt")
	}

	if err := board.Reset(ctx); err != nil {
		log.Fatalf("reset board: %v", err)
	}

	if *binaryPath != "" {
		resolvedPath, err := filepath.Abs(*binaryPath)
		if err != nil {
			log.Fatalf("resolve binary path: %v", err)
		}
		if err := ram.LoadFile(resolvedPath, loadAddr.value); err != nil {
			log.Fatalf("load binary: %v", err)
		}
		log.Printf("loaded binary %s at 0x%04X, entry point 0x%04X", resolvedPath, loadAddr.value, entryPoint)
	} else {
		program := []byte{
			0xA9, 0x42, // LDA #$42
			0x8D, 0x00, 0x10, // STA $1000
			0xAA, // TAX
			0xE8, // INX
			0xEA, // NOP
			0x00, // BRK
		}

		if err := ram.Load(loadAddr.value, program); err != nil {
			log.Fatalf("seed RAM: %v", err)
		}
		log.Printf("loaded built-in demo at 0x%04X, entry point 0x%04X", loadAddr.value, entryPoint)
	}

	if err := runMachine(ctx, board, processor, *realtime, *maxCycles, control); err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		log.Fatalf("run board: %v", err)
	}

	log.Printf("emulation stopped after %d cycles", board.Cycle())
}

func runMachine(ctx context.Context, board *emulator.Motherboard, processor *cpu.CPU6502, realtime bool, maxCycles uint64, control *runControl) error {
	var ticker *time.Ticker
	if realtime {
		ticker = time.NewTicker(time.Second / time.Duration(motherboardFrequencyHz))
		defer ticker.Stop()
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if processor.Halted() {
			return nil
		}

		if maxCycles > 0 && board.Cycle() >= maxCycles {
			return nil
		}

		if err := waitWhilePaused(ctx, control); err != nil {
			return err
		}

		if ticker != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}

		if err := board.Step(ctx); err != nil {
			return err
		}
	}
}

func waitWhilePaused(ctx context.Context, control *runControl) error {
	for control != nil && control.Paused() {
		if err := ctx.Err(); err != nil {
			return err
		}
		time.Sleep(pausePollInterval)
	}

	return nil
}

func startPauseControls(control *runControl, quit func()) (func(), error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return func() {}, nil
	}

	fd := int(tty.Fd())
	if !term.IsTerminal(fd) {
		_ = tty.Close()
		return func() {}, nil
	}

	state, err := term.GetState(fd)
	if err != nil {
		_ = tty.Close()
		return nil, fmt.Errorf("snapshot terminal state: %w", err)
	}

	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		_ = tty.Close()
		return nil, fmt.Errorf("read terminal state: %w", err)
	}

	configured := *termios
	configured.Iflag |= unix.ICRNL | unix.IXON
	configured.Oflag |= unix.OPOST | unix.ONLCR
	configured.Lflag |= unix.ISIG
	configured.Lflag &^= unix.ICANON | unix.ECHO
	configured.Cc[unix.VMIN] = 1
	configured.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &configured); err != nil {
		_ = tty.Close()
		return nil, fmt.Errorf("configure pause controls: %w", err)
	}

	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		buffer := make([]byte, 1)
		for {
			_, err := tty.Read(buffer)
			if err != nil {
				return
			}

			switch processControlKey(control, quit, buffer[0]) {
			case "pause":
				log.Printf("execution paused")
			case "resume":
				log.Printf("execution resumed")
			case "quit":
				log.Printf("execution stopping")
				return
			}
		}
	}()

	return func() {
		_ = term.Restore(fd, state)
		_ = tty.Close()
		<-doneCh
	}, nil
}
