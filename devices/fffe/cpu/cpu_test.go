package cpu

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/hexaflex/svm/arch"
	"github.com/hexaflex/svm/asm/ar"
	"github.com/hexaflex/svm/devices"
)

func TestWAIT(t *testing.T) {
	//   WAIT 500
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.WAIT, op(arch.ImmediateConstant, 500))
	ct.emit(arch.HALT)

	start := time.Now()
	runTest(t, ct)
	diff := time.Since(start)

	if diff < time.Millisecond*500 {
		t.Fatalf("expected runtime of >= %v; have %v", time.Millisecond*500, diff)
	}
}

func TestWAIT8(t *testing.T) {
	// Wait reads an 8-bit value. 500 gets truncated.
	//
	//   WAIT u8 500
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.WAIT, op(arch.ImmediateConstant, 500, arch.U8))
	ct.emit(arch.HALT)

	start := time.Now()
	runTest(t, ct)
	diff := time.Since(start)

	if diff < time.Millisecond*244 {
		t.Fatalf("expected runtime of >= %v; have %v", time.Millisecond*244, diff)
	}
}

func TestMOV1(t *testing.T) {
	//    MOV r0, 123
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)

	ct.want[R0] = 123
	ct.want[RIP] = 6
	runTest(t, ct)
}

func TestMOV2(t *testing.T) {
	//    MOV u8 r0, 123
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0, arch.U8), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)

	ct.want[R0] = 123 << 8
	ct.want[RIP] = 6
	runTest(t, ct)
}

func TestMOV3(t *testing.T) {
	//    MOV r0, 123
	//    MOV [r0], 321
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.MOV, op(arch.IndirectRegister, 0), op(arch.ImmediateConstant, 321))
	ct.emit(arch.HALT)

	ct.want[R0] = 123
	ct.want[123] = 321
	ct.want[RIP] = 0xb
	runTest(t, ct)
}

func TestMOV4(t *testing.T) {
	//    MOV [123], 321
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOV, op(arch.IndirectConstant, 123), op(arch.ImmediateConstant, 321))
	ct.emit(arch.HALT)

	ct.want[123] = 321
	ct.want[RIP] = 8
	runTest(t, ct)
}

func TestPUSHPOP(t *testing.T) {
	//    MOV r0, 123
	//   PUSH r0
	//    POP r1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.PUSH, op(arch.ImmediateRegister, 0))
	ct.emit(arch.POP, op(arch.ImmediateRegister, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 123
	ct.want[R1] = 123
	ct.want[UserMemoryCapacity-2] = 123
	ct.want[RIP] = 10
	runTest(t, ct)
}

func TestINC(t *testing.T) {
	//    MOV r0, 123
	//    INC r0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.INC, op(arch.ImmediateRegister, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 124
	ct.want[RIP] = 8
	runTest(t, ct)
}

func TestDEC(t *testing.T) {
	//    MOV r0, 123
	//    DEC r0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.DEC, op(arch.ImmediateRegister, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 122
	ct.want[RIP] = 8
	runTest(t, ct)
}
func TestADD(t *testing.T) {
	//    ADD r0, 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ADD, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 3
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestADD8(t *testing.T) {
	//   ADD u8 r0, 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ADD, op(arch.ImmediateRegister, 0, arch.U8), op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 3 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestADDOverflow(t *testing.T) {
	//    ADD r0, 0x7fff, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ADD, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 0x7fff), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = -0x8000
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestADD8Overflow(t *testing.T) {
	//   ADD i8 r0, 0x7f, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ADD, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 0x7f), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = -0x80 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestSUB1(t *testing.T) {
	//    SUB r0, 2, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB81(t *testing.T) {
	//   SUB i8 r0, 2, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB2(t *testing.T) {
	//    SUB r0, 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = -1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB82(t *testing.T) {
	//   SUB i8 r0, 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = -1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUBOverflow(t *testing.T) {
	//    SUB r0, -0x7fff, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, -0x7fff), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 0x7fff
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestSUB8Overflow(t *testing.T) {
	//   SUB i8 r0, -0x7f, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, -0x7f), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 0x7f << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestMUL(t *testing.T) {
	//    MUL r0, 2, 3
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MUL, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 3))
	ct.emit(arch.HALT)

	ct.want[R0] = 6
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMUL8(t *testing.T) {
	//   MUL u8 r0, 2, 3
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MUL, op(arch.ImmediateRegister, 0, arch.U8), op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 3))
	ct.emit(arch.HALT)

	ct.want[R0] = 6 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMULOverflow(t *testing.T) {
	//    MUL r0, 0x7fff, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MUL, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 0x7fff), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = -2
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestMUL8Overflow(t *testing.T) {
	//   MUL i8 r0, 0x7f, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MUL, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 0x7f), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = -2 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestDIV(t *testing.T) {
	//    DIV r0, 4, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.DIV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 2
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestDIV8(t *testing.T) {
	//   DIV i8 r0, 4, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.DIV, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 2 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestDIVDivideByZero(t *testing.T) {
	//    DIV r0, 4, 0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.DIV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestDIV8DivideByZero(t *testing.T) {
	//   DIV i8 r0, 4, 0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.DIV, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestMOD(t *testing.T) {
	//    MOD r0, 4, 3
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOD, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 3))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMOD8(t *testing.T) {
	//   MOD i8 r0, 4, 3
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOD, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 3))
	ct.emit(arch.HALT)

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMODDivideByZero(t *testing.T) {
	//    MOD r0, 4, 0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOD, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestMOD8DivideByZero(t *testing.T) {
	//   MOD i8 r0, 4, 0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOD, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestSHL(t *testing.T) {
	//    SHL r0, 5, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SHL, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 5), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 10
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestSHR(t *testing.T) {
	//    SHR r0, 5, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SHR, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 5), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 2
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestAND(t *testing.T) {
	//    AND r0, 5, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.AND, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 5), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestOR1(t *testing.T) {
	//     OR r0, 5, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.OR, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 5), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestOR2(t *testing.T) {
	//     OR r0, 4, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.OR, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestXOR1(t *testing.T) {
	//    XOR r0, 4, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.XOR, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 4), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestXOR2(t *testing.T) {
	//    XOR r0, 5, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.XOR, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 5), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 4
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestABS1(t *testing.T) {
	//    ABS r0, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ABS, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 6
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestABS2(t *testing.T) {
	//   ABS i8 r0, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ABS, op(arch.ImmediateRegister, 0, arch.I8), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 6
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestPOW1(t *testing.T) {
	//    POW r0, 2, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.POW, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 4
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestPOW2(t *testing.T) {
	//    POW r0, 0x7fff, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.POW, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 0x7fff), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestCEQ1(t *testing.T) {
	//    CEQ 2, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCEQ81(t *testing.T) {
	//   CEQ i8 2, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 2, arch.I8), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCEQ2(t *testing.T) {
	//    CEQ 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCNE1(t *testing.T) {
	//    CNE 2, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CNE, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCNE2(t *testing.T) {
	//    CNE 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CNE, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGT1(t *testing.T) {
	//    CGT 2, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGT, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGT2(t *testing.T) {
	//    CGT 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGT, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGT3(t *testing.T) {
	//    CGT 2, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGT, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGE1(t *testing.T) {
	//    CGE 2, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGE, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGE2(t *testing.T) {
	//    CGE 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGE, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLT1(t *testing.T) {
	//    CLT 2, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLT, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLT2(t *testing.T) {
	//    CLT 1, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLT, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCLT3(t *testing.T) {
	//    CLT 2, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLT, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLE1(t *testing.T) {
	//    CLE 2, 2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLE, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCLE2(t *testing.T) {
	//    CLE 2, 1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLE, op(arch.ImmediateConstant, 2), op(arch.ImmediateConstant, 1))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestJMP(t *testing.T) {
	//    JMP  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(arch.JMP, op(arch.ImmediateConstant, 10))
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 456))
	ct.emit(arch.HALT)

	ct.want[R0] = 456
	ct.want[RIP] = 16
	runTest(t, ct)
}

func TestJEZ1(t *testing.T) {
	//    CEQ  1, 2
	//    JEZ  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.JEZ, op(arch.ImmediateConstant, 17))
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 456))
	ct.emit(arch.HALT)

	ct.want[R0] = 456
	ct.want[RIP] = 23
	runTest(t, ct)
}

func TestJEZ2(t *testing.T) {
	//    CEQ  1, 1
	//    JEZ  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 1))
	ct.emit(arch.JEZ, op(arch.ImmediateConstant, 17))
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 456))
	ct.emit(arch.HALT)

	ct.want[R0] = 123
	ct.want[RIP] = 17
	runTest(t, ct)
}

func TestJNZ1(t *testing.T) {
	//    CEQ  1, 2
	//    JNZ  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 2))
	ct.emit(arch.JNZ, op(arch.ImmediateConstant, 17))
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 456))
	ct.emit(arch.HALT)

	ct.want[R0] = 123
	ct.want[RIP] = 17
	runTest(t, ct)
}

func TestJNZ2(t *testing.T) {
	//    CEQ  1, 1
	//    JNZ  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 1), op(arch.ImmediateConstant, 1))
	ct.emit(arch.JNZ, op(arch.ImmediateConstant, 17))
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 456))
	ct.emit(arch.HALT)

	ct.want[R0] = 456
	ct.want[RIP] = 23
	runTest(t, ct)
}

func TestCALLRET(t *testing.T) {
	//    CALL foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(arch.CALL, op(arch.ImmediateConstant, 5))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.RET)

	ct.want[R0] = 123
	ct.want[RIP] = 5
	ct.want[UserMemoryCapacity-2] = 4

	addr := UserMemoryCapacity - 2
	ct.want[RSP] = int(int16(addr))

	runTest(t, ct)
}

func TestCLEZ1(t *testing.T) {
	//    CEQ  0, 1
	//    CLEZ foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 0), op(arch.ImmediateConstant, 1))
	ct.emit(arch.CLEZ, op(arch.ImmediateConstant, 12))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.RET)

	ct.want[R0] = 123
	runTest(t, ct)
}

func TestCLEZ2(t *testing.T) {
	//    CEQ  0, 0
	//    CLEZ foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 0), op(arch.ImmediateConstant, 0))
	ct.emit(arch.CLEZ, op(arch.ImmediateConstant, 12))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.RET)

	ct.want[R0] = 0
	runTest(t, ct)
}

func TestCLNZ1(t *testing.T) {
	//    CEQ  0, 1
	//    CLNZ foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 0), op(arch.ImmediateConstant, 1))
	ct.emit(arch.CLNZ, op(arch.ImmediateConstant, 12))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.RET)

	ct.want[R0] = 0
	runTest(t, ct)
}

func TestCLNZ2(t *testing.T) {
	//    CEQ  0, 0
	//    CLNZ foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(arch.ImmediateConstant, 0), op(arch.ImmediateConstant, 0))
	ct.emit(arch.CLNZ, op(arch.ImmediateConstant, 12))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.RET)

	ct.want[R0] = 123
	runTest(t, ct)
}

func TestHWA1(t *testing.T) {
	//    HWA  r0, 0xc0, 0xffee
	//    JEZ  notfound
	//    MOV  r0, 123
	//    HALT
	// :notfound
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(arch.HWA, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, testID.Manufacturer()), op(arch.ImmediateConstant, testID.Serial()))
	ct.emit(arch.JEZ, op(arch.ImmediateConstant, 18))
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 456))
	ct.emit(arch.HALT)

	ct.want[R0] = 123
	ct.want[RIP] = 18

	runTest(t, ct)
}

func TestHWA2(t *testing.T) {
	//    HWA  r0, 0, 0
	//    JEZ  notfound
	//    MOV  r0, 123
	//    HALT
	// :notfound
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(arch.HWA, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 0, arch.U16), op(arch.ImmediateConstant, 0, arch.U16))
	ct.emit(arch.JEZ, op(arch.ImmediateConstant, 18))
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 456))
	ct.emit(arch.HALT)

	ct.want[R0] = 456
	ct.want[RIP] = 24

	runTest(t, ct)
}

func TestINT(t *testing.T) {
	//    HWA  r0, 0xc0, 0xffee
	//    JEZ  notfound
	//    INT  r0
	//    HALT
	// :notfound
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(arch.HWA,
		op(arch.ImmediateRegister, 0),
		op(arch.ImmediateConstant, testID.Manufacturer(), arch.U16),
		op(arch.ImmediateConstant, testID.Serial(), arch.U16))
	ct.emit(arch.JEZ, op(arch.ImmediateConstant, 15))
	ct.emit(arch.INT, op(arch.ImmediateRegister, 0))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 456))
	ct.emit(arch.HALT)

	ct.want[R0] = 123
	ct.want[RIP] = 15

	runTest(t, ct)
}

func TestSEEDRNG(t *testing.T) {
	//   SEED 0
	//   RNG r0, 0, 10
	//   SEED 0
	//   RNG r1, 0, 10
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SEED, op(arch.ImmediateConstant, 0))
	ct.emit(arch.RNG, op(arch.ImmediateRegister, 0), op(arch.ImmediateConstant, 0), op(arch.ImmediateConstant, 10))
	ct.emit(arch.SEED, op(arch.ImmediateConstant, 0))
	ct.emit(arch.RNG, op(arch.ImmediateRegister, 1), op(arch.ImmediateConstant, 0), op(arch.ImmediateConstant, 10))
	ct.emit(arch.HALT)

	rng := rand.New(rand.NewSource(0))
	ct.want[R0] = rng.Intn(10)

	rng = rand.New(rand.NewSource(0))
	ct.want[R1] = rng.Intn(10)

	ct.want[RIP] = 25
	runTest(t, ct)
}

func TestSEED8RNG8(t *testing.T) {
	//   SEED 0
	//   RNG u8 r0, 0, 10
	//   SEED 0
	//   RNG u8 r1, 0, 10
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SEED, op(arch.ImmediateConstant, 0))
	ct.emit(arch.RNG, op(arch.ImmediateRegister, 0, arch.U8), op(arch.ImmediateConstant, 0), op(arch.ImmediateConstant, 10))
	ct.emit(arch.SEED, op(arch.ImmediateConstant, 0))
	ct.emit(arch.RNG, op(arch.ImmediateRegister, 1, arch.U8), op(arch.ImmediateConstant, 0), op(arch.ImmediateConstant, 10))
	ct.emit(arch.HALT)

	rng := rand.New(rand.NewSource(0))
	ct.want[R0] = rng.Intn(10) << 8

	rng = rand.New(rand.NewSource(0))
	ct.want[R1] = rng.Intn(10) << 8

	ct.want[RIP] = 25
	runTest(t, ct)
}

func runTest(t *testing.T, ct *codeTest) {
	t.Helper()

	ar := ar.New()
	ar.Instructions = ct.program.Bytes()

	vm := New(nil)
	vm.Connect(&testDevice{})

	if err := vm.Startup(); err != nil {
		t.Fatalf("Startup failure: %v", err)
	}

	copy(vm.memory, ar.Instructions)

	for i := 0; i < UserMemoryCapacity; i++ {
		if err := vm.Step(); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Step failure: %v", err)
		}
	}

	if err := vm.Shutdown(); err != nil {
		t.Fatalf("Shutdown failure: %v", err)
	}

	cmp := func(addr, want, have int) {
		if have != want {
			t.Fatalf("state mismatch at 0x%04x:\nwant: %#x\nhave: %#x\n", addr, want, have)
		}
	}

	for addr, want := range ct.want {
		if addr == RST {
			cmp(addr, want, vm.memory.U8(addr))
		} else {
			cmp(addr, want, vm.memory.I16(addr))
		}
	}
}

const testID devices.ID = 0xc0ffee

type testDevice struct{}

func (d *testDevice) ID() devices.ID                { return testID }
func (d *testDevice) Startup(devices.IntFunc) error { return nil }
func (d *testDevice) Shutdown() error               { return nil }
func (d *testDevice) Int(m devices.Memory)          { m.SetI16(R0, 123) }

type codeTest struct {
	program bytes.Buffer
	want    map[int]int
}

func newCodeTest() *codeTest {
	return &codeTest{
		want: make(map[int]int),
	}
}

func (ct *codeTest) emit(opcode byte, ops ...[3]int) {
	w := &ct.program
	w.WriteByte(opcode)

	for _, v := range ops {
		attr := byte(v[0]&0x3)<<6 | byte(v[1]&0x3)<<4

		switch arch.AddressMode(v[0]) {
		case arch.ImmediateConstant, arch.IndirectConstant:
			w.WriteByte(attr)
			w.WriteByte(byte(v[2] >> 8))
			w.WriteByte(byte(v[2]))

		case arch.ImmediateRegister, arch.IndirectRegister:
			w.WriteByte(attr | byte(v[2]&0xf))
		}
	}
}

func op(mode arch.AddressMode, value int, typ ...arch.Type) [3]int {
	if len(typ) > 0 {
		return [...]int{int(mode), int(typ[0]), value}
	}
	return [...]int{int(mode), int(arch.I16), value}
}
