package arch

import (
	"fmt"
	"strings"
)

// Type represents one of the known type descriptors.
type Type int

// Known type descriptors
const (
	U8  Type = 0
	U16 Type = 1
	I8  Type = 2
	I16 Type = 3
)

// TypeFromName returns a type descriptor matching the given name.
// Returns -1 if no match was found.
func TypeFromName(name string) Type {
	switch strings.ToUpper(name) {
	case "U8":
		return U8
	case "U16":
		return U16
	case "I8":
		return I8
	case "I16":
		return I16
	}
	return -1
}

// Limits returns the minimum and maximum values for the given type.
// These can be used to perform type-appropriate overflow checks.
// Returns 0,0 if the type is not recognized.
func (t Type) Limits() (int, int) {
	switch t {
	case U8:
		return 0, 0xff
	case U16:
		return 0, 0xffff
	case I8:
		return -0x7f, 0x7f
	case I16:
		return -0x7fff, 0x7fff
	}
	return 0, 0
}

// Name returns the string representation of the given type descriptor.
func (t Type) Name() string {
	switch t {
	case U8:
		return "U8"
	case U16:
		return "U16"
	case I8:
		return "I8"
	case I16:
		return "I16"
	}
	return fmt.Sprintf("Type(%02x)", int(t))
}
