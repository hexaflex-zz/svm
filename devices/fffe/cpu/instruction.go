package cpu

import "github.com/hexaflex/svm/arch"

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
	Address int         // Optional address representation.
	Value   int         // Dereferenced value behind the address, if applicable. Otherwise same as Address.
	Mode    AddressMode // Address mode.
}

// Decode decodes the next instruction operand from the given memory bank.
func (op *Operand) Decode(m Memory) error {
	b, err := m.next8()
	if err != nil {
		return err
	}

	op.Mode = AddressMode(b >> 6)

	switch op.Mode {
	case Constant:
		op.Value, err = m.next16()
		if err != nil {
			return err
		}
		op.Value = int(int16(op.Value))
		op.Address = op.Value

	case Address:
		op.Address, err = m.next16()
		if err != nil {
			return err
		}
		op.Value = m.I16(op.Address)

	case Register:
		op.Address = (b&0x1f)*2 + UserMemoryCapacity
		op.Value = m.I16(op.Address)
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
