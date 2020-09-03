package cpu

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/hexaflex/svm/arch"
	"github.com/hexaflex/svm/asm/ar"
	"github.com/hexaflex/svm/devices"
)

func TestWAIT(t *testing.T) {
	//   WAIT $500
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.WAIT), arg(Constant, 500))
	ct.emit(op(arch.HALT))

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
	//   WAIT8 $500
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.WAIT, i8), arg(Constant, 500))
	ct.emit(op(arch.HALT))

	start := time.Now()
	runTest(t, ct)
	diff := time.Since(start)

	if diff < time.Millisecond*244 {
		t.Fatalf("expected runtime of >= %v; have %v", time.Millisecond*244, diff)
	}
}

func TestMOV(t *testing.T) {
	//    MOV r0, $123
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 123
	ct.want[RIP] = 6
	runTest(t, ct)
}

func TestMOV8(t *testing.T) {
	//    MOV8 r0, $123
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MOV, i8), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 123 << 8
	ct.want[RIP] = 6
	runTest(t, ct)
}

func TestPUSHPOP(t *testing.T) {
	//    MOV r0, $123
	//   PUSH r0
	//    POP r1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.PUSH), arg(Register, 0))
	ct.emit(op(arch.POP), arg(Register, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 123
	ct.want[R1] = 123
	ct.want[UserMemoryCapacity-2] = 123
	ct.want[RIP] = 10
	runTest(t, ct)
}

func TestADD(t *testing.T) {
	//    ADD r0, $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.ADD), arg(Register, 0), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 3
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestADD8(t *testing.T) {
	//   ADD8 r0, $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.ADD, i8), arg(Register, 0), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 3 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestADDOverflow(t *testing.T) {
	//    ADD r0, $0x7fff, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.ADD), arg(Register, 0), arg(Constant, 0x7fff), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = -0x8000
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestADD8Overflow(t *testing.T) {
	//   ADD8 r0, $0x7f, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.ADD, i8), arg(Register, 0), arg(Constant, 0x7f), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = -0x80 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestSUB1(t *testing.T) {
	//    SUB r0, $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SUB), arg(Register, 0), arg(Constant, 2), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB81(t *testing.T) {
	//   SUB8 r0, $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SUB, i8), arg(Register, 0), arg(Constant, 2), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB2(t *testing.T) {
	//    SUB r0, $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SUB), arg(Register, 0), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = -1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB82(t *testing.T) {
	//   SUB8 r0, $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SUB, i8), arg(Register, 0), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = -1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUBOverflow(t *testing.T) {
	//    SUB r0, $-0x7fff, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SUB, i16), arg(Register, 0), arg(Constant, -0x7fff), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 0x7fff
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestSUB8Overflow(t *testing.T) {
	//   SUB:i8 r0, $-0x7f, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SUB, i8), arg(Register, 0), arg(Constant, -0x7f), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 0x7f << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestMUL(t *testing.T) {
	//    MUL r0, $2, $3
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MUL), arg(Register, 0), arg(Constant, 2), arg(Constant, 3))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 6
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMUL8(t *testing.T) {
	//   MUL8 r0, $2, $3
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MUL, i8), arg(Register, 0), arg(Constant, 2), arg(Constant, 3))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 6 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMULOverflow(t *testing.T) {
	//    MUL r0, $0x7fff, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MUL), arg(Register, 0), arg(Constant, 0x7fff), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = -2
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestMUL8Overflow(t *testing.T) {
	//   MUL8 r0, $0x7f, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MUL, i8), arg(Register, 0), arg(Constant, 0x7f), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = -2 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestDIV(t *testing.T) {
	//    DIV r0, $4, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.DIV), arg(Register, 0), arg(Constant, 4), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 2
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestDIV8(t *testing.T) {
	//   DIV8 r0, $4, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.DIV, i8), arg(Register, 0), arg(Constant, 4), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 2 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestDIVDivideByZero(t *testing.T) {
	//    DIV r0, $4, $0
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.DIV), arg(Register, 0), arg(Constant, 4), arg(Constant, 0))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestDIV8DivideByZero(t *testing.T) {
	//   DIV8 r0, $4, $0
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.DIV, i8), arg(Register, 0), arg(Constant, 4), arg(Constant, 0))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestMOD(t *testing.T) {
	//    MOD r0, $4, $3
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MOD), arg(Register, 0), arg(Constant, 4), arg(Constant, 3))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMOD8(t *testing.T) {
	//   MOD8 r0, $4, $3
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MOD, i8), arg(Register, 0), arg(Constant, 4), arg(Constant, 3))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMODDivideByZero(t *testing.T) {
	//    MOD r0, $4, $0
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MOD), arg(Register, 0), arg(Constant, 4), arg(Constant, 0))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestMOD8DivideByZero(t *testing.T) {
	//   MOD8 r0, $4, $0
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.MOD, i8), arg(Register, 0), arg(Constant, 4), arg(Constant, 0))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestSHL(t *testing.T) {
	//    SHL r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SHL), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 10
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestSHL8(t *testing.T) {
	//   SHL8 r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SHL, i8), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 10 << 8
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestSHR(t *testing.T) {
	//    SHR r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SHR), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 2
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestSHR8(t *testing.T) {
	//   SHR8 r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SHR, i8), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 2 << 8
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestAND(t *testing.T) {
	//    AND r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.AND), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestAND8(t *testing.T) {
	//   AND8 r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.AND, i8), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestOR1(t *testing.T) {
	//     OR r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.OR), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestOR81(t *testing.T) {
	//    OR8 r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.OR, i8), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 5 << 8
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestOR2(t *testing.T) {
	//     OR r0, $4, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.OR), arg(Register, 0), arg(Constant, 4), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestOR82(t *testing.T) {
	//    OR8 r0, $4, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.OR, i8), arg(Register, 0), arg(Constant, 4), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 5 << 8
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestXOR1(t *testing.T) {
	//    XOR r0, $4, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.XOR), arg(Register, 0), arg(Constant, 4), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestXOR81(t *testing.T) {
	//   XOR8 r0, $4, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.XOR, i8), arg(Register, 0), arg(Constant, 4), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 5 << 8
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestXOR2(t *testing.T) {
	//    XOR r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.XOR), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 4
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestXOR82(t *testing.T) {
	//   XOR8 r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.XOR, i8), arg(Register, 0), arg(Constant, 5), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 4 << 8
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestABS1(t *testing.T) {
	//    ABS r0, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.ABS), arg(Register, 0), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1
	ct.want[RIP] = 6
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestABS81(t *testing.T) {
	//   ABS8 r0, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.ABS, i8), arg(Register, 0), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 6
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestABS2(t *testing.T) {
	//    ABS r0, $-1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.ABS), arg(Register, 0), arg(Constant, -1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1
	ct.want[RIP] = 6
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestABS82(t *testing.T) {
	//   ABS8 r0, $-1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.ABS, i8), arg(Register, 0), arg(Constant, -1))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 6
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestPOW1(t *testing.T) {
	//    POW r0, $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.POW), arg(Register, 0), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 4
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestPOW81(t *testing.T) {
	//   POW8 r0, $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.POW, i8), arg(Register, 0), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 4 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestPOW2(t *testing.T) {
	//    POW r0, $0x7fff, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.POW), arg(Register, 0), arg(Constant, 0x7fff), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestPOW82(t *testing.T) {
	//   POW8 r0, $0x7f, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.POW, i8), arg(Register, 0), arg(Constant, 0x7f), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestCEQ1(t *testing.T) {
	//    CEQ $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCEQ81(t *testing.T) {
	//   CEQ8 $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CEQ, i8), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCEQ2(t *testing.T) {
	//    CEQ $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCEQ82(t *testing.T) {
	//   CEQ8 $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CEQ, i8), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCNE1(t *testing.T) {
	//    CNE $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CNE), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCNE81(t *testing.T) {
	//   CNE8 $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CNE, i8), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCNE2(t *testing.T) {
	//    CNE $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CNE), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCNE82(t *testing.T) {
	//   CNE8 $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CNE, i8), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGT1(t *testing.T) {
	//    CGT $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGT), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGT81(t *testing.T) {
	//   CGT8 $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGT, i8), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGT2(t *testing.T) {
	//    CGT $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGT), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGT82(t *testing.T) {
	//   CGT8 $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGT, i8), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGT3(t *testing.T) {
	//    CGT $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGT), arg(Constant, 2), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGT83(t *testing.T) {
	//   CGT8 $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGT, i8), arg(Constant, 2), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGE1(t *testing.T) {
	//    CGE $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGE), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGE81(t *testing.T) {
	//   CGE8 $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGE, i8), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGE2(t *testing.T) {
	//    CGE $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGE), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGE82(t *testing.T) {
	//   CGE8 $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CGE, i8), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLT1(t *testing.T) {
	//    CLT $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLT), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLT81(t *testing.T) {
	//   CLT8 $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLT, i8), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLT2(t *testing.T) {
	//    CLT $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLT), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCLT82(t *testing.T) {
	//   CLT8 $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLT, i8), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCLT3(t *testing.T) {
	//    CLT $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLT), arg(Constant, 2), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLT83(t *testing.T) {
	//   CLT8 $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLT, i8), arg(Constant, 2), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLE1(t *testing.T) {
	//    CLE $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLE), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCLE81(t *testing.T) {
	//   CLE8 $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLE, i8), arg(Constant, 2), arg(Constant, 2))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCLE2(t *testing.T) {
	//    CLE $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLE), arg(Constant, 2), arg(Constant, 1))
	ct.emit(op(arch.HALT))

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLE82(t *testing.T) {
	//   CLE8 $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.CLE, i8), arg(Constant, 2), arg(Constant, 1))
	ct.emit(op(arch.HALT))

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
	ct.emit(op(arch.JMP), arg(Constant, 10))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 456))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 456
	ct.want[RIP] = 16
	runTest(t, ct)
}

func TestJEZ1(t *testing.T) {
	//    CEQ  $1, $2
	//    JEZ  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.JEZ), arg(Constant, 17))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 456))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 456
	ct.want[RIP] = 23
	runTest(t, ct)
}

func TestJEZ2(t *testing.T) {
	//    CEQ  $1, $1
	//    JEZ  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 1), arg(Constant, 1))
	ct.emit(op(arch.JEZ), arg(Constant, 17))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 456))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 123
	ct.want[RIP] = 17
	runTest(t, ct)
}

func TestJNZ1(t *testing.T) {
	//    CEQ  $1, $2
	//    JNZ  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 1), arg(Constant, 2))
	ct.emit(op(arch.JNZ), arg(Constant, 17))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 456))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 123
	ct.want[RIP] = 17
	runTest(t, ct)
}

func TestJNZ2(t *testing.T) {
	//    CEQ  $1, $1
	//    JNZ  foo
	//    MOV  r0, 123
	//    HALT
	// :foo
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 1), arg(Constant, 1))
	ct.emit(op(arch.JNZ), arg(Constant, 17))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 456))
	ct.emit(op(arch.HALT))

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
	ct.emit(op(arch.CALL), arg(Constant, 5))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.RET))

	ct.want[R0] = 123
	ct.want[RIP] = 5
	ct.want[UserMemoryCapacity-2] = 4

	addr := UserMemoryCapacity - 2
	ct.want[RSP] = int(int16(addr))

	runTest(t, ct)
}

func TestCLEZ1(t *testing.T) {
	//    CEQ  $0, $1
	//    CLEZ $foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 0), arg(Constant, 1))
	ct.emit(op(arch.CLEZ), arg(Constant, 12))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.RET))

	ct.want[R0] = 123
	runTest(t, ct)
}

func TestCLEZ2(t *testing.T) {
	//    CEQ  $0, $0
	//    CLEZ $foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 0), arg(Constant, 0))
	ct.emit(op(arch.CLEZ), arg(Constant, 12))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.RET))

	ct.want[R0] = 0
	runTest(t, ct)
}

func TestCLNZ1(t *testing.T) {
	//    CEQ  $0, $1
	//    CLNZ $foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 0), arg(Constant, 1))
	ct.emit(op(arch.CLNZ), arg(Constant, 12))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.RET))

	ct.want[R0] = 0
	runTest(t, ct)
}

func TestCLNZ2(t *testing.T) {
	//    CEQ  $0, $0
	//    CLNZ $foo
	//    HALT
	// :foo
	//    MOV  r0, 123
	//    RET

	ct := newCodeTest()
	ct.emit(op(arch.CEQ), arg(Constant, 0), arg(Constant, 0))
	ct.emit(op(arch.CLNZ), arg(Constant, 12))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.RET))

	ct.want[R0] = 123
	runTest(t, ct)
}

func TestHWA1(t *testing.T) {
	//    HWA  r0, $0xc0, $0xffee
	//    JEZ  notfound
	//    MOV  r0, 123
	//    HALT
	// :notfound
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(op(arch.HWA), arg(Register, 0), arg(Constant, testID.Manufacturer()), arg(Constant, testID.Serial()))
	ct.emit(op(arch.JEZ), arg(Constant, 18))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 456))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 123
	ct.want[RIP] = 18

	runTest(t, ct)
}

func TestHWA2(t *testing.T) {
	//    HWA  r0, $0, $0
	//    JEZ  notfound
	//    MOV  r0, 123
	//    HALT
	// :notfound
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(op(arch.HWA), arg(Register, 0), arg(Constant, 0), arg(Constant, 0))
	ct.emit(op(arch.JEZ), arg(Constant, 18))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 123))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 456))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 456
	ct.want[RIP] = 24

	runTest(t, ct)
}

func TestINT(t *testing.T) {
	//    HWA  r0, $0xc0, $0xffee
	//    JEZ  notfound
	//    INT  r0
	//    HALT
	// :notfound
	//    MOV  r0, 456
	//    HALT

	ct := newCodeTest()
	ct.emit(op(arch.HWA), arg(Register, 0), arg(Constant, testID.Manufacturer()), arg(Constant, testID.Serial()))
	ct.emit(op(arch.JEZ), arg(Constant, 15))
	ct.emit(op(arch.INT), arg(Register, 0))
	ct.emit(op(arch.HALT))
	ct.emit(op(arch.MOV), arg(Register, 0), arg(Constant, 456))
	ct.emit(op(arch.HALT))

	ct.want[R0] = 123
	ct.want[RIP] = 15

	runTest(t, ct)
}

func TestSEEDRNG(t *testing.T) {
	//   SEED $0
	//   RNG r0, $0, $10
	//   SEED $0
	//   RNG r1, $0, $10
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SEED), arg(Constant, 0))
	ct.emit(op(arch.RNG), arg(Register, 0), arg(Constant, 0), arg(Constant, 10))
	ct.emit(op(arch.SEED), arg(Constant, 0))
	ct.emit(op(arch.RNG), arg(Register, 1), arg(Constant, 0), arg(Constant, 10))
	ct.emit(op(arch.HALT))

	rng := rand.New(rand.NewSource(0))
	ct.want[R0] = rng.Intn(10)

	rng = rand.New(rand.NewSource(0))
	ct.want[R1] = rng.Intn(10)

	ct.want[RIP] = 25
	runTest(t, ct)
}

func TestSEED8RNG8(t *testing.T) {
	//   SEED8 $0
	//   RNG8 r0, $0, $10
	//   SEED8 $0
	//   RNG8 r1, $0, $10
	//   HALT

	ct := newCodeTest()
	ct.emit(op(arch.SEED, i8), arg(Constant, 0))
	ct.emit(op(arch.RNG, i8), arg(Register, 0), arg(Constant, 0), arg(Constant, 10))
	ct.emit(op(arch.SEED, i8), arg(Constant, 0))
	ct.emit(op(arch.RNG, i8), arg(Register, 1), arg(Constant, 0), arg(Constant, 10))
	ct.emit(op(arch.HALT, i8))

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

	vm := New(trace)
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

func (ct *codeTest) emit(opcode byte, args ...[2]int) {
	w := &ct.program
	w.WriteByte(opcode)

	for _, v := range args {
		switch AddressMode(v[0]) {
		case Constant, Address:
			w.WriteByte(byte(v[0]&0x3) << 6)
			w.WriteByte(byte(v[1] >> 8))
			w.WriteByte(byte(v[1]))

		case Register:
			b := byte(v[0]&0x3) << 6
			b |= byte(v[1]) & 0x3f
			w.WriteByte(b)
		}
	}
}

const (
	u8  = 1 << 6
	u16 = 1<<7 | 1<<6
	i8  = 0
	i16 = 1 << 7
)

func op(opcode int, suffix ...byte) byte {
	if len(suffix) > 0 {
		return suffix[0] | byte(opcode&0x3f)
	}
	return i16 | byte(opcode&0x3f)
}

func arg(mode AddressMode, value int) [2]int {
	return [2]int{int(mode), value}
}

func trace(i *Instruction) {
	name, ok := arch.Name(i.Opcode)
	if !ok {
		name = fmt.Sprintf("%02x", i.Opcode)
	}

	var suffix string
	if i.Wide {
		if i.Signed {
			suffix = ":i16"
		} else {
			suffix = ":u16"
		}
	} else {
		if i.Signed {
			suffix = ":i8"
		} else {
			suffix = ":u8"
		}
	}

	name += suffix

	var sb strings.Builder

	for j := 0; j < arch.Argc(i.Opcode); j++ {
		if j > 0 {
			sb.WriteString(", ")
		}

		addr := i.Args[j].Address
		if addr >= UserMemoryCapacity {
			index := (addr - UserMemoryCapacity) / 2
			reg := arch.RegisterName(index)
			sb.WriteString(fmt.Sprintf("%s (%04x)", reg, i.Args[j].Value))
		} else {
			sb.WriteString(fmt.Sprintf("%04x (%04x)", addr, i.Args[j].Value))
		}
	}

	fmt.Printf("%04x %5s %s\n", i.IP, name, sb.String())
}
