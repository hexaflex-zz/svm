// Package cpu implements the SVM CPU
package cpu

import (
	"errors"
	"io"
	"log"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/hexaflex/svm/arch"
	"github.com/hexaflex/svm/devices"
)

// TraceFunc represents a callback handler for debug trace output.
type TraceFunc func(*Instruction)

type CPU struct {
	devices     devices.Map // Connected peripherals.
	trace       TraceFunc   // Handler for debug trace output.
	memory      Memory      // System memory.
	instr       Instruction // Decoded instruction data.
	rng         *rand.Rand  // Random number generator.
	initialized uint32      // Is there a valid program loaded?
}

// New creates a new CPU for the given program.
// Optionally with the given debug trace handler.
func New(trace TraceFunc) *CPU {
	if trace == nil {
		trace = func(*Instruction) { /* nop */ }
	}

	return &CPU{
		trace:  trace,
		memory: make(Memory, MemoryCapacity),
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *CPU) Id() devices.Id {
	return devices.NewId(0xfffe, 0x0001)
}

// Connect connects the given hardware peripheral to the system.
// Returns false if the given device type is already connected.
func (c *CPU) Connect(dev devices.Device) bool {
	return c.devices.Connect(dev)
}

// Startup loads the given program and initializes internal resources.
// Returns an error if a program is already loaded. Use Shutdown() first.
func (c *CPU) Startup(program []byte, entrypoint int) error {
	if !atomic.CompareAndSwapUint32(&c.initialized, 0, 1) {
		return errors.New(c.Id().String() + " program is already loaded")
	}

	if len(program) > UserMemoryCapacity {
		return errors.New(c.Id().String() + " program exceeds memory capacity")
	}

	log.Println(c.Id(), "startup")
	copy(c.memory, program)

	c.memory.SetU16(RIP, entrypoint)
	c.memory.SetU16(RSP, UserMemoryCapacity-2)
	c.memory.SetU8(RST, 0)
	return c.devices.Startup()
}

// Shutdown cleans up internal resources.
func (c *CPU) Shutdown() error {
	if !atomic.CompareAndSwapUint32(&c.initialized, 1, 0) {
		return nil
	}
	log.Println(c.Id(), "shutdown")
	return c.devices.Shutdown()
}

// Step performs a single execution step.
// Returns io.EOF if the program has reached its end.
func (c *CPU) Step() error {
	if atomic.LoadUint32(&c.initialized) == 0 {
		return io.EOF
	}

	mem := c.memory
	instr := &c.instr
	args := instr.Args[:]

	if err := instr.Decode(mem); err != nil {
		return err
	}

	c.trace(instr)

	switch instr.Opcode {
	case arch.MOV:
		va := args[0].Address
		vb := args[1].Value
		mem.SetI16(va, vb)
	case arch.PUSH:
		rsp := mem.U16(RSP)
		mem.SetU16(RSP, rsp-2)
		mem.SetU16(rsp, args[0].Value)
	case arch.POP:
		rsp := mem.U16(RSP)
		va := mem.I16(rsp + 2)
		mem.SetU16(RSP, rsp+2)
		mem.SetU16(args[0].Address, va)
	case arch.RNG:
		va := args[0].Address
		vb := int(uint16(args[1].Value))
		vc := int(uint16(args[2].Value))
		if vc-vb < 0 {
			mem.SetRSTOverflow(true)
		} else {
			mem.SetRSTOverflow(false)
			mem.SetI16(va, vb+c.rng.Intn(vc-vb))
		}
	case arch.SEED:
		va := args[0].Value
		c.rng = rand.New(rand.NewSource(int64(va)))

	case arch.ADD:
		v := args[1].Value + args[2].Value
		mem.SetRSTOverflow(v < -0x7fff || v > 0x7fff)
		mem.SetU16(args[0].Address, v)
	case arch.SUB:
		v := args[1].Value - args[2].Value
		mem.SetRSTOverflow(v < -0x7fff || v > 0x7fff)
		mem.SetU16(args[0].Address, v)
	case arch.MUL:
		v := args[1].Value * args[2].Value
		mem.SetRSTOverflow(v < -0x7fff || v > 0x7fff)
		mem.SetU16(args[0].Address, v)
	case arch.DIV:
		if args[2].Value == 0 {
			mem.SetRSTDivideByZero(true)
		} else {
			v := args[1].Value / args[2].Value
			mem.SetU16(args[0].Address, v)
			mem.SetRSTDivideByZero(false)
		}
	case arch.MOD:
		if args[2].Value == 0 {
			mem.SetRSTDivideByZero(true)
		} else {
			v := args[1].Value % args[2].Value
			mem.SetU16(args[0].Address, v)
			mem.SetRSTDivideByZero(false)
		}
	case arch.SHL:
		v := args[1].Value << uint(args[2].Value)
		mem.SetU16(args[0].Address, v)
	case arch.SHR:
		v := args[1].Value >> uint(args[2].Value)
		mem.SetU16(args[0].Address, v)
	case arch.AND:
		v := args[1].Value & args[2].Value
		mem.SetU16(args[0].Address, v)
	case arch.OR:
		v := args[1].Value | args[2].Value
		mem.SetU16(args[0].Address, v)
	case arch.XOR:
		v := args[1].Value ^ args[2].Value
		mem.SetU16(args[0].Address, v)
	case arch.ABS:
		v := int(math.Abs(float64(args[1].Value)))
		mem.SetU16(args[0].Address, v)
	case arch.POW:
		va := float64(args[1].Value)
		vb := float64(args[2].Value)
		vc := int(math.Pow(va, vb))
		mem.SetRSTOverflow(vc < -0x7fff || vc > 0x7fff)
		mem.SetU16(args[0].Address, vc)

	case arch.CEQ:
		mem.SetRSTCompare(args[0].Value == args[1].Value)
	case arch.CNE:
		mem.SetRSTCompare(args[0].Value != args[1].Value)
	case arch.CGT:
		mem.SetRSTCompare(args[0].Value > args[1].Value)
	case arch.CGE:
		mem.SetRSTCompare(args[0].Value >= args[1].Value)
	case arch.CLT:
		mem.SetRSTCompare(args[0].Value < args[1].Value)
	case arch.CLE:
		mem.SetRSTCompare(args[0].Value <= args[1].Value)

	case arch.JMP:
		mem.SetU16(RIP, args[0].Value)
	case arch.JEZ:
		if !mem.RSTCompare() {
			mem.SetU16(RIP, args[0].Value)
		}
	case arch.JNZ:
		if mem.RSTCompare() {
			mem.SetU16(RIP, args[0].Value)
		}
	case arch.CALL:
		rsp := mem.U16(RSP)
		mem.SetU16(RSP, rsp-2)
		mem.SetU16(rsp, mem.U16(RIP))
		mem.SetU16(RIP, args[0].Value)
	case arch.CLEZ:
		if !mem.RSTCompare() {
			rsp := mem.U16(RSP)
			mem.SetU16(RSP, rsp-2)
			mem.SetU16(rsp, mem.U16(RIP))
			mem.SetU16(RIP, args[0].Value)
		}
	case arch.CLNZ:
		if mem.RSTCompare() {
			rsp := mem.U16(RSP)
			mem.SetU16(RSP, rsp-2)
			mem.SetU16(rsp, mem.U16(RIP))
			mem.SetU16(RIP, args[0].Value)
		}

	case arch.RET:
		rsp := mem.U16(RSP)
		va := mem.U16(rsp + 2)
		mem.SetU16(RSP, rsp+2)
		mem.SetU16(RIP, va)

	case arch.HWA:
		id := devices.NewId(args[1].Value, args[2].Value)
		if index := c.devices.Find(id); index == -1 {
			mem.SetRSTCompare(false)
		} else {
			mem.SetRSTCompare(true)
			mem.SetU16(args[0].Address, index)
		}
	case arch.INT:
		if !c.devices.Int(args[0].Value, mem) {
			return NewError(instr, "invalid device index %d", args[0].Value)
		}

	case arch.NOP:
		/* nop */
	case arch.HALT:
		return io.EOF
	}

	return nil
}
