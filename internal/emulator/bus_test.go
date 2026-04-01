package emulator

import "testing"

type stubAddressableDevice struct {
	readValue byte
	writes    []byte
}

func (d *stubAddressableDevice) Read(_ uint16) (byte, error) {
	return d.readValue, nil
}

func (d *stubAddressableDevice) Write(_ uint16, value byte) error {
	d.writes = append(d.writes, value)
	return nil
}

func TestBusPrefersMostSpecificMapping(t *testing.T) {
	bus := NewBus()
	ram := &stubAddressableDevice{readValue: 0xAA}
	keyboard := &stubAddressableDevice{readValue: 0xC1}

	if err := bus.MapDevice(0x0000, 0xFFFF, "ram", ram); err != nil {
		t.Fatalf("map RAM: %v", err)
	}
	if err := bus.MapDevice(0xC000, 0xC000, "keyboard", keyboard); err != nil {
		t.Fatalf("map keyboard: %v", err)
	}

	got, err := bus.Read(0xC000)
	if err != nil {
		t.Fatalf("read keyboard-mapped address: %v", err)
	}
	if got != 0xC1 {
		t.Fatalf("unexpected keyboard byte: got 0x%02X want 0xC1", got)
	}

	got, err = bus.Read(0xC001)
	if err != nil {
		t.Fatalf("read RAM-mapped address: %v", err)
	}
	if got != 0xAA {
		t.Fatalf("unexpected RAM byte: got 0x%02X want 0xAA", got)
	}

	if err := bus.Write(0xC000, 0x55); err != nil {
		t.Fatalf("write keyboard-mapped address: %v", err)
	}
	if len(keyboard.writes) != 1 || keyboard.writes[0] != 0x55 {
		t.Fatalf("expected write to target keyboard device, got %#v", keyboard.writes)
	}
	if len(ram.writes) != 0 {
		t.Fatalf("expected RAM writes to remain untouched, got %#v", ram.writes)
	}
}
