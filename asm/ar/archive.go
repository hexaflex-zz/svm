// Package ar defines the compiled archive type, as well as an encoder
// and decoder for its file format.
package ar

import (
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// Archive defines a complete, compiled archive.
//
// When passed to a linker along with any other associated archives,
// it can be turned into a compiled binary.
type Archive struct {
	Debug        Debug  // Optional debug symbols.
	Instructions []byte // Compiled code.
}

// New creates a new, empty archive.
func New() *Archive {
	return &Archive{}
}

// Load reads archive data from the given stream.
func (a *Archive) Load(r io.Reader) (err error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return errors.Wrapf(err, "ar: invalid archive format")
	}

	defer gz.Close()
	defer recoverOnPanic(&err)

	a.Debug.read(gz)
	a.Instructions = readBytes(gz)
	return
}

// Save writes archive data to the given stream.
func (a *Archive) Save(w io.Writer) (err error) {
	defer recoverOnPanic(&err)

	gz := gzip.NewWriter(w)
	defer gz.Close()

	a.Debug.write(gz)
	writeBytes(gz, a.Instructions)
	return
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
