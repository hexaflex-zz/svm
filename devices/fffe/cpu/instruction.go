package cpu

import (
	"github.com/hexaflex/svm/arch"
)

// Instruction defines decoded instruction data.
type Instruction struct {
	IP     int        // Instruction address.
	Opcode int        // Instruction opcode.
	Args   [3]Operand // Operand A, B and C.
	Wide   bool       // Does the instruction operate on 16-bit values?
	Signed bool       // Does the instruction operate on signed or unsigned values?
}

// Decode decodes the next instruction from the given memory bank.
func (i *Instruction) Decode(m Memory) error {
	i.IP = m.U16(RIP)

	b, err := m.next8()
	if err != nil {
		return err
	}

	i.Opcode = b & 0x3f
	i.Wide = (b>>7)&1 == 1
	i.Signed = (b>>6)&1 == 0

	argc := arch.Argc(i.Opcode)
	if argc < 0 {
		return NewError(i, "unknown opcode %02x", i.Opcode)
	}

	for j := 0; j < argc; j++ {
		if err := i.Args[j].Decode(m, i.Wide, i.Signed); err != nil {
			return err
		}
	}

	return nil
}

// Operand defines decoded instruction operand data.
type Operand struct {
	Address int         // Optional address representation.
	Value   int         // Dereferenced value behind the address, if applicable. Otherwise same as Address.
	Mode    AddressMode // Address mode.
}

// Decode decodes the next instruction operand from the given memory bank.
// isWide determines if the operands are to be treated as 8- or 16 bit values.
// signed determines if the operands are signed or unsigned values.
func (op *Operand) Decode(m Memory, isWide, signed bool) error {
	b, err := m.next8()
	if err != nil {
		return err
	}

	op.Mode = AddressMode(b >> 6)

	switch op.Mode {
	case Constant:
		v, err := m.next16()
		if err != nil {
			return err
		}
		if signed {
			op.Value = int(int8(v))
			if isWide {
				op.Value = int(int16(v))
			}
		} else {
			op.Value = int(uint8(v))
			if isWide {
				op.Value = int(uint16(v))
			}
		}
		op.Address = op.Value

	case Address:
		op.Address, err = m.next16()
		if err != nil {
			return err
		}
		if signed {
			op.Value = m.I8(op.Address)
			if isWide {
				op.Value = m.I16(op.Address)
			}
		} else {
			op.Value = m.U8(op.Address)
			if isWide {
				op.Value = m.U16(op.Address)
			}
		}

	case Register:
		op.Address = (b&0x3f)*2 + UserMemoryCapacity
		if signed {
			op.Value = m.I8(op.Address)
			if isWide {
				op.Value = m.I16(op.Address)
			}
		} else {
			op.Value = m.U8(op.Address)
			if isWide {
				op.Value = m.U16(op.Address)
			}
		}
	}

	return nil
}

// AddressMode defines instruction operand address modes.
type AddressMode byte

// Known address modes.
const (
	Constant AddressMode = iota // Operand is a constant numeric value.
	Address                     // Operand is a memory address.
	Register                    // Operand is a register index.
)
