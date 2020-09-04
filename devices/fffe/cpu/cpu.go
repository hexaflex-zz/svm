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

// IntQueueCapacity capacity of the CPU interrupt queue.
const IntQueueCapacity = 32

// TraceFunc represents a callback handler for debug trace output.
type TraceFunc func(*Instruction)

// CPU implements the runtime.
type CPU struct {
	devices      devices.Map // Connected peripherals.
	trace        TraceFunc   // Handler for debug trace output.
	memory       Memory      // System memory.
	instr        Instruction // Decoded instruction data.
	rng          *rand.Rand  // Random number generator.
	intQueue     chan int    // Hardware interrupt queue.
	initialized  uint32      // Is there a valid program loaded?
	inIntHandler bool        // Is the CPU currently executing an interrupt handler?
}

// New creates a new CPU for the given program.
// Optionally with the given debug trace handler.
func New(trace TraceFunc) *CPU {
	if trace == nil {
		trace = func(*Instruction) { /* nop */ }
	}

	return &CPU{
		trace:    trace,
		memory:   make(Memory, MemoryCapacity),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		intQueue: make(chan int, IntQueueCapacity),
	}
}

// ID returns the cpu's device Id.
func (c *CPU) ID() devices.ID {
	return devices.NewID(0xfffe, 0x0001)
}

// Memory returns the cpu's internal memory bank.
func (c *CPU) Memory() Memory {
	return c.memory
}

// Connect connects the given hardware peripheral to the system.
// Returns false if the given device type is already connected.
func (c *CPU) Connect(dev devices.Device) bool {
	return c.devices.Connect(dev)
}

// Startup initializes the cpu and connected peripherals.
// Returns an error if a program is already loaded. Use Shutdown() first.
func (c *CPU) Startup() error {
	if !atomic.CompareAndSwapUint32(&c.initialized, 0, 1) {
		return errors.New(c.ID().String() + " program is already loaded")
	}

	log.Println(c.ID(), "startup")
	for i := range c.memory {
		c.memory[i] = 0
	}

	c.memory.SetU16(RIP, 0)
	c.memory.SetU16(RSP, UserMemoryCapacity-2)
	c.memory.SetU8(RST, 0)
	c.inIntHandler = false

	return c.devices.Startup(c.queueInterrupt)
}

// Shutdown cleans up internal resources.
func (c *CPU) Shutdown() error {
	if !atomic.CompareAndSwapUint32(&c.initialized, 1, 0) {
		return nil
	}
	log.Println(c.ID(), "shutdown")
	return c.devices.Shutdown()
}

// Step performs a single execution step.
// Returns io.EOF if the program has reached its end
// or no program is loaded.
func (c *CPU) Step() error {
	if atomic.LoadUint32(&c.initialized) == 0 {
		return io.EOF
	}

	c.checkIntQueue()

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
		setVal(mem, args[0].Type, va, vb)

	case arch.PUSH:
		c.push(args[0].Value)
	case arch.POP:
		setVal(mem, args[0].Type, args[0].Address, c.pop())
	case arch.RNG:
		va := args[0].Address
		vb := int(uint(args[1].Value))
		vc := int(uint(args[2].Value))
		if vc-vb < 0 {
			mem.SetRSTOverflow(true)
		} else {
			mem.SetRSTOverflow(false)
			setVal(mem, args[0].Type, va, vb+c.rng.Intn(vc-vb))
		}

	case arch.SEED:
		va := args[0].Value
		c.rng = rand.New(rand.NewSource(int64(va)))

	case arch.ADD:
		va := args[0].Address
		vb := args[1].Value + args[2].Value
		min, max := args[0].Type.Limits()
		mem.SetRSTOverflow(vb < min || vb > max)
		setVal(mem, args[0].Type, va, vb)
	case arch.SUB:
		va := args[0].Address
		vb := args[1].Value - args[2].Value
		min, max := args[0].Type.Limits()
		mem.SetRSTOverflow(vb < min || vb > max)
		setVal(mem, args[0].Type, va, vb)
	case arch.MUL:
		va := args[0].Address
		vb := args[1].Value * args[2].Value
		min, max := args[0].Type.Limits()
		mem.SetRSTOverflow(vb < min || vb > max)
		setVal(mem, args[0].Type, va, vb)
	case arch.DIV:
		if args[2].Value == 0 {
			mem.SetRSTDivideByZero(true)
		} else {
			va := args[0].Address
			vb := args[1].Value / args[2].Value
			setVal(mem, args[0].Type, va, vb)
			mem.SetRSTDivideByZero(false)
		}
	case arch.MOD:
		if args[2].Value == 0 {
			mem.SetRSTDivideByZero(true)
		} else {
			va := args[0].Address
			vb := args[1].Value % args[2].Value
			setVal(mem, args[0].Type, va, vb)
			mem.SetRSTDivideByZero(false)
		}
	case arch.SHL:
		va := args[0].Address
		vb := args[1].Value << uint(args[2].Value)
		setVal(mem, args[0].Type, va, vb)
	case arch.SHR:
		va := args[0].Address
		vb := args[1].Value >> uint(args[2].Value)
		setVal(mem, args[0].Type, va, vb)
	case arch.AND:
		va := args[0].Address
		vb := args[1].Value & args[2].Value
		setVal(mem, args[0].Type, va, vb)
	case arch.OR:
		va := args[0].Address
		vb := args[1].Value | args[2].Value
		setVal(mem, args[0].Type, va, vb)
	case arch.XOR:
		va := args[0].Address
		vb := args[1].Value ^ args[2].Value
		setVal(mem, args[0].Type, va, vb)
	case arch.ABS:
		va := args[0].Address
		vb := int(math.Abs(float64(args[1].Value)))
		setVal(mem, args[0].Type, va, vb)
	case arch.POW:
		va := args[0].Address
		vb := float64(args[1].Value)
		vc := float64(args[2].Value)
		vd := int(math.Pow(vb, vc))
		min, max := args[0].Type.Limits()
		mem.SetRSTOverflow(vd < min || vd > max)
		setVal(mem, args[0].Type, va, vd)

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
		mem.SetU16(RIP, c.pop())
	case arch.IRET:
		c.inIntHandler = false
		setVal(mem, args[0].Type, R0, c.pop())
		mem.SetU16(RIP, c.pop())

	case arch.HWA:
		id := devices.NewID(args[1].Value, args[2].Value)
		if index := c.devices.Find(id); index == -1 {
			mem.SetRSTCompare(false)
		} else {
			mem.SetRSTCompare(true)
			setVal(mem, args[0].Type, args[0].Address, index)
		}
	case arch.INT:
		if !c.devices.Int(args[0].Value, mem) {
			return NewError(instr, "invalid device index %d", args[0].Value)
		}

	case arch.WAIT:
		<-time.After(time.Millisecond * time.Duration(args[0].Value))
	case arch.NOP:
		/* nop */
	case arch.HALT:
		return io.EOF
	}

	return nil
}

// checkIntQueue checks if there are pending messages in the interrupt queue.
// If so, it hands control over to the interrupt handler defined in RIA.
func (c *CPU) checkIntQueue() {
	// If we are currently inside an interrupt handler, don't jump into another one.
	if c.inIntHandler {
		return
	}

	select {
	case msg := <-c.intQueue:
		mem := c.memory[:]
		ria := mem.U16(RIA)

		c.push(mem.U16(RIP))
		c.push(mem.U16(R0))

		mem.SetU16(R0, msg)
		mem.SetU16(RIP, ria)

		c.inIntHandler = true
	default:
	}
}

// queueInterrupt adds a new message to the interrupt queue, provided interrupts are enabled.
func (c *CPU) queueInterrupt(msg int) {
	if c.memory.U16(RIA) == 0 {
		return
	}

	select {
	case c.intQueue <- msg:
	default:
	}
}

// push pushes the given value onto the callstack and updates RSP.
func (c *CPU) push(value int) {
	mem := c.memory[:]
	rsp := mem.U16(RSP)
	mem.SetU16(RSP, rsp-2)
	mem.SetU16(rsp, value)
}

// pop returns the top value from the callstack and updates RSP.
func (c *CPU) pop() int {
	mem := c.memory[:]
	rsp := mem.U16(RSP)
	mem.SetU16(RSP, rsp+2)
	return mem.U16(rsp + 2)
}

// setVal sets the value at the given address, using the type-specific storage method.
func setVal(mem Memory, _type arch.Type, addr, value int) {
	switch _type {
	case arch.U8:
		mem.SetU8(addr, value)
	case arch.U16:
		mem.SetU16(addr, value)
	case arch.I8:
		mem.SetI8(addr, value)
	case arch.I16:
		mem.SetI16(addr, value)
	}
}
