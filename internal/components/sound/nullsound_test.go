package sound

import "testing"

func TestNullSoundSpeakerSoftSwitchTogglesOnReadAndWrite(t *testing.T) {
	device := NewNullSound("sound")

	if err := device.Write(appleIISpeakerToggleAddress, 0); err != nil {
		t.Fatalf("write speaker soft-switch: %v", err)
	}
	if !device.PhaseHigh() {
		t.Fatal("expected speaker phase to toggle high after write")
	}
	if device.ToggleCount() != 1 {
		t.Fatalf("unexpected toggle count after write: got %d want 1", device.ToggleCount())
	}

	if _, err := device.Read(appleIISpeakerToggleAddress); err != nil {
		t.Fatalf("read speaker soft-switch: %v", err)
	}
	if device.PhaseHigh() {
		t.Fatal("expected speaker phase to toggle low after read")
	}
	if device.ToggleCount() != 2 {
		t.Fatalf("unexpected toggle count after read: got %d want 2", device.ToggleCount())
	}
}
