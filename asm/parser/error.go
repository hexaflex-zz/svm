package parser

import (
	"fmt"
)

// Error defines a parse error with source context.
type Error struct {
	Pos Position
	Msg string
}

// NewError creates a new, formatted error message with the given source context.
func NewError(pos Position, f string, argv ...interface{}) *Error {
	return &Error{
		Pos: pos,
		Msg: fmt.Sprintf(f, argv...),
	}
}

func (e *Error) Error() string {
	return e.Pos.String() + " " + e.Msg
}
