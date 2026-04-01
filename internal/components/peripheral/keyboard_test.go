package peripheral

import (
	"testing"

	"github.com/Djoulzy/emuai/internal/emulator"
)

func TestKeyboardLatchesUppercaseASCII(t *testing.T) {
	keyboard := NewKeyboard("kbd")
	keyboard.HandleKeyEvent(emulator.KeyEvent{Rune: 'a', Action: emulator.KeyActionPress})

	got, err := keyboard.Read(appleIIKeyboardDataAddress)
	if err != nil {
		t.Fatalf("read keyboard data: %v", err)
	}
	if got != 0xC1 {
		t.Fatalf("unexpected latched byte: got 0x%02X want 0xC1", got)
	}

	if err := keyboard.Write(appleIIKeyboardStrobeAddress, 0); err != nil {
		t.Fatalf("clear strobe: %v", err)
	}

	got, err = keyboard.Read(appleIIKeyboardDataAddress)
	if err != nil {
		t.Fatalf("read cleared keyboard data: %v", err)
	}
	if got != 0x41 {
		t.Fatalf("unexpected post-strobe byte: got 0x%02X want 0x41", got)
	}
}

func TestKeyboardQueuesKeysUntilStrobeClears(t *testing.T) {
	keyboard := NewKeyboard("kbd")
	keyboard.HandleKeyEvent(emulator.KeyEvent{Rune: 'a', Action: emulator.KeyActionPress})
	keyboard.HandleKeyEvent(emulator.KeyEvent{Rune: 'b', Action: emulator.KeyActionPress})

	got, err := keyboard.Read(appleIIKeyboardDataAddress)
	if err != nil {
		t.Fatalf("read first key: %v", err)
	}
	if got != 0xC1 {
		t.Fatalf("unexpected first key: got 0x%02X want 0xC1", got)
	}

	if err := keyboard.Write(appleIIKeyboardStrobeAddress, 0); err != nil {
		t.Fatalf("clear strobe for first key: %v", err)
	}

	got, err = keyboard.Read(appleIIKeyboardDataAddress)
	if err != nil {
		t.Fatalf("read second key: %v", err)
	}
	if got != 0xC2 {
		t.Fatalf("unexpected second key: got 0x%02X want 0xC2", got)
	}
}

func TestKeyboardTranslatesSpecialKeys(t *testing.T) {
	keyboard := NewKeyboard("kbd")
	keyboard.HandleKeyEvent(emulator.KeyEvent{Code: emulator.KeyCodeEnter, Action: emulator.KeyActionPress})

	got, err := keyboard.Read(appleIIKeyboardDataAddress)
	if err != nil {
		t.Fatalf("read enter key: %v", err)
	}
	if got != 0x8D {
		t.Fatalf("unexpected enter byte: got 0x%02X want 0x8D", got)
	}

	if err := keyboard.Write(appleIIKeyboardStrobeAddress, 0); err != nil {
		t.Fatalf("clear enter strobe: %v", err)
	}

	keyboard.HandleKeyEvent(emulator.KeyEvent{Code: emulator.KeyCodeRight, Action: emulator.KeyActionPress})
	got, err = keyboard.Read(appleIIKeyboardDataAddress)
	if err != nil {
		t.Fatalf("read right key: %v", err)
	}
	if got != 0x95 {
		t.Fatalf("unexpected right-arrow byte: got 0x%02X want 0x95", got)
	}
}
