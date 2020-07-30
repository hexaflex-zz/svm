package cpu

import "io"

const (
	UserMemoryCapacity = 0x10000                               // Size of user space.
	RegisterCapacity   = 8 * 2                                 // Space occupied by registers.
	MemoryCapacity     = UserMemoryCapacity + RegisterCapacity // Total memory capacity: userspace + registers.
)

const (
	R0  = UserMemoryCapacity // Address for general purpose register R0.
	R1  = R0 + 2             // Address for general purpose register R1.
	R2  = R1 + 2             // Address for general purpose register R2.
	R3  = R2 + 2             // Address for general purpose register R3.
	RSP = R3 + 2             // Address for stack pointer register.
	RIP = RSP + 2            // Address for instruction pointer register.
	RIA = RIP + 2            // Address for Interrupt Address register.
	RST = RIA + 2            // Address for status register.
)

// Memory defines the system's memory bank.
type Memory []byte

// SetI8 sets the 8-bit value at the given address.
func (m Memory) SetI8(addr, value int) {
	m[addr] = byte(int8(value))
}

// I8 returns the 8-bit value at the given address.
func (m Memory) I8(addr int) int {
	return int(int8(m[addr]))
}

// SetU8 sets the 8-bit value at the given address.
func (m Memory) SetU8(addr, value int) {
	m[addr] = byte(value)
}

// U8 returns the 8-bit value at the given address.
func (m Memory) U8(addr int) int {
	return int(m[addr])
}

// SetI16 sets the 16-bit value at the given address.
func (m Memory) SetI16(addr, value int) {
	m[addr] = byte(int16(value) >> 8)
	m[addr+1] = byte(int16(value))
}

// I16 returns the 16-bit value at the given address.
func (m Memory) I16(addr int) int {
	return int(int16(m[addr])<<8 | int16(m[addr+1]))
}

// SetU16 sets the 16-bit value at the given address.
func (m Memory) SetU16(addr, value int) {
	m[addr] = byte(value >> 8)
	m[addr+1] = byte(value)
}

// U16 returns the 16-bit value at the given address.
func (m Memory) U16(addr int) int {
	return int(uint16(m[addr])<<8 | uint16(m[addr+1]))
}

// Write writes len(p) bytes from p into memory, starting at the given address.
func (m Memory) Write(address int, p []byte) {
	copy(m[address:], p)
}

// Read reads len(p) bytes from memory into p, starting at the given address.
func (m Memory) Read(address int, p []byte) {
	copy(p, m[address:])
}

// RSTCompare defines the state of the RST/compare flag.
func (m Memory) RSTCompare() bool     { return m.rst(1) }
func (m Memory) SetRSTCompare(v bool) { m.setRST(1, v) }

// RSTOverflow defines the state of the RST/overflow flag.
func (m Memory) RSTOverflow() bool     { return m.rst(2) }
func (m Memory) SetRSTOverflow(v bool) { m.setRST(2, v) }

// RSTOverflow defines the state of the RST/divide-by-zero flag.
func (m Memory) RSTDivideByZero() bool     { return m.rst(4) }
func (m Memory) SetRSTDivideByZero(v bool) { m.setRST(4, v) }

// RSTCompare returns the state of the given RST flag.
func (m Memory) rst(flag int) bool {
	return int(m[RST])&flag == flag
}

// rst sets or unsets the given RST flag.
func (m Memory) setRST(flag int, v bool) {
	if v {
		m[RST] |= byte(flag)
	} else {
		m[RST] &^= byte(flag)
	}
}

// next8 reads the next byte at the current instruction pointer value and increments
// said instruction pointer.
func (m Memory) next8() (int, error) {
	rip := m.U16(RIP)
	m.SetU16(RIP, rip+1)

	if rip < 0 || rip >= UserMemoryCapacity {
		return 0, io.EOF
	}

	return int(m[rip]), nil
}

// nnext16ext reads the next 16 bit value at the current instruction pointer value and increments
// said instruction pointer.
func (m Memory) next16() (int, error) {
	rip := m.U16(RIP)
	m.SetU16(RIP, rip+2)

	if rip < 0 || rip >= UserMemoryCapacity-1 {
		return 0, io.EOF
	}

	return m.U16(rip), nil
}
