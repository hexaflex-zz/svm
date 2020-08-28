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
)

type astCache struct {
	file string
	ast  *parser.AST
}

// Build builds a binary program from the given module and its dependencies.
// It optionally emits debug symbols. The module and its dependencies are expected
// to have their sources located in the given import root directory.
func Build(file string, includeSearchPaths []string, debug bool) (*ar.Archive, error) {
	ast, err := BuildAST(file, includeSearchPaths)
	if err != nil {
		return nil, err
	}

	asm := newAssembler(debug)
	return asm.assemble(ast)
}

// BuildAST builds the full AST for the given file and its dependencies.
func BuildAST(file string, includeSearchPaths []string) (*parser.AST, error) {
	ast := parser.NewAST()
	return ast, buildAST(ast, file, includeSearchPaths, nil)
}

// buildASTreads the given source file and its dependencies into the specified AST.
// It ensures the file and its dependencies do not contain any circular include references.
func buildAST(ast *parser.AST, file string, includeSearchPaths, dependencyChain []string) error {
	file = findSourceFile(file, includeSearchPaths)

	if containsString(dependencyChain, file) {
		return fmt.Errorf("circular reference to file %q detected", file)
	}

	dependencyChain = append(dependencyChain, file)

	if err := ast.ParseFile(file); err != nil {
		return err
	}

	dir, _ := filepath.Split(file)
	return testAndBuildIncludes(ast.Nodes(), append(includeSearchPaths, dir), dependencyChain)
}

// testAndBuildIncludes finds all include statements in the given AST and checks them recursively.
// If valid, parses them into the AST.
func testAndBuildIncludes(nodes *parser.List, includeSearchPaths, dependencyChain []string) error {
	for i := 0; i < nodes.Len(); i++ {
		node := nodes.At(i)
		if node.Type() != parser.Instruction {
			continue
		}

		instr := node.(*parser.List)
		name := instr.At(0).(*parser.Value).Value
		if !strings.EqualFold(name, "include") {
			continue
		}

		if instr.Len() != 2 {
			return parser.NewError(instr.Position(), "invalid include statement; expected `include <path>`")
		}

		expr := instr.At(1).(*parser.List)
		if expr.Len() == 0 {
			return parser.NewError(expr.Position(), "invalid include statement; expected `include <path>`")
		}

		arg := expr.At(0)
		if arg.Type() != parser.String {
			return parser.NewError(arg.Position(), "invalid include path; expected string")
		}

		// Parse source file into its own AST.
		path := arg.(*parser.Value).Value
		ast := parser.NewAST()

		if err := buildAST(ast, path, includeSearchPaths, dependencyChain); err != nil {
			if _, ok := err.(*parser.Error); ok {
				return err
			}
			return parser.NewError(instr.Position(), err.Error())
		}

		// Replace include node with contents if new AST.
		set := ast.Nodes().Slice()
		nodes.ReplaceAt(i, set...)
	}
	return nil
}

// findSourceFile returns the fully qualified version of file.
// Returns file as-is if it exists on disk. If not, looks in directories
// specified by the given include search paths.
func findSourceFile(file string, includeSearchPaths []string) string {
	if stat, err := os.Stat(file); err == nil && !stat.IsDir() {
		return file
	}

	for _, inc := range includeSearchPaths {
		path := filepath.Join(inc, file)
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			return path
		}
	}

	return file
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
