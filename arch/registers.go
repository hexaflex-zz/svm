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
	case "r4":
		return 4
	case "r5":
		return 5
	case "r6":
		return 6
	case "r7":
		return 7
	case "rsp":
		return 8
	case "rip":
		return 9
	case "ria":
		return 10
	case "rst":
		return 11
	}
	return -1
}

// RegisterName returns the name associated with the given register index.
// Returns "" if the index is not recognized.
func RegisterName(n int) string {
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
		return "R4"
	case 5:
		return "R5"
	case 6:
		return "R6"
	case 7:
		return "R7"
	case 8:
		return "RSP"
	case 9:
		return "RIP"
	case 10:
		return "RIA"
	case 11:
		return "RST"
	}
	return ""
}
