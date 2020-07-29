// Package ar defines the compiled archive. It contains program code and
// optional debug symbols for a complete SVM program.
package ar

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Archive defines a complete, compiled archive.
type Archive struct {
	Debug        Debug  // Optional debug symbols.
	Instructions []byte // Compiled code.
}

// New creates a new, empty archive.
func New() *Archive {
	return &Archive{}
}

// String returns a human-readable dump of the archive's contents.
func (a *Archive) String() string {
	var sb strings.Builder

	if len(a.Debug.Files) > 0 {
		fmt.Fprintf(&sb, "Source files (%d):\n", len(a.Debug.Files))
		for i, v := range a.Debug.Files {
			fmt.Fprintf(&sb, " %d: %s\n", i, v)
		}

		fmt.Fprintf(&sb, "Debug symbols (%d):\n", len(a.Debug.Symbols))
		for _, v := range a.Debug.Symbols {
			fmt.Fprintf(&sb, " %04x: File: %d, Line: %d, Col: %d Flags: %02x\n",
				v.Address, v.File, v.Line, v.Col, v.Flags)
		}
	}

	if len(a.Instructions) > 0 {
		fmt.Fprintf(&sb, "Instructions:\n")
		fmt.Fprintf(&sb, "%s\n", hex.Dump(a.Instructions))
	}

	return sb.String()
}
