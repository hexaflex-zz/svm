package arch

import (
	"strings"
)

// Known type descriptors
const (
	U8  = 0
	U16 = 1
	I8  = 2
	I16 = 3
)

// Type returns a type descriptor matching the given name.
// Returns -1 if no match was found.
func Type(name string) int {
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

// TypeName returns the string representation of the given type descriptor.
func TypeName(n int) string {
	switch n {
	case U8:
		return "U8"
	case U16:
		return "U16"
	case I8:
		return "I8"
	case I16:
		return "I16"
	}
	return ""
}
