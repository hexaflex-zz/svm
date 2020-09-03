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
	ct.emit(arch.WAIT, op(Constant, 500))
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
	//   WAIT u8 $500
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.WAIT, op(Constant, 500, arch.U8))
	ct.emit(arch.HALT)

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
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.HALT)

	ct.want[R0] = 123
	ct.want[RIP] = 6
	runTest(t, ct)
}

func TestMOV8(t *testing.T) {
	//    MOV u8 r0, $123
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOV, op(Register, 0, arch.U8), op(Constant, 123))
	ct.emit(arch.HALT)

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
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.PUSH, op(Register, 0))
	ct.emit(arch.POP, op(Register, 1))
	ct.emit(arch.HALT)

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
	ct.emit(arch.ADD, op(Register, 0), op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 3
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestADD8(t *testing.T) {
	//   ADD u8 r0, $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ADD, op(Register, 0, arch.U8), op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 3 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestADDOverflow(t *testing.T) {
	//    ADD r0, $0x7fff, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ADD, op(Register, 0), op(Constant, 0x7fff), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = -0x8000
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestADD8Overflow(t *testing.T) {
	//   ADD i8 r0, $0x7f, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ADD, op(Register, 0, arch.I8), op(Constant, 0x7f), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = -0x80 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestSUB1(t *testing.T) {
	//    SUB r0, $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(Register, 0), op(Constant, 2), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB81(t *testing.T) {
	//   SUB i8 r0, $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(Register, 0, arch.I8), op(Constant, 2), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB2(t *testing.T) {
	//    SUB r0, $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(Register, 0), op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = -1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUB82(t *testing.T) {
	//   SUB i8 r0, $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(Register, 0, arch.I8), op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = -1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestSUBOverflow(t *testing.T) {
	//    SUB r0, $-0x7fff, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(Register, 0), op(Constant, -0x7fff), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 0x7fff
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestSUB8Overflow(t *testing.T) {
	//   SUB i8 r0, $-0x7f, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SUB, op(Register, 0, arch.I8), op(Constant, -0x7f), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 0x7f << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestMUL(t *testing.T) {
	//    MUL r0, $2, $3
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MUL, op(Register, 0), op(Constant, 2), op(Constant, 3))
	ct.emit(arch.HALT)

	ct.want[R0] = 6
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMUL8(t *testing.T) {
	//   MUL u8 r0, $2, $3
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MUL, op(Register, 0, arch.U8), op(Constant, 2), op(Constant, 3))
	ct.emit(arch.HALT)

	ct.want[R0] = 6 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMULOverflow(t *testing.T) {
	//    MUL r0, $0x7fff, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MUL, op(Register, 0), op(Constant, 0x7fff), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = -2
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestMUL8Overflow(t *testing.T) {
	//   MUL i8 r0, $0x7f, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MUL, op(Register, 0, arch.I8), op(Constant, 0x7f), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = -2 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestDIV(t *testing.T) {
	//    DIV r0, $4, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.DIV, op(Register, 0), op(Constant, 4), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 2
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestDIV8(t *testing.T) {
	//   DIV i8 r0, $4, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.DIV, op(Register, 0, arch.I8), op(Constant, 4), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 2 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestDIVDivideByZero(t *testing.T) {
	//    DIV r0, $4, $0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.DIV, op(Register, 0), op(Constant, 4), op(Constant, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestDIV8DivideByZero(t *testing.T) {
	//   DIV i8 r0, $4, $0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.DIV, op(Register, 0, arch.I8), op(Constant, 4), op(Constant, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestMOD(t *testing.T) {
	//    MOD r0, $4, $3
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOD, op(Register, 0), op(Constant, 4), op(Constant, 3))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMOD8(t *testing.T) {
	//   MOD i8 r0, $4, $3
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOD, op(Register, 0, arch.I8), op(Constant, 4), op(Constant, 3))
	ct.emit(arch.HALT)

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestMODDivideByZero(t *testing.T) {
	//    MOD r0, $4, $0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOD, op(Register, 0), op(Constant, 4), op(Constant, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestMOD8DivideByZero(t *testing.T) {
	//   MOD i8 r0, $4, $0
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.MOD, op(Register, 0, arch.I8), op(Constant, 4), op(Constant, 0))
	ct.emit(arch.HALT)

	ct.want[R0] = 0
	ct.want[RIP] = 9
	ct.want[RST] = 4
	runTest(t, ct)
}

func TestSHL(t *testing.T) {
	//    SHL r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SHL, op(Register, 0), op(Constant, 5), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 10
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestSHR(t *testing.T) {
	//    SHR r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SHR, op(Register, 0), op(Constant, 5), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 2
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestAND(t *testing.T) {
	//    AND r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.AND, op(Register, 0), op(Constant, 5), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestOR1(t *testing.T) {
	//     OR r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.OR, op(Register, 0), op(Constant, 5), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestOR2(t *testing.T) {
	//     OR r0, $4, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.OR, op(Register, 0), op(Constant, 4), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestXOR1(t *testing.T) {
	//    XOR r0, $4, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.XOR, op(Register, 0), op(Constant, 4), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 5
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestXOR2(t *testing.T) {
	//    XOR r0, $5, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.XOR, op(Register, 0), op(Constant, 5), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 4
	ct.want[RIP] = 9
	runTest(t, ct)
}

func TestABS1(t *testing.T) {
	//    ABS r0, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ABS, op(Register, 0), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 6
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestABS2(t *testing.T) {
	//   ABS i8 r0, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.ABS, op(Register, 0, arch.I8), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[R0] = 1 << 8
	ct.want[RIP] = 6
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestPOW1(t *testing.T) {
	//    POW r0, $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.POW, op(Register, 0), op(Constant, 2), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 4
	ct.want[RIP] = 9
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestPOW2(t *testing.T) {
	//    POW r0, $0x7fff, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.POW, op(Register, 0), op(Constant, 0x7fff), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[R0] = 1
	ct.want[RIP] = 9
	ct.want[RST] = 2
	runTest(t, ct)
}

func TestCEQ1(t *testing.T) {
	//    CEQ $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(Constant, 2), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCEQ81(t *testing.T) {
	//   CEQ i8 $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(Constant, 2, arch.I8), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCEQ2(t *testing.T) {
	//    CEQ $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CEQ, op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCNE1(t *testing.T) {
	//    CNE $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CNE, op(Constant, 2), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCNE2(t *testing.T) {
	//    CNE $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CNE, op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGT1(t *testing.T) {
	//    CGT $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGT, op(Constant, 2), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGT2(t *testing.T) {
	//    CGT $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGT, op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCGT3(t *testing.T) {
	//    CGT $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGT, op(Constant, 2), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGE1(t *testing.T) {
	//    CGE $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGE, op(Constant, 2), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCGE2(t *testing.T) {
	//    CGE $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CGE, op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLT1(t *testing.T) {
	//    CLT $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLT, op(Constant, 2), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLT2(t *testing.T) {
	//    CLT $1, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLT, op(Constant, 1), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCLT3(t *testing.T) {
	//    CLT $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLT, op(Constant, 2), op(Constant, 1))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 0
	runTest(t, ct)
}

func TestCLE1(t *testing.T) {
	//    CLE $2, $2
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLE, op(Constant, 2), op(Constant, 2))
	ct.emit(arch.HALT)

	ct.want[RIP] = 8
	ct.want[RST] = 1
	runTest(t, ct)
}

func TestCLE2(t *testing.T) {
	//    CLE $2, $1
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.CLE, op(Constant, 2), op(Constant, 1))
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
	ct.emit(arch.JMP, op(Constant, 10))
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 456))
	ct.emit(arch.HALT)

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
	ct.emit(arch.CEQ, op(Constant, 1), op(Constant, 2))
	ct.emit(arch.JEZ, op(Constant, 17))
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 456))
	ct.emit(arch.HALT)

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
	ct.emit(arch.CEQ, op(Constant, 1), op(Constant, 1))
	ct.emit(arch.JEZ, op(Constant, 17))
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 456))
	ct.emit(arch.HALT)

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
	ct.emit(arch.CEQ, op(Constant, 1), op(Constant, 2))
	ct.emit(arch.JNZ, op(Constant, 17))
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 456))
	ct.emit(arch.HALT)

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
	ct.emit(arch.CEQ, op(Constant, 1), op(Constant, 1))
	ct.emit(arch.JNZ, op(Constant, 17))
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 456))
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
	ct.emit(arch.CALL, op(Constant, 5))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.RET)

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
	ct.emit(arch.CEQ, op(Constant, 0), op(Constant, 1))
	ct.emit(arch.CLEZ, op(Constant, 12))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.RET)

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
	ct.emit(arch.CEQ, op(Constant, 0), op(Constant, 0))
	ct.emit(arch.CLEZ, op(Constant, 12))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.RET)

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
	ct.emit(arch.CEQ, op(Constant, 0), op(Constant, 1))
	ct.emit(arch.CLNZ, op(Constant, 12))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.RET)

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
	ct.emit(arch.CEQ, op(Constant, 0), op(Constant, 0))
	ct.emit(arch.CLNZ, op(Constant, 12))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.RET)

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
	ct.emit(arch.HWA, op(Register, 0), op(Constant, testID.Manufacturer()), op(Constant, testID.Serial()))
	ct.emit(arch.JEZ, op(Constant, 18))
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 456))
	ct.emit(arch.HALT)

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
	ct.emit(arch.HWA, op(Register, 0), op(Constant, 0), op(Constant, 0))
	ct.emit(arch.JEZ, op(Constant, 18))
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 123))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 456))
	ct.emit(arch.HALT)

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
	ct.emit(arch.HWA, op(Register, 0), op(Constant, testID.Manufacturer()), op(Constant, testID.Serial()))
	ct.emit(arch.JEZ, op(Constant, 15))
	ct.emit(arch.INT, op(Register, 0))
	ct.emit(arch.HALT)
	ct.emit(arch.MOV, op(Register, 0), op(Constant, 456))
	ct.emit(arch.HALT)

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
	ct.emit(arch.SEED, op(Constant, 0))
	ct.emit(arch.RNG, op(Register, 0), op(Constant, 0), op(Constant, 10))
	ct.emit(arch.SEED, op(Constant, 0))
	ct.emit(arch.RNG, op(Register, 1), op(Constant, 0), op(Constant, 10))
	ct.emit(arch.HALT)

	rng := rand.New(rand.NewSource(0))
	ct.want[R0] = rng.Intn(10)

	rng = rand.New(rand.NewSource(0))
	ct.want[R1] = rng.Intn(10)

	ct.want[RIP] = 25
	runTest(t, ct)
}

func TestSEED8RNG8(t *testing.T) {
	//   SEED $0
	//   RNG u8 r0, $0, $10
	//   SEED $0
	//   RNG u8 r1, $0, $10
	//   HALT

	ct := newCodeTest()
	ct.emit(arch.SEED, op(Constant, 0))
	ct.emit(arch.RNG, op(Register, 0, arch.U8), op(Constant, 0), op(Constant, 10))
	ct.emit(arch.SEED, op(Constant, 0))
	ct.emit(arch.RNG, op(Register, 1, arch.U8), op(Constant, 0), op(Constant, 10))
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

func (ct *codeTest) emit(opcode byte, ops ...[3]int) {
	w := &ct.program
	w.WriteByte(opcode)

	for _, v := range ops {
		attr := byte(v[0]&0x3)<<6 | byte(v[1]&0x3)<<4

		switch AddressMode(v[0]) {
		case Constant, Address:
			w.WriteByte(attr)
			w.WriteByte(byte(v[2] >> 8))
			w.WriteByte(byte(v[2]))

		case Register:
			w.WriteByte(attr | byte(v[2]&0xf))
		}
	}
}

func op(mode AddressMode, value int, typ ...int) [3]int {
	if len(typ) > 0 {
		return [...]int{int(mode), typ[0], value}
	}
	return [...]int{int(mode), arch.I16, value}
}

func trace(i *Instruction) {
	name, ok := arch.Name(i.Opcode)
	if !ok {
		name = fmt.Sprintf("%02x", i.Opcode)
	}

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
