package devices

// Memory defines the system's memory bank.
type Memory interface {
	// I8 defines a signed 8-bit value at the given address.
	I8(addr int) int
	SetI8(addr, value int)

	// U8 defines an unsigned 8-bit value at the given address.
	U8(addr int) int
	SetU8(addr, value int)

	// I16 defines a signed 16-bit value at the given address.
	I16(addr int) int
	SetI16(addr, value int)

	// U16 defines an unsigned 16-bit value at the given address.
	U16(addr int) int
	SetU16(addr, value int)

	// Write writes len(p) bytes from p into memory, starting at the given address.
	Write(address int, p []byte)

	// Read reads len(p) bytes from memory into p, starting at the given address.
	Read(address int, p []byte)

	// RSTCompare defines the state of the RST/compare flag.
	RSTCompare() bool
	SetRSTCompare(v bool)

	// RSTOverflow defines the state of the RST/overflow flag.
	RSTOverflow() bool
	SetRSTOverflow(v bool)

	// RSTOverflow defines the state of the RST/divide-by-zero flag.
	RSTDivideByZero() bool
	SetRSTDivideByZero(v bool)
}
