package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexaflex/svm/asm"
	"github.com/hexaflex/svm/asm/ar"
)

func main() {
	config := parseArgs()

	switch {
	case config.DumpAST:
		dumpAST(config)
	case config.DumpArchive:
		dumpArchive(config)
	default:
		ar := buildBinary(config)
		buildDebug(config, ar)
	}
}

// dumpArchive builds the final bnary archive and prints a human readable version of it
// to the requested output.
func dumpArchive(c *Config) {
	ar, err := asm.Build(c.Input, c.Includes, c.DebugBuild)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, ar.String())
}

// dumpAST loads the source AST and writes a human readable version of it to the
// requested output.
func dumpAST(c *Config) {
	ast, err := asm.BuildAST(c.Input, c.Includes)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, ast)
}

// buildBinary builds a binary program and writes it to the requested output location.
func buildBinary(c *Config) *ar.Archive {
	ar, err := asm.Build(c.Input, c.Includes, c.DebugBuild)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	w, close := makeWriter(c.Output)
	defer close()

	if _, err = w.Write(ar.Instructions); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return ar
}

// buildDebug builds a file with debug symbols.
func buildDebug(c *Config, ar *ar.Archive) {
	if !c.DebugBuild {
		return
	}

	file := c.Output
	if index := strings.LastIndex(file, "."); index > -1 {
		file = file[:index]
	}
	file += ".dbg"

	w, close := makeWriter(file)
	defer close()

	if err := ar.Debug.Save(w); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// makeWriter creates an output writer and a cleanup function for it.
func makeWriter(file string) (io.Writer, func()) {
	dir, _ := filepath.Split(file)
	err := os.MkdirAll(dir, 0744)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fd, err := os.Create(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return fd, func() { fd.Close() }
}
