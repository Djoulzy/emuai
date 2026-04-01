package peripheral

import (
	"github.com/Djoulzy/emuai/internal/components/sound"
	"github.com/Djoulzy/emuai/internal/components/video"
	"github.com/Djoulzy/emuai/internal/emulator"
)

const (
	appleIIeSoftSwitchReadKeyboardData   uint16 = 0xC000
	appleIIeSoftSwitchWrite80StoreOn     uint16 = 0xC001
	appleIIeSoftSwitchWrite80StoreOff    uint16 = 0xC000
	appleIIeSoftSwitchWriteRAMReadOff    uint16 = 0xC002
	appleIIeSoftSwitchWriteRAMReadOn     uint16 = 0xC003
	appleIIeSoftSwitchWriteRAMWriteOff   uint16 = 0xC004
	appleIIeSoftSwitchWriteRAMWriteOn    uint16 = 0xC005
	appleIIeSoftSwitchWriteIntCxROMOff   uint16 = 0xC006
	appleIIeSoftSwitchWriteIntCxROMOn    uint16 = 0xC007
	appleIIeSoftSwitchWriteAltZPOff      uint16 = 0xC008
	appleIIeSoftSwitchWriteAltZPOn       uint16 = 0xC009
	appleIIeSoftSwitchWriteSlotC3ROMOff  uint16 = 0xC00A
	appleIIeSoftSwitchWriteSlotC3ROMOn   uint16 = 0xC00B
	appleIIeSoftSwitchWrite80ColOff      uint16 = 0xC00C
	appleIIeSoftSwitchWrite80ColOn       uint16 = 0xC00D
	appleIIeSoftSwitchWriteAltCharsetOff uint16 = 0xC00E
	appleIIeSoftSwitchWriteAltCharsetOn  uint16 = 0xC00F
	appleIIeSoftSwitchReadKeyboardStrobe uint16 = 0xC010
	appleIIeSoftSwitchReadRAMRead        uint16 = 0xC013
	appleIIeSoftSwitchReadRAMWrite       uint16 = 0xC014
	appleIIeSoftSwitchReadIntCxROM       uint16 = 0xC015
	appleIIeSoftSwitchReadAltZP          uint16 = 0xC016
	appleIIeSoftSwitchReadSlotC3ROM      uint16 = 0xC017
	appleIIeSoftSwitchRead80Store        uint16 = 0xC018
	appleIIeSoftSwitchReadVBlank         uint16 = 0xC019
	appleIIeSoftSwitchReadText           uint16 = 0xC01A
	appleIIeSoftSwitchReadMixed          uint16 = 0xC01B
	appleIIeSoftSwitchReadPage2          uint16 = 0xC01C
	appleIIeSoftSwitchReadHiRes          uint16 = 0xC01D
	appleIIeSoftSwitchReadAltCharset     uint16 = 0xC01E
	appleIIeSoftSwitchRead80Col          uint16 = 0xC01F
	appleIIeSoftSwitchSpeakerToggle      uint16 = 0xC030
	appleIIeSoftSwitchGraphics           uint16 = 0xC050
	appleIIeSoftSwitchText               uint16 = 0xC051
	appleIIeSoftSwitchFull               uint16 = 0xC052
	appleIIeSoftSwitchMixed              uint16 = 0xC053
	appleIIeSoftSwitchPage1              uint16 = 0xC054
	appleIIeSoftSwitchPage2              uint16 = 0xC055
	appleIIeSoftSwitchLoRes              uint16 = 0xC056
	appleIIeSoftSwitchHiRes              uint16 = 0xC057
)

type AppleIIeSoftSwitches struct {
	keyboard *Keyboard
	video    *video.AppleIIeCRTC
	sound    *sound.NullSound
	memory   emulator.AddressableDevice
}

func NewAppleIIeSoftSwitches(keyboard *Keyboard, videoDevice *video.AppleIIeCRTC, soundDevice *sound.NullSound, memoryDevice emulator.AddressableDevice) *AppleIIeSoftSwitches {
	return &AppleIIeSoftSwitches{keyboard: keyboard, video: videoDevice, sound: soundDevice, memory: memoryDevice}
}

func (s *AppleIIeSoftSwitches) Read(addr uint16) (byte, error) {
	switch addr {
	case appleIIeSoftSwitchReadKeyboardData, appleIIeSoftSwitchReadKeyboardStrobe:
		if s.keyboard != nil {
			return s.keyboard.Read(addr)
		}
	case appleIIeSoftSwitchReadRAMRead,
		appleIIeSoftSwitchReadRAMWrite,
		appleIIeSoftSwitchReadIntCxROM,
		appleIIeSoftSwitchReadAltZP,
		appleIIeSoftSwitchReadSlotC3ROM,
		appleIIeSoftSwitchRead80Store:
		if s.memory != nil {
			return s.memory.Read(addr)
		}
	case appleIIeSoftSwitchReadVBlank,
		appleIIeSoftSwitchReadText,
		appleIIeSoftSwitchReadMixed,
		appleIIeSoftSwitchReadPage2,
		appleIIeSoftSwitchReadHiRes,
		appleIIeSoftSwitchReadAltCharset,
		appleIIeSoftSwitchRead80Col,
		appleIIeSoftSwitchWrite80ColOff,
		appleIIeSoftSwitchWrite80ColOn,
		appleIIeSoftSwitchGraphics,
		appleIIeSoftSwitchText,
		appleIIeSoftSwitchFull,
		appleIIeSoftSwitchMixed,
		appleIIeSoftSwitchPage1,
		appleIIeSoftSwitchPage2,
		appleIIeSoftSwitchLoRes,
		appleIIeSoftSwitchHiRes:
		if s.video != nil {
			return s.video.Read(addr)
		}
	case appleIIeSoftSwitchSpeakerToggle:
		if s.sound != nil {
			return s.sound.Read(addr)
		}
	}
	return 0x00, nil
}

func (s *AppleIIeSoftSwitches) Write(addr uint16, value byte) error {
	switch addr {
	case appleIIeSoftSwitchReadKeyboardStrobe:
		if s.keyboard != nil {
			return s.keyboard.Write(addr, value)
		}
	case appleIIeSoftSwitchWrite80StoreOff,
		appleIIeSoftSwitchWrite80StoreOn,
		appleIIeSoftSwitchWriteRAMReadOff,
		appleIIeSoftSwitchWriteRAMReadOn,
		appleIIeSoftSwitchWriteRAMWriteOff,
		appleIIeSoftSwitchWriteRAMWriteOn,
		appleIIeSoftSwitchWriteIntCxROMOff,
		appleIIeSoftSwitchWriteIntCxROMOn,
		appleIIeSoftSwitchWriteAltZPOff,
		appleIIeSoftSwitchWriteAltZPOn,
		appleIIeSoftSwitchWriteSlotC3ROMOff,
		appleIIeSoftSwitchWriteSlotC3ROMOn,
		appleIIeSoftSwitchPage1,
		appleIIeSoftSwitchPage2,
		appleIIeSoftSwitchLoRes,
		appleIIeSoftSwitchHiRes:
		if s.memory != nil {
			if err := s.memory.Write(addr, value); err != nil {
				return err
			}
		}
		if s.video != nil {
			switch addr {
			case appleIIeSoftSwitchWrite80StoreOff,
				appleIIeSoftSwitchWrite80StoreOn,
				appleIIeSoftSwitchPage1,
				appleIIeSoftSwitchPage2,
				appleIIeSoftSwitchLoRes,
				appleIIeSoftSwitchHiRes:
				return s.video.Write(addr, value)
			}
		}
		return nil
	case appleIIeSoftSwitchWrite80ColOff,
		appleIIeSoftSwitchWrite80ColOn,
		appleIIeSoftSwitchWriteAltCharsetOff,
		appleIIeSoftSwitchWriteAltCharsetOn,
		appleIIeSoftSwitchGraphics,
		appleIIeSoftSwitchText,
		appleIIeSoftSwitchFull,
		appleIIeSoftSwitchMixed:
		if s.video != nil {
			return s.video.Write(addr, value)
		}
	case appleIIeSoftSwitchSpeakerToggle:
		if s.sound != nil {
			return s.sound.Write(addr, value)
		}
	}
	return nil
}
