package peripheral

import (
	"context"
	"sync"
	"unicode"

	"github.com/Djoulzy/emuai/internal/emulator"
)

const (
	appleIIKeyboardDataAddress   uint16 = 0xC000
	appleIIKeyboardStrobeAddress uint16 = 0xC010
)

type Keyboard struct {
	mu     sync.Mutex
	name   string
	latch  byte
	strobe bool
	queue  []byte
}

func NewKeyboard(name string) *Keyboard {
	return &Keyboard{name: name}
}

func (k *Keyboard) Name() string { return k.name }

func (k *Keyboard) Reset(_ context.Context, _ *emulator.Bus) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.latch = 0
	k.strobe = false
	k.queue = k.queue[:0]
	return nil
}

func (k *Keyboard) Tick(_ context.Context, _ emulator.Tick, _ *emulator.Bus) error { return nil }

func (k *Keyboard) Read(addr uint16) (byte, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	switch addr {
	case appleIIKeyboardDataAddress:
		return k.readLatchedKeyLocked(), nil
	case appleIIKeyboardStrobeAddress:
		value := k.readLatchedKeyLocked()
		k.clearStrobeLocked()
		return value, nil
	default:
		return 0, nil
	}
}

func (k *Keyboard) Write(addr uint16, _ byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if addr == appleIIKeyboardStrobeAddress {
		k.clearStrobeLocked()
	}
	return nil
}

func (k *Keyboard) HandleKeyEvent(event emulator.KeyEvent) {
	if event.Action != emulator.KeyActionPress && event.Action != emulator.KeyActionRepeat {
		return
	}

	translated, ok := translateHostKey(event)
	if !ok {
		return
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	k.queue = append(k.queue, translated)
	k.promoteNextLocked()
}

func (k *Keyboard) Close() error { return nil }

func (k *Keyboard) readLatchedKeyLocked() byte {
	value := k.latch & 0x7F
	if k.strobe {
		value |= 0x80
	}
	return value
}

func (k *Keyboard) clearStrobeLocked() {
	k.strobe = false
	k.promoteNextLocked()
}

func (k *Keyboard) promoteNextLocked() {
	if k.strobe || len(k.queue) == 0 {
		return
	}

	k.latch = k.queue[0] & 0x7F
	k.queue = k.queue[1:]
	k.strobe = true
}

func translateHostKey(event emulator.KeyEvent) (byte, bool) {
	if event.Rune != 0 {
		return translatePrintableRune(event.Rune)
	}

	switch event.Code {
	case emulator.KeyCodeEnter:
		return 0x0D, true
	case emulator.KeyCodeEscape:
		return 0x1B, true
	case emulator.KeyCodeBackspace, emulator.KeyCodeDelete, emulator.KeyCodeLeft:
		return 0x08, true
	case emulator.KeyCodeRight:
		return 0x15, true
	case emulator.KeyCodeUp:
		return 0x0B, true
	case emulator.KeyCodeDown:
		return 0x0A, true
	case emulator.KeyCodeTab:
		return 0x09, true
	case emulator.KeyCodeSpace:
		return byte(' '), true
	default:
		return 0, false
	}
}

func translatePrintableRune(char rune) (byte, bool) {
	if char > unicode.MaxASCII {
		return 0, false
	}

	if char >= 'a' && char <= 'z' {
		char = unicode.ToUpper(char)
	}

	if char < 0x20 || char > 0x7E {
		return 0, false
	}

	return byte(char), true
}
