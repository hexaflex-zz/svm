package devices

import "fmt"

// ID identifies a device.
// The upper 16 bits hold the device manufacturer id.
// The lower 16 bits hold the device serial number.
type ID uint32

// NewId creates a new id with the given components.
func NewID(manufacturer, serial int) ID {
	return ID(manufacturer&0xffff)<<16 | ID(serial&0xffff)
}

// Manufacturer returns the manufacturer component of the Id.
func (id ID) Manufacturer() int {
	return int(id>>16) & 0xffff
}

// Serial returns the device serial number component of the Id.
func (id ID) Serial() int {
	return int(id) & 0xffff
}

func (id ID) String() string {
	return fmt.Sprintf("%04x:%04x", id.Manufacturer(), id.Serial())
}
