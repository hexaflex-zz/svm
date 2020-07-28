package devices

// Memory defines the system's memory bank.
type Memory interface {
	// SetI8 sets the 8-bit value at the given address.
	SetI8(addr, value int)

	// I8 returns the 8-bit value at the given address.
	I8(addr int) int

	// SetU8 sets the 8-bit value at the given address.
	SetU8(addr, value int)

	// U8 returns the 8-bit value at the given address.
	U8(addr int) int

	// SetI16 sets the 16-bit value at the given address.
	SetI16(addr, value int)

	// I16 returns the 16-bit value at the given address.
	I16(addr int) int

	// SetU16 sets the 16-bit value at the given address.
	SetU16(addr, value int)

	// U16 returns the 16-bit value at the given address.
	U16(addr int) int

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
