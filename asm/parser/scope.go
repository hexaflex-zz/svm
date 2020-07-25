package parser

import (
	"os"
	"path/filepath"
	"strings"
)

// Scope defines a scope path.
type Scope string

// Join returns a copy of the current scope with the given value append to it.
func (s Scope) Join(value string) Scope {
	return Scope(filepath.Join(string(s), value))
}

// Split splits the scope immediately following the final separator.
// If there is no separator, the first returned value is empty and
// the second is the scope as-is.
func (s Scope) Split() (Scope, Scope) {
	a, b := filepath.Split(string(s))
	sep := string(os.PathSeparator)
	if strings.HasSuffix(a, sep) {
		a = a[:len(a)-len(sep)]
	}
	return Scope(a), Scope(b)
}

func (s Scope) String() string {
	return string(s)
}

func (s Scope) Len() int {
	return len(s)
}
