package vm

import "fmt"

// Id identifies a device.
// The upper 16 bits hold the device manufacturer id.
// The lower 16 bits hold the device serial number.
type Id uint32

// NewId creates a new id with the given components.
func NewId(manufacturer, serial int) Id {
	return Id(manufacturer&0xffff)<<16 | Id(serial&0xffff)
}

func (id Id) Manufacturer() int {
	return int(id>>16) & 0xffff
}

func (id Id) Serial() int {
	return int(id) & 0xffff
}

func (id Id) String() string {
	return fmt.Sprintf("%04x:%04x", id.Manufacturer(), id.Serial())
}
