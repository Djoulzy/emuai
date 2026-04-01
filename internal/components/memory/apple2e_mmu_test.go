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

func TestAppleIIeMMUSlotROMOverridesInternalCxROMWhenEnabled(t *testing.T) {
	mmu, err := NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	if err := mmu.Load(0xC600, []byte{0x11}); err != nil {
		t.Fatalf("load internal Cx ROM byte: %v", err)
	}
	if err := mmu.LoadSlotROM(6, []byte{0x22}); err != nil {
		t.Fatalf("load slot ROM: %v", err)
	}

	if got, err := mmu.Read(0xC600); err != nil || got != 0x22 {
		t.Fatalf("expected slot ROM byte 0x22, got 0x%02X err=%v", got, err)
	}

	if err := mmu.Write(appleIIeMMUSwitchIntCxROMOn, 0); err != nil {
		t.Fatalf("enable INTCXROM: %v", err)
	}
	if got, err := mmu.Read(0xC600); err != nil || got != 0x11 {
		t.Fatalf("expected internal Cx ROM byte 0x11, got 0x%02X err=%v", got, err)
	}
}

func TestAppleIIeMMUSlotExpansionROMMapsAtC800AfterSlotAccess(t *testing.T) {
	mmu, err := NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	slotROM := make([]byte, 0x0102)
	slotROM[0x00] = 0xAA
	slotROM[0x100] = 0xBB
	slotROM[0x101] = 0xCC
	if err := mmu.LoadSlotROM(5, slotROM); err != nil {
		t.Fatalf("load slot ROM: %v", err)
	}

	if got, err := mmu.Read(0xC500); err != nil || got != 0xAA {
		t.Fatalf("expected slot C5 ROM byte 0xAA, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0xC800); err != nil || got != 0xBB {
		t.Fatalf("expected slot expansion byte 0xBB, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0xC801); err != nil || got != 0xCC {
		t.Fatalf("expected slot expansion byte 0xCC, got 0x%02X err=%v", got, err)
	}

	if _, err := mmu.Read(0xCFFF); err != nil {
		t.Fatalf("read CFFF reset byte: %v", err)
	}
	if got, err := mmu.Read(0xC800); err != nil || got != 0x00 {
		t.Fatalf("expected expansion ROM to be inactive after CFFF, got 0x%02X err=%v", got, err)
	}
}

func TestAppleIIeMMUSlot3DefaultsToInternalROMAndCanBeEnabled(t *testing.T) {
	mmu, err := NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	if err := mmu.Load(0xC300, []byte{0x11}); err != nil {
		t.Fatalf("load internal C3 byte: %v", err)
	}
	slot3ROM := make([]byte, 0x0101)
	slot3ROM[0x00] = 0x22
	slot3ROM[0x100] = 0x33
	if err := mmu.LoadSlotROM(3, slot3ROM); err != nil {
		t.Fatalf("load slot3 ROM: %v", err)
	}

	if got, err := mmu.Read(0xC300); err != nil || got != 0x11 {
		t.Fatalf("expected internal C3 ROM by default, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(appleIIeMMUSwitchReadSlotC3ROM); err != nil || got != 0x00 {
		t.Fatalf("expected SLOTC3ROM status=0x00 by default, got 0x%02X err=%v", got, err)
	}

	if err := mmu.Write(appleIIeMMUSwitchSlotC3ROMOn, 0); err != nil {
		t.Fatalf("enable SLOTC3ROM: %v", err)
	}
	if got, err := mmu.Read(0xC300); err != nil || got != 0x22 {
		t.Fatalf("expected slot3 C3 ROM when enabled, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(appleIIeMMUSwitchReadSlotC3ROM); err != nil || got != 0x80 {
		t.Fatalf("expected SLOTC3ROM status=0x80 when enabled, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0xC800); err != nil || got != 0x33 {
		t.Fatalf("expected slot3 expansion ROM at C800, got 0x%02X err=%v", got, err)
	}

	if err := mmu.Write(appleIIeMMUSwitchSlotC3ROMOff, 0); err != nil {
		t.Fatalf("disable SLOTC3ROM: %v", err)
	}
	if got, err := mmu.Read(0xC300); err != nil || got != 0x11 {
		t.Fatalf("expected internal C3 ROM after disable, got 0x%02X err=%v", got, err)
	}
}

func TestAppleIIeMMUInternalC3AccessClearsActiveC8SlotROM(t *testing.T) {
	mmu, err := NewAppleIIeMMU("mmu")
	if err != nil {
		t.Fatalf("new Apple IIe MMU: %v", err)
	}

	if err := mmu.Load(0xC300, []byte{0x44}); err != nil {
		t.Fatalf("load internal C3 byte: %v", err)
	}
	slot6ROM := make([]byte, 0x0101)
	slot6ROM[0x00] = 0x55
	slot6ROM[0x100] = 0x66
	if err := mmu.LoadSlotROM(6, slot6ROM); err != nil {
		t.Fatalf("load slot6 ROM: %v", err)
	}

	if got, err := mmu.Read(0xC600); err != nil || got != 0x55 {
		t.Fatalf("expected slot6 C6 ROM, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0xC800); err != nil || got != 0x66 {
		t.Fatalf("expected slot6 expansion ROM at C800, got 0x%02X err=%v", got, err)
	}

	if got, err := mmu.Read(0xC300); err != nil || got != 0x44 {
		t.Fatalf("expected internal C3 ROM access to succeed, got 0x%02X err=%v", got, err)
	}
	if got, err := mmu.Read(0xC800); err != nil || got != 0x00 {
		t.Fatalf("expected internal C3 access to clear active C8 slot ROM, got 0x%02X err=%v", got, err)
	}
}
