package ar

import (
	"fmt"
	"io"
	"runtime"

	"github.com/pkg/errors"
)

// DebugFlags defines debug bitflags.
type DebugFlags byte

// Known debug bit flags.
const (
	// When an instruction with this flag is encountered
	// by the VM, the VM pauses execution.
	Breakpoint DebugFlags = 1 << iota
)

// Debug defines any debug data stored in an archive.
type Debug struct {
	Files   []string    // File names associated with the source that makes up this archive. Only set when there are debug symbols.
	Symbols []DebugData // Per-instruction source context.
}

// Clear empties all data.
func (d *Debug) Clear() {
	d.Files = nil
	d.Symbols = nil
}

// Find returns the debug data associated with the given address.
// Returns nil if there is none.
func (d *Debug) Find(addr int) *DebugData {
	for i := range d.Symbols {
		if d.Symbols[i].Address == addr {
			return &d.Symbols[i]
		}
	}
	return nil
}

// Load reads debug data from the given stream.
func (d *Debug) Load(r io.Reader) (err error) {
	defer recoverOnPanic(&err)

	d.Files = make([]string, readU8(r))
	for i := range d.Files {
		d.Files[i] = string(readBytes(r))
	}

	d.Symbols = make([]DebugData, readU16(r))
	for i := range d.Symbols {
		d.Symbols[i].read(r)
	}

	return
}

// Save writes debug data to the given stream.
func (d *Debug) Save(w io.Writer) (err error) {
	defer recoverOnPanic(&err)

	writeU8(w, uint8(len(d.Files)))
	for i := range d.Files {
		writeBytes(w, []byte(d.Files[i]))
	}

	writeU16(w, uint16(len(d.Symbols)))
	for i := range d.Symbols {
		d.Symbols[i].write(w)
	}

	return
}

// DebugData defines one set of debug symbols.
type DebugData struct {
	Address int        // Address for which this debug data is defined.
	File    int        // Index into list of file paths in which symbol was defined.
	Line    int        // Line number at which symbol was defined.
	Col     int        // Column number at which symbol was defined.
	Offset  int        // Byte offset at which symbol was defined.
	Flags   DebugFlags // Any debug flags defined for this entry.
}

func (d *DebugData) read(r io.Reader) {
	d.Address = int(readU16(r))
	d.File = int(readU8(r))
	d.Line = int(readU16(r))
	d.Col = int(readU16(r))
	d.Offset = int(readU32(r))
	d.Flags = DebugFlags(readU8(r))
}

func (d *DebugData) write(w io.Writer) {
	writeU16(w, uint16(d.Address))
	writeU8(w, uint8(d.File))
	writeU16(w, uint16(d.Line))
	writeU16(w, uint16(d.Col))
	writeU32(w, uint32(d.Offset))
	writeU8(w, uint8(d.Flags))
}

func recoverOnPanic(err *error) {
	x := recover()
	if x == nil {
		return
	}

	switch tx := x.(type) {
	case runtime.Error:
		panic(tx)
	case error:
		*err = errors.Wrapf(tx, "ar")
	default:
		*err = fmt.Errorf("ar: %v", tx)
	}
}
