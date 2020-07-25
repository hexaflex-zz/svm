package arch

import "strings"

// IsRegister returns true if the given name represents a known register.
func IsRegister(name string) bool {
	return RegisterIndex(name) > -1
}

// RegisterIndex returns the index for the given register.
// Returns -1 if the name is not recognized.
func RegisterIndex(name string) int {
	switch strings.ToLower(name) {
	case "r0":
		return 0
	case "r1":
		return 1
	case "r2":
		return 2
	case "r3":
		return 3
	case "rsp":
		return 4
	case "rip":
		return 5
	case "rst":
		return 6
	}
	return -1
}

// RegiserName returns the name associated with the given register index.
// Returns "" if the index is not recognized.
func RegiserName(n int) string {
	switch n {
	case 0:
		return "R0"
	case 1:
		return "R1"
	case 2:
		return "R2"
	case 3:
		return "R3"
	case 4:
		return "RSP"
	case 5:
		return "RIP"
	case 6:
		return "RST"
	}
	return ""
}
