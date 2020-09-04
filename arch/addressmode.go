package arch

// AddressMode defines instruction operand address modes.
type AddressMode byte

// Known address modes.
const (
	ImmediateConstant AddressMode = 0 // x = 123
	IndirectConstant  AddressMode = 1 // x = mem[123]
	ImmediateRegister AddressMode = 2 // x = r0
	IndirectRegister  AddressMode = 3 // x = mem[r0]
)
