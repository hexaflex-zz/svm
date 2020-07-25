package cpu

import "fmt"

// Error defines a runtime error.
type Error struct {
	*Instruction
	Msg string
}

// NewError creates a new, formatted error message for the given instruction.
func NewError(instr *Instruction, f string, argv ...interface{}) *Error {
	return &Error{
		Instruction: instr,
		Msg:         fmt.Sprintf(f, argv...),
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%04x: %s", e.IP, e.Msg)
}
