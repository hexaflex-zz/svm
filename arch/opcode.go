// Package arch defines the system's instruction set along with
// some related helper functions.
package arch

import "strings"

// Known opcodes.
const (
	NOP = iota
	HALT
	MOV
	PUSH
	POP
	RNG
	SEED

	ADD
	SUB
	MUL
	DIV
	MOD
	SHL
	SHR
	AND
	OR
	XOR
	ABS
	POW

	CEQ
	CNE
	CGT
	CGE
	CLT
	CLE

	JMP
	JEZ
	JNZ
	CALL
	CLEZ
	CLNZ
	RET

	HWA
	INT

	WAIT
)

// Opcode returns the opcode for the given instruction name.
// Returns false if the name is not recognized.
func Opcode(name string) (int, bool) {
	switch strings.ToUpper(name) {
	case "NOP":
		return NOP, true
	case "HALT":
		return HALT, true
	case "MOV":
		return MOV, true
	case "PUSH":
		return PUSH, true
	case "POP":
		return POP, true
	case "RNG":
		return RNG, true
	case "SEED":
		return SEED, true

	case "ADD":
		return ADD, true
	case "SUB":
		return SUB, true
	case "MUL":
		return MUL, true
	case "DIV":
		return DIV, true
	case "MOD":
		return MOD, true
	case "SHL":
		return SHL, true
	case "SHR":
		return SHR, true
	case "AND":
		return AND, true
	case "OR":
		return OR, true
	case "XOR":
		return XOR, true
	case "ABS":
		return ABS, true
	case "POW":
		return POW, true

	case "CEQ":
		return CEQ, true
	case "CNE":
		return CNE, true
	case "CGT":
		return CGT, true
	case "CGE":
		return CGE, true
	case "CLT":
		return CLT, true
	case "CLE":
		return CLE, true

	case "JMP":
		return JMP, true
	case "JEZ":
		return JEZ, true
	case "JNZ":
		return JNZ, true
	case "CALL":
		return CALL, true
	case "CLEZ":
		return CLEZ, true
	case "CLNZ":
		return CLNZ, true
	case "RET":
		return RET, true

	case "HWA":
		return HWA, true
	case "INT":
		return INT, true

	case "WAIT":
		return WAIT, true
	}

	return 0, false
}

// Name returns the name for the given opcode.
// Returns false if the opcode is not recognized.
func Name(opcode int) (string, bool) {
	switch opcode {
	case NOP:
		return "NOP", true
	case HALT:
		return "HALT", true
	case MOV:
		return "MOV", true
	case PUSH:
		return "PUSH", true
	case POP:
		return "POP", true
	case RNG:
		return "RNG", true
	case SEED:
		return "SEED", true

	case ADD:
		return "ADD", true
	case SUB:
		return "SUB", true
	case MUL:
		return "MUL", true
	case DIV:
		return "DIV", true
	case MOD:
		return "MOD", true
	case SHL:
		return "SHL", true
	case SHR:
		return "SHR", true
	case AND:
		return "AND", true
	case OR:
		return "OR", true
	case XOR:
		return "XOR", true
	case ABS:
		return "ABS", true
	case POW:
		return "POW", true

	case CEQ:
		return "CEQ", true
	case CNE:
		return "CNE", true
	case CGT:
		return "CGT", true
	case CGE:
		return "CGE", true
	case CLT:
		return "CLT", true
	case CLE:
		return "CLE", true

	case JMP:
		return "JMP", true
	case JEZ:
		return "JEZ", true
	case JNZ:
		return "JNZ", true
	case CALL:
		return "CALL", true
	case CLEZ:
		return "CLEZ", true
	case CLNZ:
		return "CLNZ", true
	case RET:
		return "RET", true

	case HWA:
		return "HWA", true
	case INT:
		return "INT", true

	case WAIT:
		return "WAIT", true
	}

	return "", false
}

// Argc returns the number of arguments the given instruction requires.
// Returns -1 if the opcode is not recognized.
func Argc(opcode int) int {
	switch opcode {
	case ADD, SUB, MUL, DIV, MOD, SHL, SHR, AND, OR, XOR, HWA, POW, RNG:
		return 3
	case MOV, CEQ, CNE, CGT, CGE, CLT, CLE, ABS:
		return 2
	case INT, JMP, JEZ, JNZ, CALL, CLEZ, CLNZ, PUSH, POP, SEED, WAIT:
		return 1
	case NOP, HALT, RET:
		return 0
	}
	return -1
}
