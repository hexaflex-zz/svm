// Package asm implements an assembler which turns a module and its dependencies into
// a binary program, ready for use on a VM.
package asm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexaflex/svm/asm/ar"
	"github.com/hexaflex/svm/asm/parser"

	"github.com/pkg/errors"
)

// Build builds a binary program from the given module and its dependencies.
// It optionally emits debug symbols. The module and its dependencies are expected
// to have their sources located in the given import root directory.
func Build(importpath, module string, debug bool) (*ar.Archive, error) {
	// Construct the AST from all sources.
	ast := parser.NewAST()

	err := buildAST(ast, importpath, module, nil)
	if err != nil {
		return nil, err
	}

	// Construct the binary archive.
	asm := newAssembler(debug)
	return asm.assemble(ast, module)
}

// BuildAST builds only the AST and exits. This is mostly useful for debugging.
// Build() is probably the function you want.
func BuildAST(importpath, module string) (*parser.AST, error) {
	ast := parser.NewAST()
	return ast, buildAST(ast, importpath, module, nil)
}

// containsString returns true if set contains v.
func containsString(set []string, v string) bool {
	for _, sv := range set {
		if sv == v {
			return true
		}
	}
	return false
}

// buildAST constructs an AST from all the module's sources and its dependencies.
// It ensures the module and its dependencies do not contain any circular references.
func buildAST(ast *parser.AST, importpath, module string, queue []string) error {
	module = strings.ToLower(module)

	if containsString(queue, module) {
		return fmt.Errorf("circular reference to module %q detected", module)
	}

	queue = append(queue, module)

	// Find all the source files for the given module.
	sources, err := collateSources(importpath, module)
	if err != nil {
		return err
	}

	// Load them all into an AST.
	newAst := parser.NewAST()
	newAst.Nodes().Append(parser.NewValue(parser.Position{}, parser.ScopeBegin, module))

	for _, file := range sources {
		if err := newAst.ParseFile(file); err != nil {
			return err
		}
	}

	newAst.Nodes().Append(parser.NewValue(parser.Position{}, parser.ScopeEnd, ""))

	err = testCircularImportsAST(newAst, importpath, queue)
	ast.Merge(newAst)
	return err
}

// testCircularImportsAST finds all import statenents in the given AST and checks them recursively.
func testCircularImportsAST(ast *parser.AST, importpath string, queue []string) error {
	return ast.Nodes().Each(func(_ int, n parser.Node) error {
		if n.Type() != parser.Instruction {
			return nil
		}

		instr := n.(*parser.List)
		name := instr.At(0).(*parser.Value).Value
		if !strings.EqualFold(name, "import") {
			return nil
		}

		var path string
		switch instr.Len() {
		case 2:
			expr := instr.At(1).(*parser.List)
			path = expr.At(0).(*parser.Value).Value
		case 3:
			expr := instr.At(2).(*parser.List)
			path = expr.At(0).(*parser.Value).Value
		default:
			return parser.NewError(instr.Position(), "invalid import path")
		}

		if err := buildAST(ast, importpath, path, queue); err != nil {
			if _, ok := err.(*parser.Error); ok {
				return err
			}
			return parser.NewError(instr.Position(), err.Error())
		}

		return nil
	})
}

// collateSources returns a list of all the source files associated with
// the given module. These will be absolute paths and are expected to be
// located in the import root directory.
func collateSources(importpath, module string) ([]string, error) {
	path := filepath.Join(importpath, module)

	fd, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to locate source directory for module %q", module)
	}

	files, err := fd.Readdirnames(-1)
	fd.Close()

	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file names for moduke %q", module)
	}

	// remove entries with invalid file extensions.
	// Ensure the rest are absolute paths.
	for i := 0; i < len(files); i++ {
		if isSourceFile(files[i]) {
			files[i] = filepath.Join(path, files[i])
			files[i], _ = filepath.Abs(files[i])
			continue
		}

		copy(files[i:], files[i+1:])
		files = files[:len(files)-1]
		i--
	}

	return files, nil
}

// isSourceFile returns true if file has an expected file extension.
func isSourceFile(file string) bool {
	ext := filepath.Ext(file)
	switch strings.ToLower(ext) {
	case ".svm", ".asm":
		return true
	}
	return false
}
