package memory

import "testing"

func TestAppleIIeMMUSwitchesReadAndWriteBanks(t *testing.T) {
	mmu, err := NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	if err := mmu.Write(0x0200, 0x11); err != nil {
		t.Fatalf("write main memory: %v", err)
	}
	if err := mmu.Write(appleIIeMMUSwitchRAMWriteOn, 0); err != nil {
		t.Fatalf("enable RAMWRT: %v", err)
	}
	if err := mmu.Write(0x0200, 0x22); err != nil {
		t.Fatalf("write aux memory: %v", err)
	}
	if err := mmu.Write(appleIIeMMUSwitchRAMReadOn, 0); err != nil {
		t.Fatalf("enable RAMRD: %v", err)
	}

	if got, err := mmu.Read(0x0200); err != nil || got != 0x22 {
		t.Fatalf("expected aux read 0x22, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(appleIIeMMUSwitchReadRAMRead); err != nil || got != 0x80 {
		t.Fatalf("expected RAMRD status=0x80, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(appleIIeMMUSwitchReadRAMWrt); err != nil || got != 0x80 {
		t.Fatalf("expected RAMWRT status=0x80, got 0x%02X err=%v", got, err)
	}

	if err := mmu.Write(appleIIeMMUSwitchRAMReadOff, 0); err != nil {
		t.Fatalf("disable RAMRD: %v", err)
	}
	if got, err := mmu.Read(0x0200); err != nil || got != 0x11 {
		t.Fatalf("expected main read 0x11, got 0x%02X err=%v", got, err)
	}
}

func TestAppleIIeMMUAltZeroPageBanksZeroPageAndStack(t *testing.T) {
	mmu, err := NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	if err := mmu.Write(0x0001, 0x10); err != nil {
		t.Fatalf("write main zero page: %v", err)
	}
	if err := mmu.Write(0x0100, 0x20); err != nil {
		t.Fatalf("write main stack: %v", err)
	}
	if err := mmu.Write(appleIIeMMUSwitchAltZPOn, 0); err != nil {
		t.Fatalf("enable ALTZP: %v", err)
	}
	if err := mmu.Write(0x0001, 0x33); err != nil {
		t.Fatalf("write aux zero page: %v", err)
	}
	if err := mmu.Write(0x0100, 0x44); err != nil {
		t.Fatalf("write aux stack: %v", err)
	}

	if got, err := mmu.Read(0x0001); err != nil || got != 0x33 {
		t.Fatalf("expected aux zero page 0x33, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0x0100); err != nil || got != 0x44 {
		t.Fatalf("expected aux stack 0x44, got 0x%02X err=%v", got, err)
	}

	if err := mmu.Write(appleIIeMMUSwitchAltZPOff, 0); err != nil {
		t.Fatalf("disable ALTZP: %v", err)
	}
	if got, err := mmu.Read(0x0001); err != nil || got != 0x10 {
		t.Fatalf("expected main zero page 0x10, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0x0100); err != nil || got != 0x20 {
		t.Fatalf("expected main stack 0x20, got 0x%02X err=%v", got, err)
	}
}

func TestAppleIIeMMU80StoreOverridesVideoAccessBank(t *testing.T) {
	mmu, err := NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	if err := mmu.Load(0x0400, []byte{0x11}); err != nil {
		t.Fatalf("load main text page: %v", err)
	}
	if err := mmu.LoadAux(0x0400, []byte{0x22}); err != nil {
		t.Fatalf("load aux text page: %v", err)
	}
	if err := mmu.Load(0x2000, []byte{0x33}); err != nil {
		t.Fatalf("load main hires page: %v", err)
	}
	if err := mmu.LoadAux(0x2000, []byte{0x44}); err != nil {
		t.Fatalf("load aux hires page: %v", err)
	}

	if err := mmu.Write(appleIIeMMUSwitch80StoreOn, 0); err != nil {
		t.Fatalf("enable 80STORE: %v", err)
	}
	if err := mmu.Write(appleIIeMMUSwitchPage2, 0); err != nil {
		t.Fatalf("enable PAGE2: %v", err)
	}

	if got, err := mmu.Read(0x0400); err != nil || got != 0x22 {
		t.Fatalf("expected text page read from aux bank, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0x2000); err != nil || got != 0x33 {
		t.Fatalf("expected hires read to remain on main bank before HIRES, got 0x%02X err=%v", got, err)
	}

	if err := mmu.Write(appleIIeMMUSwitchHiRes, 0); err != nil {
		t.Fatalf("enable HIRES: %v", err)
	}
	if got, err := mmu.Read(0x2000); err != nil || got != 0x44 {
		t.Fatalf("expected hires read from aux bank, got 0x%02X err=%v", got, err)
	}
}

func TestAppleIIeMMULoadsInternalROM(t *testing.T) {
	mmu, err := NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	if err := mmu.Load(0xD000, []byte{0xA9, 0x42}); err != nil {
		t.Fatalf("load ROM bytes: %v", err)
	}
	if got, err := mmu.Read(0xD000); err != nil || got != 0xA9 {
		t.Fatalf("expected ROM read 0xA9, got 0x%02X err=%v", got, err)
	}
	if err := mmu.Write(0xD000, 0x99); err != nil {
		t.Fatalf("write underlying RAM byte: %v", err)
	}
	if got, err := mmu.Read(0xD000); err != nil || got != 0xA9 {
		t.Fatalf("expected ROM overlay to stay visible, got 0x%02X err=%v", got, err)
	}

	if err := mmu.Write(appleIIeMMUSwitchIntCxROMOn, 0); err != nil {
		t.Fatalf("enable INTCXROM: %v", err)
	}
	if got, err := mmu.Read(appleIIeMMUSwitchReadIntCxROM); err != nil || got != 0x80 {
		t.Fatalf("expected INTCXROM status=0x80, got 0x%02X err=%v", got, err)
	}
}
