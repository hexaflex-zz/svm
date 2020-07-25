package vm

import (
	"fmt"
	"strings"
)

// ErrorSet defines a list of one or more errors and is itself an error.
type ErrorSet []error

func (e ErrorSet) Len() int {
	return len(e)
}

func (e *ErrorSet) Append(args ...error) {
	*e = append(*e, args...)
}

func (e ErrorSet) Error() string {
	var sb strings.Builder
	for _, err := range e {
		sb.WriteString(err.Error() + "\n")
	}
	return sb.String()
}

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
