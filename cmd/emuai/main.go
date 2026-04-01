package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Djoulzy/emuai/internal/components/cpu"
	"github.com/Djoulzy/emuai/internal/components/memory"
	"github.com/Djoulzy/emuai/internal/components/peripheral"
	"github.com/Djoulzy/emuai/internal/components/sound"
	"github.com/Djoulzy/emuai/internal/components/video"
	romconfig "github.com/Djoulzy/emuai/internal/config"
	"github.com/Djoulzy/emuai/internal/emulator"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	motherboardFrequencyHz    = 1_000_000
	pausePollInterval         = 25 * time.Millisecond
	pauseDumpInstructionCount = 16
	startupScreenColumns      = 40
	startupScreenRows         = 24
	appleIIeCharacterROMName  = "3410036.bin"
	defaultROMConfigName      = "apple2-roms.yaml"
	appleIIeTextPageSize      = 0x0400
	appleIIeTextPage1Address  = 0x0400
	resetVectorAddress        = 0xFFFC
)

func selectTraceWriter(traceOverlay *video.TraceOverlay, fallback io.Writer) io.Writer {
	if traceOverlay != nil {
		return traceOverlay
	}
	return fallback
}

type uint16Flag struct {
	value uint16
	set   bool
}

type runControl struct {
	paused atomic.Bool
}

type memoryLoader interface {
	Load(addr uint16, data []byte) error
}

type memoryFileLoader interface {
	LoadFile(path string, addr uint16) error
}

type slotROMLoader interface {
	LoadSlotROMFile(slot int, path string) error
}

type memoryResetter interface {
	Reset(ctx context.Context, bus *emulator.Bus) error
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
	trace := flag.Bool("trace", false, "enable CPU instruction trace output")
	romConfigPath := flag.String("rom-config", "", "path to a YAML file describing ROM images to load into memory")
	timeout := flag.Duration("timeout", 0, "maximum wall-clock run duration; 0 disables timeout")
	maxCycles := flag.Uint64("max-cycles", 0, "maximum number of motherboard cycles to execute; 0 disables the limit")
	stopPC := &uint16Flag{}
	realtime := flag.Bool("realtime", false, "run using the motherboard clock instead of stepping as fast as possible")
	videoBackend := flag.String("video-backend", string(video.BackendNull), "video backend to use: null or vulkan")
	videoWidth := flag.Int("video-width", 560, "video framebuffer width in pixels")
	videoHeight := flag.Int("video-height", 384, "video framebuffer height in pixels")
	videoRefreshHz := flag.Int("video-refresh-hz", 60, "video refresh rate in Hz")
	flag.Var(stopPC, "stop-pc", "stop execution before the instruction at this program counter executes")
	flag.Parse()

	if *romConfigPath == "" {
		defaultROMConfigPath, err := resolveRepositoryPath(filepath.Join("ROMs", defaultROMConfigName))
		if err != nil {
			log.Fatalf("default ROM config unavailable: %v", err)
		}
		*romConfigPath = defaultROMConfigPath
		log.Printf("using default ROM config %s", defaultROMConfigPath)
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

	mmu, err := memory.NewAppleIIeMMU("main-memory")
	if err != nil {
		log.Fatalf("create Apple IIe MMU: %v", err)
	}

	if err := board.Bus().MapDevice(0x0000, 0xFFFF, "main-memory", mmu); err != nil {
		log.Fatalf("map Apple IIe MMU: %v", err)
	}

	processor := cpu.NewCPU6502("cpu-main")
	processor.SetHaltOnBRK(false)
	var traceOverlay *video.TraceOverlay
	if video.Backend(*videoBackend) == video.BackendVulkan {
		traceOverlay = video.NewTraceOverlay(256)
	}
	if *trace {
		processor.SetTraceWriter(selectTraceWriter(traceOverlay, os.Stdout))
	}
	soundDevice := sound.NewNullSound("sound-main")
	keyboardDevice := peripheral.NewKeyboard("kbd-main")

	characterROM, characterROMPath, err := loadAppleIIeCharacterROM(*romConfigPath)
	if err != nil {
		log.Fatalf("load Apple IIe character ROM: %v", err)
	}

	videoDevice, err := video.NewAppleIIeCRTC("video-main", video.Config{
		Backend: video.Backend(*videoBackend),
		ClockHz: motherboardFrequencyHz,
		CRT: video.CRTConfig{
			Width:     *videoWidth,
			Height:    *videoHeight,
			RefreshHz: *videoRefreshHz,
		},
		Trace:    traceOverlay,
		TraceOn:  *trace,
		Keyboard: keyboardDevice,
	}, video.AppleIIeOptions{
		CharacterROM: characterROM,
		BankedMemory: mmu,
	})
	if err != nil {
		log.Fatalf("create video: %v", err)
	}
	log.Printf("loaded Apple IIe character ROM %s (%s)", characterROMPath, video.DescribeAppleIIeCharacterROM(characterROM))

	if err := mapAppleIIeSoftSwitches(board.Bus(), mmu, videoDevice, soundDevice, keyboardDevice); err != nil {
		log.Fatalf("map Apple IIe soft-switches: %v", err)
	}

	components := []emulator.ClockedComponent{
		mmu,
		processor,
		videoDevice,
		soundDevice,
		keyboardDevice,
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
	stopPauseControls, err := startPauseControls(control, cancel, board.Bus())
	if err != nil {
		log.Printf("pause controls disabled: %v", err)
	} else {
		defer stopPauseControls()
		log.Printf("interactive controls: press space to pause, then use the pause menu for resume, memory dump, or quit; Ctrl+C interrupts")
	}

	if err := resetForBoot(ctx, board.Bus(), mmu, videoDevice, soundDevice, keyboardDevice); err != nil {
		log.Fatalf("prepare machine state: %v", err)
	}

	writeStartupTextToRAM(mmu, startupScreenLines(*videoBackend, *romConfigPath, characterROMPath))

	if *romConfigPath != "" {
		loadedROMs, err := loadROMsFromConfig(mmu, *romConfigPath)
		if err != nil {
			log.Fatalf("load ROM config: %v", err)
		}
		for _, loadedROM := range loadedROMs {
			log.Print(loadedROM)
		}
	}

	if err := processor.Reset(ctx, board.Bus()); err != nil {
		log.Fatalf("reset CPU after loading boot sources: %v", err)
	}

	if err := runMachine(ctx, board, processor, *realtime, *maxCycles, control, stopPC); err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		log.Fatalf("run board: %v", err)
	}

	log.Printf("emulation stopped after %d cycles", board.Cycle())
}

func loadROMsFromConfig(memoryDevice memoryFileLoader, configPath string) ([]string, error) {
	resolvedConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("resolve ROM config path: %w", err)
	}

	set, err := romconfig.LoadROMSet(resolvedConfigPath)
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Dir(resolvedConfigPath)
	loadedROMs := make([]string, 0, len(set.ROMs))
	for idx, rom := range set.ROMs {
		resolvedROMPath := rom.ResolvePath(baseDir)
		if err := memoryDevice.LoadFile(resolvedROMPath, rom.Start.Uint16()); err != nil {
			return nil, fmt.Errorf("load ROM %d (%s): %w", idx, resolvedROMPath, err)
		}

		name := rom.Name
		if name == "" {
			name = filepath.Base(resolvedROMPath)
		}

		loadedROMs = append(loadedROMs, fmt.Sprintf("loaded ROM %s from %s at 0x%04X", name, resolvedROMPath, rom.Start.Uint16()))
	}

	configuredSlots := set.ConfiguredSlots()
	if len(configuredSlots) == 0 {
		return loadedROMs, nil
	}

	slotLoader, ok := memoryDevice.(slotROMLoader)
	if !ok {
		return nil, fmt.Errorf("configured slot ROMs require a memory device with slot ROM support")
	}

	for _, slot := range configuredSlots {
		resolvedSlotROMPath := set.ResolveSlotROMPath(slot, baseDir)
		if err := slotLoader.LoadSlotROMFile(slot, resolvedSlotROMPath); err != nil {
			return nil, fmt.Errorf("load slot %d ROM (%s): %w", slot, resolvedSlotROMPath, err)
		}
		loadedROMs = append(loadedROMs, fmt.Sprintf("loaded slot %d ROM from %s", slot, resolvedSlotROMPath))
	}

	return loadedROMs, nil
}

func resetForBoot(ctx context.Context, bus *emulator.Bus, memoryDevice memoryResetter, videoDevice *video.AppleIIeCRTC, soundDevice *sound.NullSound, keyboardDevice *peripheral.Keyboard) error {
	if err := memoryDevice.Reset(ctx, bus); err != nil {
		return fmt.Errorf("reset memory: %w", err)
	}
	if err := videoDevice.Reset(ctx, bus); err != nil {
		return fmt.Errorf("reset video: %w", err)
	}
	if err := soundDevice.Reset(ctx, bus); err != nil {
		return fmt.Errorf("reset sound: %w", err)
	}
	if err := keyboardDevice.Reset(ctx, bus); err != nil {
		return fmt.Errorf("reset keyboard: %w", err)
	}
	return nil
}

func mapAppleIIeSoftSwitches(bus *emulator.Bus, memoryDevice emulator.AddressableDevice, videoDevice *video.AppleIIeCRTC, soundDevice *sound.NullSound, keyboardDevice *peripheral.Keyboard) error {
	if bus == nil {
		return fmt.Errorf("bus is nil")
	}

	softSwitches := peripheral.NewAppleIIeSoftSwitches(keyboardDevice, videoDevice, soundDevice, memoryDevice)
	var slot3Device emulator.AddressableDevice
	if auxMemory, ok := memoryDevice.(interface {
		ArmAuxSlotAccess()
		ClearAuxSlotAccess()
	}); ok {
		slot3Device = peripheral.NewAppleIIe80ColumnCard("apple2e-slot3-80col", auxMemory)
	}

	mappings := []struct {
		start  uint16
		end    uint16
		name   string
		device emulator.AddressableDevice
	}{
		{start: 0xC000, end: 0xC01F, name: "apple2e-softswitches-low", device: softSwitches},
		{start: 0xC030, end: 0xC03F, name: "apple2e-softswitches-speaker", device: softSwitches},
		{start: 0xC050, end: 0xC05F, name: "apple2e-softswitches-video", device: softSwitches},
		{start: 0xC0B0, end: 0xC0BF, name: "apple2e-slot3-80col", device: slot3Device},
	}

	for _, mapping := range mappings {
		if mapping.device == nil {
			continue
		}
		if err := bus.MapDevice(mapping.start, mapping.end, mapping.name, mapping.device); err != nil {
			return fmt.Errorf("map %s: %w", mapping.name, err)
		}
	}

	return nil
}

func loadAppleIIeCharacterROM(romConfigPath string) ([]byte, string, error) {
	resolvedPath, err := resolveAppleIIeCharacterROMPath(romConfigPath)
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", resolvedPath, err)
	}

	if len(data) < 256*8 {
		return nil, "", fmt.Errorf("character ROM %s is too small: got %d bytes", resolvedPath, len(data))
	}

	return data, resolvedPath, nil
}

func resolveAppleIIeCharacterROMPath(romConfigPath string) (string, error) {
	if strings.TrimSpace(romConfigPath) != "" {
		resolvedConfigPath, err := filepath.Abs(romConfigPath)
		if err != nil {
			return "", fmt.Errorf("resolve ROM config path for character ROM: %w", err)
		}

		set, err := romconfig.LoadROMSet(resolvedConfigPath)
		if err != nil {
			return "", fmt.Errorf("load ROM config for character ROM: %w", err)
		}

		if chargenPath := set.ResolveChargenPath(filepath.Dir(resolvedConfigPath)); chargenPath != "" {
			return chargenPath, nil
		}
	}

	return resolveDefaultAppleIIeCharacterROMPath()
}

func resolveDefaultAppleIIeCharacterROMPath() (string, error) {
	candidates := []string{
		filepath.Join("ROMs", "Apple2", appleIIeCharacterROMName),
		filepath.Join("ROMs", appleIIeCharacterROMName),
	}

	for _, candidate := range candidates {
		resolvedPath, err := resolveRepositoryPath(candidate)
		if err == nil {
			return resolvedPath, nil
		}
	}

	return "", fmt.Errorf("could not locate default Apple IIe character ROM in %s", strings.Join(candidates, ", "))
}

func resolveRepositoryPath(relativePath string) (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	for dir := workingDir; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, relativePath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return "", fmt.Errorf("could not locate %s from %s", relativePath, workingDir)
}

func startupScreenLines(backend string, romConfigPath string, characterROMPath string) []string {
	source := "DEFAULT ROM SET"
	if romConfigPath != "" {
		source = strings.ToUpper(filepath.Base(romConfigPath))
	}

	characterROMName := strings.ToUpper(filepath.Base(characterROMPath))
	if characterROMName == "" {
		characterROMName = strings.ToUpper(appleIIeCharacterROMName)
	}

	return []string{
		"EMUAI APPLE IIE ROM BOOT",
		"",
		fmt.Sprintf("VIDEO BACKEND : %s", strings.ToUpper(backend)),
		fmt.Sprintf("CHAR ROM      : %s", characterROMName),
		fmt.Sprintf("ROM CONFIG    : %s", source),
		"CPU RESET AFTER ROM LOAD",
		"",
		"BOOT SCREEN READY.",
		"PRESS SPACE FOR THE PAUSE MENU.",
		"",
		"TEXT PAGE 1  40 COLUMNS",
	}
}

func writeStartupTextToRAM(memoryDevice memoryLoader, lines []string) {
	if memoryDevice == nil {
		return
	}

	page := make([]byte, appleIIeTextPageSize)
	for i := range page {
		page[i] = encodeAppleIIeTextByte(' ')
	}

	for row, line := range lines {
		if row >= startupScreenRows {
			break
		}
		writeStartupTextLine(page, row, line)
	}

	if err := memoryDevice.Load(appleIIeTextPage1Address, page); err != nil {
		log.Printf("startup text page load warning: %v", err)
	}
}

func writeResetVector(memoryDevice memoryLoader, addr uint16) error {
	if memoryDevice == nil {
		return fmt.Errorf("memory device is nil")
	}

	vector := []byte{byte(addr & 0x00FF), byte(addr >> 8)}
	return memoryDevice.Load(resetVectorAddress, vector)
}

func writeStartupTextLine(page []byte, row int, text string) {
	if row < 0 || row >= startupScreenRows {
		return
	}

	for col := 0; col < startupScreenColumns; col++ {
		glyph := byte(' ')
		if col < len(text) {
			glyph = text[col]
		}
		page[startupTextOffset(row, col)] = encodeAppleIIeTextByte(glyph)
	}
}

func startupTextOffset(row, col int) int {
	return ((row & 0x07) << 7) + ((row >> 3) * 0x28) + col
}

func encodeAppleIIeTextByte(value byte) byte {
	if value >= 'a' && value <= 'z' {
		value -= 'a' - 'A'
	}
	if value < 0x20 || value > 0x5F {
		value = '?'
	}
	return value | 0x80
}

func runMachine(ctx context.Context, board *emulator.Motherboard, processor *cpu.CPU6502, realtime bool, maxCycles uint64, control *runControl, stopPC *uint16Flag) error {
	var ticker *time.Ticker
	if realtime {
		ticker = time.NewTicker(time.Second / time.Duration(motherboardFrequencyHz))
		defer ticker.Stop()
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if stopPC != nil && stopPC.set && processor.ReadyForInstruction() && processor.PC == stopPC.value {
			log.Printf("stopped at PC 0x%04X after %d cycles", stopPC.value, board.Cycle())
			return nil
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

func startPauseControls(control *runControl, quit func(), bus *emulator.Bus) (func(), error) {
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
	configured.Cc[unix.VMIN] = 0
	configured.Cc[unix.VTIME] = 1
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &configured); err != nil {
		_ = tty.Close()
		return nil, fmt.Errorf("configure pause controls: %w", err)
	}

	doneCh := make(chan struct{})
	stopCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		buffer := make([]byte, 1)
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			n, err := tty.Read(buffer)
			if err != nil {
				return
			}
			if n == 0 {
				continue
			}

			switch processControlKey(control, quit, buffer[0]) {
			case "pause":
				log.Printf("execution paused")
				if err := runPauseMenu(tty, control, quit, bus); err != nil {
					_, _ = fmt.Fprintf(tty, "\npause menu error: %v\n", err)
					log.Printf("pause menu error: %v", err)
				}
			case "resume":
				log.Printf("execution resumed")
			case "quit":
				log.Printf("execution stopping")
				return
			}
		}
	}()

	return func() {
		close(stopCh)
		_ = term.Restore(fd, state)
		_ = tty.Close()
		<-doneCh
	}, nil
}

func runPauseMenu(tty io.ReadWriter, control *runControl, quit func(), bus *emulator.Bus) error {
	for control != nil && control.Paused() {
		if _, err := fmt.Fprint(tty, "\nPause menu\n  r: resume\n  m: dump memory\n  q: quit\nchoice> "); err != nil {
			return err
		}

		choice, err := readMenuChoice(tty)
		if err != nil {
			return err
		}

		switch choice {
		case 'r', 'R':
			control.SetPaused(false)
			_, _ = fmt.Fprintln(tty, "resuming execution")
			log.Printf("execution resumed")
			return nil
		case 'm', 'M':
			startAddr, err := promptMemoryDumpAddress(tty)
			if err != nil {
				_, _ = fmt.Fprintf(tty, "invalid address: %v\n", err)
				continue
			}

			if err := dumpMemoryBlock(tty, bus, startAddr, pauseDumpInstructionCount); err != nil {
				_, _ = fmt.Fprintf(tty, "dump failed: %v\n", err)
				continue
			}
		case 'q', 'Q':
			_, _ = fmt.Fprintln(tty, "stopping execution")
			if quit != nil {
				quit()
			}
			log.Printf("execution stopping")
			return nil
		default:
			_, _ = fmt.Fprintf(tty, "unknown option %q\n", string(choice))
		}
	}

	return nil
}

func readMenuChoice(tty io.Reader) (byte, error) {
	buffer := make([]byte, 1)
	for {
		if _, err := tty.Read(buffer); err != nil {
			return 0, err
		}

		switch buffer[0] {
		case '\r', '\n':
			continue
		default:
			return buffer[0], nil
		}
	}
}

func promptMemoryDumpAddress(tty io.ReadWriter) (uint16, error) {
	line, err := readInteractiveLine(tty, "start address (hex, e.g. 0xD000 or $D000)> ")
	if err != nil {
		return 0, err
	}

	return parseHexAddress(line)
}

func readInteractiveLine(tty io.ReadWriter, prompt string) (string, error) {
	if _, err := fmt.Fprint(tty, prompt); err != nil {
		return "", err
	}

	buffer := make([]byte, 1)
	var builder strings.Builder
	for {
		if _, err := tty.Read(buffer); err != nil {
			return "", err
		}

		switch buffer[0] {
		case '\r', '\n':
			_, _ = fmt.Fprint(tty, "\n")
			return strings.TrimSpace(builder.String()), nil
		case 0x7F, 0x08:
			if builder.Len() == 0 {
				continue
			}
			current := builder.String()
			builder.Reset()
			builder.WriteString(current[:len(current)-1])
			_, _ = fmt.Fprint(tty, "\b \b")
		default:
			if buffer[0] < 0x20 || buffer[0] > 0x7E {
				continue
			}
			builder.WriteByte(buffer[0])
			_, _ = tty.Write(buffer)
		}
	}
}

func parseHexAddress(raw string) (uint16, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("address is required")
	}

	if strings.HasPrefix(value, "$") {
		value = "0x" + value[1:]
	} else if !strings.HasPrefix(value, "0x") && !strings.HasPrefix(value, "0X") {
		value = "0x" + value
	}

	parsed, err := strconv.ParseUint(value, 0, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid 16-bit address %q: %w", raw, err)
	}

	return uint16(parsed), nil
}

func dumpMemoryBlock(w io.Writer, bus *emulator.Bus, start uint16, instructionCount int) error {
	lines, err := cpu.DisassembleBlock(bus, start, instructionCount)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "\n%s\n", cpu.FormatDisassembly(lines))
	return err
}
