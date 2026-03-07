package mos6510

import "golang.org/x/exp/maps"

func (C *CPU) initLanguage() {
	opcode_6502 := map[byte]Instruction{

		0x69: {Name: "ADC", bytes: 2, Cycles: 2, action: C.ADC_imm, addr: immediate},
		0x65: {Name: "ADC", bytes: 2, Cycles: 3, action: C.ADC_zep, addr: zeropage},
		0x75: {Name: "ADC", bytes: 2, Cycles: 4, action: C.ADC_zpx, addr: zeropageX},
		0x6D: {Name: "ADC", bytes: 3, Cycles: 4, action: C.ADC_abs, addr: absolute},
		0x7D: {Name: "ADC", bytes: 3, Cycles: 4, action: C.ADC_abx, addr: absoluteX},
		0x79: {Name: "ADC", bytes: 3, Cycles: 4, action: C.ADC_aby, addr: absoluteY},
		0x61: {Name: "ADC", bytes: 2, Cycles: 6, action: C.ADC_inx, addr: indirectX},
		0x71: {Name: "ADC", bytes: 2, Cycles: 5, action: C.ADC_iny, addr: indirectY},

		0x29: {Name: "AND", bytes: 2, Cycles: 2, action: C.AND_imm, addr: immediate},
		0x25: {Name: "AND", bytes: 2, Cycles: 3, action: C.AND_zep, addr: zeropage},
		0x35: {Name: "AND", bytes: 2, Cycles: 4, action: C.AND_zpx, addr: zeropageX},
		0x2D: {Name: "AND", bytes: 3, Cycles: 4, action: C.AND_abs, addr: absolute},
		0x3D: {Name: "AND", bytes: 3, Cycles: 4, action: C.AND_abx, addr: absoluteX},
		0x39: {Name: "AND", bytes: 3, Cycles: 4, action: C.AND_aby, addr: absoluteY},
		0x21: {Name: "AND", bytes: 2, Cycles: 6, action: C.AND_inx, addr: indirectX},
		0x31: {Name: "AND", bytes: 2, Cycles: 5, action: C.AND_iny, addr: indirectY},

		0x0A: {Name: "ASL", bytes: 1, Cycles: 2, action: C.ASL_imp, addr: implied},
		0x06: {Name: "ASL", bytes: 2, Cycles: 5, action: C.ASL_zep, addr: zeropage},
		0x16: {Name: "ASL", bytes: 2, Cycles: 6, action: C.ASL_zpx, addr: zeropageX},
		0x0E: {Name: "ASL", bytes: 3, Cycles: 6, action: C.ASL_abs, addr: absolute},
		0x1E: {Name: "ASL", bytes: 3, Cycles: 7, action: C.ASL_abx, addr: absoluteX},

		0x90: {Name: "BCC", bytes: 2, Cycles: 2, action: C.BCC_rel, addr: relative},
		0xB0: {Name: "BCS", bytes: 2, Cycles: 2, action: C.BCS_rel, addr: relative},
		0xF0: {Name: "BEQ", bytes: 2, Cycles: 2, action: C.BEQ_rel, addr: relative},

		0x24: {Name: "BIT", bytes: 2, Cycles: 3, action: C.BIT_zep, addr: zeropage},
		0x2C: {Name: "BIT", bytes: 3, Cycles: 4, action: C.BIT_abs, addr: absolute},

		0x30: {Name: "BMI", bytes: 2, Cycles: 2, action: C.BMI_rel, addr: relative},

		0xD0: {Name: "BNE", bytes: 2, Cycles: 2, action: C.BNE_rel, addr: relative},

		0x10: {Name: "BPL", bytes: 2, Cycles: 2, action: C.BPL_rel, addr: relative},

		0x00: {Name: "BRK", bytes: 2, Cycles: 7, action: C.BRK_imp, addr: implied},

		0x50: {Name: "BVC", bytes: 2, Cycles: 2, action: C.BVC_rel, addr: relative},

		0x70: {Name: "BVS", bytes: 2, Cycles: 2, action: C.BVS_rel, addr: relative},

		0x18: {Name: "CLC", bytes: 1, Cycles: 2, action: C.CLC_imp, addr: implied},

		0xD8: {Name: "CLD", bytes: 1, Cycles: 2, action: C.CLD_imp, addr: implied},

		0x58: {Name: "CLI", bytes: 1, Cycles: 2, action: C.CLI_imp, addr: implied},

		0xB8: {Name: "CLV", bytes: 1, Cycles: 2, action: C.CLV_imp, addr: implied},

		0xC9: {Name: "CMP", bytes: 2, Cycles: 2, action: C.CMP_imm, addr: immediate},
		0xC5: {Name: "CMP", bytes: 2, Cycles: 3, action: C.CMP_zep, addr: zeropage},
		0xD5: {Name: "CMP", bytes: 2, Cycles: 4, action: C.CMP_zpx, addr: zeropageX},
		0xCD: {Name: "CMP", bytes: 3, Cycles: 4, action: C.CMP_abs, addr: absolute},
		0xDD: {Name: "CMP", bytes: 3, Cycles: 4, action: C.CMP_abx, addr: absoluteX},
		0xD9: {Name: "CMP", bytes: 3, Cycles: 4, action: C.CMP_aby, addr: absoluteY},
		0xC1: {Name: "CMP", bytes: 2, Cycles: 6, action: C.CMP_inx, addr: indirectX},
		0xD1: {Name: "CMP", bytes: 2, Cycles: 5, action: C.CMP_iny, addr: indirectY},

		0xE0: {Name: "CPX", bytes: 2, Cycles: 2, action: C.CPX_imm, addr: immediate},
		0xE4: {Name: "CPX", bytes: 2, Cycles: 3, action: C.CPX_zep, addr: zeropage},
		0xEC: {Name: "CPX", bytes: 3, Cycles: 4, action: C.CPX_abs, addr: absolute},

		0xC0: {Name: "CPY", bytes: 2, Cycles: 2, action: C.CPY_imm, addr: immediate},
		0xC4: {Name: "CPY", bytes: 2, Cycles: 3, action: C.CPY_zep, addr: zeropage},
		0xCC: {Name: "CPY", bytes: 3, Cycles: 4, action: C.CPY_abs, addr: absolute},

		0xC6: {Name: "DEC", bytes: 2, Cycles: 5, action: C.DEC_zep, addr: zeropage},
		0xD6: {Name: "DEC", bytes: 2, Cycles: 6, action: C.DEC_zpx, addr: zeropageX},
		0xCE: {Name: "DEC", bytes: 3, Cycles: 6, action: C.DEC_abs, addr: absolute},
		0xDE: {Name: "DEC", bytes: 3, Cycles: 7, action: C.DEC_abx, addr: absoluteX},

		0xCA: {Name: "DEX", bytes: 1, Cycles: 2, action: C.DEX_imp, addr: implied},
		0x88: {Name: "DEY", bytes: 1, Cycles: 2, action: C.DEY_imp, addr: implied},

		0x49: {Name: "EOR", bytes: 2, Cycles: 2, action: C.EOR_imm, addr: immediate},
		0x45: {Name: "EOR", bytes: 2, Cycles: 3, action: C.EOR_zep, addr: zeropage},
		0x55: {Name: "EOR", bytes: 2, Cycles: 4, action: C.EOR_zpx, addr: zeropageX},
		0x4D: {Name: "EOR", bytes: 3, Cycles: 4, action: C.EOR_abs, addr: absolute},
		0x5D: {Name: "EOR", bytes: 3, Cycles: 4, action: C.EOR_abx, addr: absoluteX},
		0x59: {Name: "EOR", bytes: 3, Cycles: 4, action: C.EOR_aby, addr: absoluteY},
		0x41: {Name: "EOR", bytes: 2, Cycles: 6, action: C.EOR_inx, addr: indirectX},
		0x51: {Name: "EOR", bytes: 2, Cycles: 5, action: C.EOR_iny, addr: indirectY},

		0xE6: {Name: "INC", bytes: 2, Cycles: 5, action: C.INC_zep, addr: zeropage},
		0xF6: {Name: "INC", bytes: 2, Cycles: 6, action: C.INC_zpx, addr: zeropageX},
		0xEE: {Name: "INC", bytes: 3, Cycles: 6, action: C.INC_abs, addr: absolute},
		0xFE: {Name: "INC", bytes: 3, Cycles: 7, action: C.INC_abx, addr: absoluteX},

		0xE8: {Name: "INX", bytes: 1, Cycles: 2, action: C.INX_imp, addr: implied},
		0xC8: {Name: "INY", bytes: 1, Cycles: 2, action: C.INY_imp, addr: implied},

		0x4C: {Name: "JMP", bytes: 3, Cycles: 3, action: C.JMP_abs, addr: absolute},
		0x6C: {Name: "JMP", bytes: 3, Cycles: 5, action: C.JMP_ind, addr: indirect},

		0x20: {Name: "JSR", bytes: 3, Cycles: 6, action: C.JSR_abs, addr: absolute},

		0xA9: {Name: "LDA", bytes: 2, Cycles: 2, action: C.LDA_imm, addr: immediate},
		0xA5: {Name: "LDA", bytes: 2, Cycles: 3, action: C.LDA_zep, addr: zeropage},
		0xB5: {Name: "LDA", bytes: 2, Cycles: 4, action: C.LDA_zpx, addr: zeropageX},
		0xAD: {Name: "LDA", bytes: 3, Cycles: 4, action: C.LDA_abs, addr: absolute},
		0xBD: {Name: "LDA", bytes: 3, Cycles: 4, action: C.LDA_abx, addr: absoluteX},
		0xB9: {Name: "LDA", bytes: 3, Cycles: 4, action: C.LDA_aby, addr: absoluteY},
		0xA1: {Name: "LDA", bytes: 2, Cycles: 6, action: C.LDA_inx, addr: indirectX},
		0xB1: {Name: "LDA", bytes: 2, Cycles: 5, action: C.LDA_iny, addr: indirectY},

		0xA2: {Name: "LDX", bytes: 2, Cycles: 2, action: C.LDX_imm, addr: immediate},
		0xA6: {Name: "LDX", bytes: 2, Cycles: 3, action: C.LDX_zep, addr: zeropage},
		0xB6: {Name: "LDX", bytes: 2, Cycles: 4, action: C.LDX_zpy, addr: zeropageY},
		0xAE: {Name: "LDX", bytes: 3, Cycles: 4, action: C.LDX_abs, addr: absolute},
		0xBE: {Name: "LDX", bytes: 3, Cycles: 4, action: C.LDX_aby, addr: absoluteY},

		0xA0: {Name: "LDY", bytes: 2, Cycles: 2, action: C.LDY_imm, addr: immediate},
		0xA4: {Name: "LDY", bytes: 2, Cycles: 3, action: C.LDY_zep, addr: zeropage},
		0xB4: {Name: "LDY", bytes: 2, Cycles: 4, action: C.LDY_zpx, addr: zeropageX},
		0xAC: {Name: "LDY", bytes: 3, Cycles: 4, action: C.LDY_abs, addr: absolute},
		0xBC: {Name: "LDY", bytes: 3, Cycles: 4, action: C.LDY_abx, addr: absoluteX},

		0x4A: {Name: "LSR", bytes: 1, Cycles: 2, action: C.LSR_imp, addr: implied},
		0x46: {Name: "LSR", bytes: 2, Cycles: 5, action: C.LSR_zep, addr: zeropage},
		0x56: {Name: "LSR", bytes: 2, Cycles: 6, action: C.LSR_zpx, addr: zeropageX},
		0x4E: {Name: "LSR", bytes: 3, Cycles: 6, action: C.LSR_abs, addr: absolute},
		0x5E: {Name: "LSR", bytes: 3, Cycles: 7, action: C.LSR_abx, addr: absoluteX},

		0xEA: {Name: "NOP", bytes: 1, Cycles: 2, action: C.NOP_1x2, addr: implied},

		0x09: {Name: "ORA", bytes: 2, Cycles: 2, action: C.ORA_imm, addr: immediate},
		0x05: {Name: "ORA", bytes: 2, Cycles: 3, action: C.ORA_zep, addr: zeropage},
		0x15: {Name: "ORA", bytes: 2, Cycles: 4, action: C.ORA_zpx, addr: zeropageX},
		0x0D: {Name: "ORA", bytes: 3, Cycles: 4, action: C.ORA_abs, addr: absolute},
		0x1D: {Name: "ORA", bytes: 3, Cycles: 4, action: C.ORA_abx, addr: absoluteX},
		0x19: {Name: "ORA", bytes: 3, Cycles: 4, action: C.ORA_aby, addr: absoluteY},
		0x01: {Name: "ORA", bytes: 2, Cycles: 6, action: C.ORA_inx, addr: indirectX},
		0x11: {Name: "ORA", bytes: 2, Cycles: 5, action: C.ORA_iny, addr: indirectY},

		0x48: {Name: "PHA", bytes: 1, Cycles: 3, action: C.PHA_imp, addr: implied},
		0x08: {Name: "PHP", bytes: 1, Cycles: 3, action: C.PHP_imp, addr: implied},
		0x68: {Name: "PLA", bytes: 1, Cycles: 4, action: C.PLA_imp, addr: implied},
		0x28: {Name: "PLP", bytes: 1, Cycles: 4, action: C.PLP_imp, addr: implied},

		0x2A: {Name: "ROL", bytes: 1, Cycles: 2, action: C.ROL_imp, addr: implied},
		0x26: {Name: "ROL", bytes: 2, Cycles: 5, action: C.ROL_zep, addr: zeropage},
		0x36: {Name: "ROL", bytes: 2, Cycles: 6, action: C.ROL_zpx, addr: zeropageX},
		0x2E: {Name: "ROL", bytes: 3, Cycles: 6, action: C.ROL_abs, addr: absolute},
		0x3E: {Name: "ROL", bytes: 3, Cycles: 7, action: C.ROL_abx, addr: absoluteX},

		0x6A: {Name: "ROR", bytes: 1, Cycles: 2, action: C.ROR_imp, addr: implied},
		0x66: {Name: "ROR", bytes: 2, Cycles: 5, action: C.ROR_zep, addr: zeropage},
		0x76: {Name: "ROR", bytes: 2, Cycles: 6, action: C.ROR_zpx, addr: zeropageX},
		0x6E: {Name: "ROR", bytes: 3, Cycles: 6, action: C.ROR_abs, addr: absolute},
		0x7E: {Name: "ROR", bytes: 3, Cycles: 7, action: C.ROR_abx, addr: absoluteX},

		0x40: {Name: "RTI", bytes: 1, Cycles: 6, action: C.RTI_imp, addr: implied},

		0x60: {Name: "RTS", bytes: 1, Cycles: 6, action: C.RTS_imp, addr: implied},

		0xE9: {Name: "SBC", bytes: 2, Cycles: 2, action: C.SBC_imm, addr: immediate},
		0xE5: {Name: "SBC", bytes: 2, Cycles: 3, action: C.SBC_zep, addr: zeropage},
		0xF5: {Name: "SBC", bytes: 2, Cycles: 4, action: C.SBC_zpx, addr: zeropageX},
		0xED: {Name: "SBC", bytes: 3, Cycles: 4, action: C.SBC_abs, addr: absolute},
		0xFD: {Name: "SBC", bytes: 3, Cycles: 4, action: C.SBC_abx, addr: absoluteX},
		0xF9: {Name: "SBC", bytes: 3, Cycles: 4, action: C.SBC_aby, addr: absoluteY},
		0xE1: {Name: "SBC", bytes: 2, Cycles: 6, action: C.SBC_inx, addr: indirectX},
		0xF1: {Name: "SBC", bytes: 2, Cycles: 5, action: C.SBC_iny, addr: indirectY},

		0x38: {Name: "SEC", bytes: 1, Cycles: 2, action: C.SEC_imp, addr: implied},

		0xF8: {Name: "SED", bytes: 1, Cycles: 2, action: C.SED_imp, addr: implied},

		0x78: {Name: "SEI", bytes: 1, Cycles: 2, action: C.SEI_imp, addr: implied},

		0x85: {Name: "STA", bytes: 2, Cycles: 3, action: C.STA_zep, addr: zeropage},
		0x95: {Name: "STA", bytes: 2, Cycles: 4, action: C.STA_zpx, addr: zeropageX},
		0x8D: {Name: "STA", bytes: 3, Cycles: 4, action: C.STA_abs, addr: absolute},
		0x9D: {Name: "STA", bytes: 3, Cycles: 5, action: C.STA_abx, addr: absoluteX},
		0x99: {Name: "STA", bytes: 3, Cycles: 5, action: C.STA_aby, addr: absoluteY},
		0x81: {Name: "STA", bytes: 2, Cycles: 6, action: C.STA_inx, addr: indirectX},
		0x91: {Name: "STA", bytes: 2, Cycles: 6, action: C.STA_iny, addr: indirectY},

		0x86: {Name: "STX", bytes: 2, Cycles: 3, action: C.STX_zep, addr: zeropage},
		0x96: {Name: "STX", bytes: 2, Cycles: 4, action: C.STX_zpy, addr: zeropageY},
		0x8E: {Name: "STX", bytes: 3, Cycles: 4, action: C.STX_abs, addr: absolute},

		0x84: {Name: "STY", bytes: 2, Cycles: 3, action: C.STY_zep, addr: zeropage},
		0x94: {Name: "STY", bytes: 2, Cycles: 4, action: C.STY_zpx, addr: zeropageX},
		0x8C: {Name: "STY", bytes: 3, Cycles: 4, action: C.STY_abs, addr: absolute},

		0xAA: {Name: "TAX", bytes: 1, Cycles: 2, action: C.TAX_imp, addr: implied},
		0xA8: {Name: "TAY", bytes: 1, Cycles: 2, action: C.TAY_imp, addr: implied},
		0xBA: {Name: "TSX", bytes: 1, Cycles: 2, action: C.TSX_imp, addr: implied},
		0x8A: {Name: "TXA", bytes: 1, Cycles: 2, action: C.TXA_imp, addr: implied},
		0x9A: {Name: "TXS", bytes: 1, Cycles: 2, action: C.TXS_imp, addr: implied},
		0x98: {Name: "TYA", bytes: 1, Cycles: 2, action: C.TYA_imp, addr: implied},
	}

	opcode_65C02 := map[byte]Instruction{

		0x32: {Name: "AND", bytes: 2, Cycles: 5, action: C.AND_izp, addr: indirectzp},
		0x12: {Name: "ORA", bytes: 2, Cycles: 5, action: C.ORA_izp, addr: indirectzp},
		0x52: {Name: "EOR", bytes: 2, Cycles: 5, action: C.EOR_izp, addr: indirectzp},
		0x72: {Name: "ADC", bytes: 2, Cycles: 5, action: C.ADC_izp, addr: indirectzp},
		0xF2: {Name: "SBC", bytes: 2, Cycles: 5, action: C.SBC_izp, addr: indirectzp},
		0x80: {Name: "BRA", bytes: 2, Cycles: 2, action: C.BRA_rel, addr: relative},
		0x0F: {Name: "BBR0", bytes: 3, Cycles: 5, action: C.BBR0_rel, addr: relative},
		0x1F: {Name: "BBR1", bytes: 3, Cycles: 5, action: C.BBR1_rel, addr: relative},
		0x2F: {Name: "BBR2", bytes: 3, Cycles: 5, action: C.BBR2_rel, addr: relative},
		0x3F: {Name: "BBR3", bytes: 3, Cycles: 5, action: C.BBR3_rel, addr: relative},
		0x4F: {Name: "BBR4", bytes: 3, Cycles: 5, action: C.BBR4_rel, addr: relative},
		0x5F: {Name: "BBR5", bytes: 3, Cycles: 5, action: C.BBR5_rel, addr: relative},
		0x6F: {Name: "BBR6", bytes: 3, Cycles: 5, action: C.BBR6_rel, addr: relative},
		0x7F: {Name: "BBR7", bytes: 3, Cycles: 5, action: C.BBR7_rel, addr: relative},
		0x8F: {Name: "BBS0", bytes: 3, Cycles: 5, action: C.BBS0_rel, addr: relative},
		0x9F: {Name: "BBS1", bytes: 3, Cycles: 5, action: C.BBS1_rel, addr: relative},
		0xAF: {Name: "BBS2", bytes: 3, Cycles: 5, action: C.BBS2_rel, addr: relative},
		0xBF: {Name: "BBS3", bytes: 3, Cycles: 5, action: C.BBS3_rel, addr: relative},
		0xCF: {Name: "BBS4", bytes: 3, Cycles: 5, action: C.BBS4_rel, addr: relative},
		0xDF: {Name: "BBS5", bytes: 3, Cycles: 5, action: C.BBS5_rel, addr: relative},
		0xEF: {Name: "BBS6", bytes: 3, Cycles: 5, action: C.BBS6_rel, addr: relative},
		0xFF: {Name: "BBS7", bytes: 3, Cycles: 5, action: C.BBS7_rel, addr: relative},
		0x07: {Name: "RMB0", bytes: 2, Cycles: 5, action: C.RMB0_zep, addr: zeropage},
		0x17: {Name: "RMB1", bytes: 2, Cycles: 5, action: C.RMB1_zep, addr: zeropage},
		0x27: {Name: "RMB2", bytes: 2, Cycles: 5, action: C.RMB2_zep, addr: zeropage},
		0x37: {Name: "RMB3", bytes: 2, Cycles: 5, action: C.RMB3_zep, addr: zeropage},
		0x47: {Name: "RMB4", bytes: 2, Cycles: 5, action: C.RMB4_zep, addr: zeropage},
		0x57: {Name: "RMB5", bytes: 2, Cycles: 5, action: C.RMB5_zep, addr: zeropage},
		0x67: {Name: "RMB6", bytes: 2, Cycles: 5, action: C.RMB6_zep, addr: zeropage},
		0x77: {Name: "RMB7", bytes: 2, Cycles: 5, action: C.RMB7_zep, addr: zeropage},
		0x87: {Name: "SMB0", bytes: 2, Cycles: 5, action: C.SMB0_zep, addr: zeropage},
		0x97: {Name: "SMB1", bytes: 2, Cycles: 5, action: C.SMB1_zep, addr: zeropage},
		0xA7: {Name: "SMB2", bytes: 2, Cycles: 5, action: C.SMB2_zep, addr: zeropage},
		0xB7: {Name: "SMB3", bytes: 2, Cycles: 5, action: C.SMB3_zep, addr: zeropage},
		0xC7: {Name: "SMB4", bytes: 2, Cycles: 5, action: C.SMB4_zep, addr: zeropage},
		0xD7: {Name: "SMB5", bytes: 2, Cycles: 5, action: C.SMB5_zep, addr: zeropage},
		0xE7: {Name: "SMB6", bytes: 2, Cycles: 5, action: C.SMB6_zep, addr: zeropage},
		0xF7: {Name: "SMB7", bytes: 2, Cycles: 5, action: C.SMB7_zep, addr: zeropage},
		0x3A: {Name: "DEA", bytes: 1, Cycles: 2, action: C.DEA_imp, addr: implied},
		0x1A: {Name: "INA", bytes: 1, Cycles: 2, action: C.INA_imp, addr: implied},
		0x7C: {Name: "JMP", bytes: 3, Cycles: 6, action: C.JMP_inx, addr: indirectX},
		0x20: {Name: "JSR", bytes: 3, Cycles: 6, action: C.JSR_abs, addr: absolute},
		0xB2: {Name: "LDA", bytes: 2, Cycles: 5, action: C.LDA_izp, addr: indirectzp},
		0x03: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x13: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x23: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x33: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x43: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x53: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x63: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x73: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x83: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x93: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xA3: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xB3: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xC3: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xD3: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xE3: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xF3: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x0B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x1B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x2B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x3B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x4B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x5B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x6B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x7B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x8B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x9B: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xAB: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xBB: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xCB: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xDB: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xEB: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0xFB: {Name: "NOP", bytes: 1, Cycles: 1, action: C.NOP_1x1, addr: immediate},
		0x02: {Name: "NOP", bytes: 2, Cycles: 2, action: C.NOP_2x2, addr: immediate},
		0x22: {Name: "NOP", bytes: 2, Cycles: 2, action: C.NOP_2x2, addr: immediate},
		0x42: {Name: "NOP", bytes: 2, Cycles: 2, action: C.NOP_2x2, addr: immediate},
		0x62: {Name: "NOP", bytes: 2, Cycles: 2, action: C.NOP_2x2, addr: immediate},
		0x82: {Name: "NOP", bytes: 2, Cycles: 2, action: C.NOP_2x2, addr: immediate},
		0xC2: {Name: "NOP", bytes: 2, Cycles: 2, action: C.NOP_2x2, addr: immediate},
		0xE2: {Name: "NOP", bytes: 2, Cycles: 2, action: C.NOP_2x2, addr: immediate},
		0x54: {Name: "NOP", bytes: 2, Cycles: 4, action: C.NOP_2x4, addr: immediate},
		0xD4: {Name: "NOP", bytes: 2, Cycles: 4, action: C.NOP_2x4, addr: immediate},
		0xF4: {Name: "NOP", bytes: 2, Cycles: 4, action: C.NOP_2x4, addr: immediate},
		0x5C: {Name: "NOP", bytes: 3, Cycles: 8, action: C.NOP_3x8, addr: immediate},
		0xDC: {Name: "NOP", bytes: 3, Cycles: 4, action: C.NOP_3x4, addr: immediate},
		0xFC: {Name: "NOP", bytes: 3, Cycles: 4, action: C.NOP_3x4, addr: immediate},
		0x44: {Name: "NOP", bytes: 2, Cycles: 3, action: C.NOP_2x3, addr: immediate},
		0xDA: {Name: "PHX", bytes: 1, Cycles: 3, action: C.PHX_imp, addr: implied},
		0x5A: {Name: "PHY", bytes: 1, Cycles: 3, action: C.PHY_imp, addr: implied},
		0xFA: {Name: "PLX", bytes: 1, Cycles: 4, action: C.PLX_imp, addr: implied},
		0x7A: {Name: "PLY", bytes: 1, Cycles: 4, action: C.PLY_imp, addr: implied},
		0x92: {Name: "STA", bytes: 2, Cycles: 5, action: C.STA_izp, addr: indirectzp},
		0xD2: {Name: "CMP", bytes: 2, Cycles: 5, action: C.CMP_izp, addr: indirectzp},
		0x64: {Name: "STZ", bytes: 2, Cycles: 3, action: C.STZ_zep, addr: zeropage},
		0x74: {Name: "STZ", bytes: 2, Cycles: 4, action: C.STZ_zpx, addr: zeropageX},
		0x9C: {Name: "STZ", bytes: 3, Cycles: 4, action: C.STZ_abs, addr: absolute},
		0x9E: {Name: "STZ", bytes: 3, Cycles: 5, action: C.STZ_abx, addr: absoluteX},
		0x89: {Name: "BIT", bytes: 2, Cycles: 3, action: C.BIT_imm, addr: immediate},
		0x34: {Name: "BIT", bytes: 3, Cycles: 4, action: C.BIT_zpx, addr: zeropageX},
		0x3C: {Name: "BIT", bytes: 3, Cycles: 4, action: C.BIT_abx, addr: absoluteX},
		0x14: {Name: "TRB", bytes: 2, Cycles: 5, action: C.TRB_zep, addr: zeropage},
		0x1C: {Name: "TRB", bytes: 3, Cycles: 6, action: C.TRB_abs, addr: absolute},
		0x04: {Name: "TSB", bytes: 2, Cycles: 5, action: C.TSB_zep, addr: zeropage},
		0x0C: {Name: "TSB", bytes: 3, Cycles: 6, action: C.TSB_abs, addr: absolute},
	}

	switch C.model {
	case "6502":
		C.Mnemonic = opcode_6502
	case "65C02":
		C.Mnemonic = opcode_6502
		maps.Copy(C.Mnemonic, opcode_65C02)
	}
}
