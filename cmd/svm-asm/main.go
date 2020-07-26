package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hexaflex/svm/asm"
)

func main() {
	config := parseArgs()

	switch {
	case config.DumpAST:
		dumpAST(config)
	case config.DumpArchive:
		dumpArchive(config)
	default:
		buildBinary(config)
	}
}

// dumpArchive builds the final bnary archive and prints a human readable version of it
// to the requested output.
func dumpArchive(c *Config) {
	ar, err := asm.Build(c.ImportRoot, c.Program, c.DebugBuild)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	w, close := makeWriter(c)
	defer close()

	fmt.Fprintln(w, ar.String())
}

// dumpAST loads the source AST and writes a human readable version of it to the
// requested output.
func dumpAST(c *Config) {
	ast, err := asm.BuildAST(c.ImportRoot, c.Program)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	w, close := makeWriter(c)
	defer close()

	fmt.Fprintln(w, ast)
}

// buildBinary builds a binary program and writes it to the requested output location.
func buildBinary(c *Config) {
	ar, err := asm.Build(c.ImportRoot, c.Program, c.DebugBuild)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	w, close := makeWriter(c)
	defer close()

	err = ar.Save(w)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// makeWriter creates an output writer and a cleanup function for it.
func makeWriter(c *Config) (io.Writer, func()) {
	if c.Output == "" {
		return os.Stdout, func() {}
	}

	dir, _ := filepath.Split(c.Output)
	err := os.MkdirAll(dir, 0744)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fd, err := os.Create(c.Output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return fd, func() { fd.Close() }
}
