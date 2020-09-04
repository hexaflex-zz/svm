package cpu

import (
	"github.com/hexaflex/svm/arch"
)

// Instruction defines decoded instruction data.
type Instruction struct {
	IP     int        // Instruction address.
	Opcode int        // Instruction opcode.
	Args   [3]Operand // Operand A, B and C.
}

// Decode decodes the next instruction from the given memory bank.
func (i *Instruction) Decode(m Memory) error {
	i.IP = m.U16(RIP)

	b, err := m.next8()
	if err != nil {
		return err
	}

	i.Opcode = b

	argc := arch.Argc(i.Opcode)
	if argc < 0 {
		return NewError(i, "unknown opcode %02x", i.Opcode)
	}

	for j := 0; j < argc; j++ {
		if err := i.Args[j].Decode(m); err != nil {
			return err
		}
	}

	return nil
}

// Operand defines decoded instruction operand data.
type Operand struct {
	Address int              // Optional address representation.
	Value   int              // Dereferenced value behind the address, if applicable. Otherwise same as Address.
	Mode    arch.AddressMode // Address mode.
	Type    arch.Type        // Operand data type.
}

// Decode decodes the next instruction operand from the given memory bank.
func (op *Operand) Decode(m Memory) error {
	b, err := m.next8()
	if err != nil {
		return err
	}

	op.Mode = arch.AddressMode(b>>6) & 0x3
	op.Type = arch.Type(b>>4) & 0x3

	switch op.Mode {
	case arch.ImmediateConstant:
		v, err := m.next16()
		if err != nil {
			return err
		}

		switch op.Type {
		case arch.U8:
			op.Value = int(uint8(v))
		case arch.U16:
			op.Value = int(uint16(v))
		case arch.I8:
			op.Value = int(int8(v))
		case arch.I16:
			op.Value = int(int16(v))
		}

		op.Address = op.Value

	case arch.IndirectConstant:
		op.Address, err = m.next16()
		if err != nil {
			return err
		}
		op.readMem(m)

	case arch.ImmediateRegister:
		op.Address = (b&0xf)*2 + UserMemoryCapacity
		op.readMem(m)

	case arch.IndirectRegister:
		op.Address = m.U16((b&0xf)*2 + UserMemoryCapacity)
		op.readMem(m)
	}

	return nil
}

func (op *Operand) readMem(m Memory) {
	switch op.Type {
	case arch.U8:
		op.Value = m.U8(op.Address)
	case arch.U16:
		op.Value = m.U16(op.Address)
	case arch.I8:
		op.Value = m.I8(op.Address)
	case arch.I16:
		op.Value = m.I16(op.Address)
	}
}
