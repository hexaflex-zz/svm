package asm

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/hexaflex/svm/arch"
	"github.com/hexaflex/svm/asm/ar"
	"github.com/hexaflex/svm/asm/eval"
	"github.com/hexaflex/svm/asm/parser"
	"github.com/hexaflex/svm/asm/syntax"
)

// assembler holds assembler context. It turns the source AST for a single module into a binary archive.
type assembler struct {
	ar      *ar.Archive             // Target archive.
	symbols map[string]int          // Table of labels or constants mapped to their respective addresses and values.
	macros  map[string]*parser.List // Macro definitions.
	address int                     // Address at which next instruction is written.
	flags   ar.DebugFlags           // Optional one-shot flags to be provided with debug symbols.
	debug   bool                    // Emit debug symbols?
}

func newAssembler(debug bool) *assembler {
	return &assembler{
		ar:      ar.New(),
		symbols: make(map[string]int),
		macros:  make(map[string]*parser.List),
		debug:   debug,
	}
}

// assemble compiles the given source AST into an archive.
// Any provided options dictate custom assembler behaviour.
func (a *assembler) assemble(ast *parser.AST, module string) (*ar.Archive, error) {
	if err := syntax.Verify(ast); err != nil {
		return nil, err
	}

	if err := a.resolveMacros(ast.Nodes(), ""); err != nil {
		return nil, err
	}

	if err := a.replaceMacroInvocations(ast.Nodes(), ""); err != nil {
		return nil, err
	}

	if err := a.resolveLabels(ast.Nodes(), ""); err != nil {
		return nil, err
	}

	if err := a.resolveEntrypoint(module); err != nil {
		return nil, err
	}

	if err := a.evaluateConstants(ast.Nodes(), ""); err != nil {
		return nil, err
	}

	if err := a.evaluateInstructions(ast.Nodes(), ""); err != nil {
		return nil, err
	}

	if err := a.compile(ast.Nodes()); err != nil {
		return nil, err
	}

	return a.ar, nil
}

// resolveEntrypoint finds the program entrypoint.
// There is expected to be one label named "main".
func (a *assembler) resolveEntrypoint(module string) error {
	name := parser.Scope(module).Join("main").String()
	if addr, ok := a.symbols[name]; ok {
		a.ar.Entrypoint = addr
		return nil
	}
	return fmt.Errorf("missing entrypoint in program; expected to find %q", name)
}

// evaluateConstants evaluates constant definitions.
func (a *assembler) evaluateConstants(nodes *parser.List, scope parser.Scope) error {
	return nodes.Each(func(_ int, n parser.Node) error {
		switch n.Type() {
		case parser.ScopeBegin:
			scope = scope.Join(n.(*parser.Value).Value)

		case parser.ScopeEnd:
			scope, _ = scope.Split()

		case parser.Constant:
			return a.evaluateConstant(n.(*parser.List), scope)
		}

		return nil
	})
}

// evaluateConstant evaluates the given constant expression.
func (a *assembler) evaluateConstant(instr *parser.List, scope parser.Scope) error {
	err := eval.Evaluate(instr, a.resolveReference, scope)
	if err != nil {
		return err
	}

	name := instr.At(0).(*parser.Value).Value
	expr := instr.At(1).(*parser.List)

	if expr.Len() != 1 || expr.At(0).Type() != parser.Number {
		return newError(expr.Position(), "invalid constant expression")
	}

	key := string(scope.Join(name))
	key = strings.ToLower(key)

	if _, ok := a.symbols[key]; ok {
		return newError(instr.Position(), "duplicate symbol %q", name)
	}

	value := expr.At(0).(*parser.Value).Value
	num, _ := parser.ParseNumber(value)

	a.symbols[key] = int(num)
	return nil
}

// evaluateInstructions evaluates all compile-time expressions in the given node list.
func (a *assembler) evaluateInstructions(nodes *parser.List, scope parser.Scope) error {
	a.address = 0
	return nodes.Each(func(_ int, n parser.Node) error {
		switch n.Type() {
		case parser.ScopeBegin:
			scope = scope.Join(n.(*parser.Value).Value)

		case parser.ScopeEnd:
			scope, _ = scope.Split()

		case parser.Instruction:
			err := eval.Evaluate(n.(*parser.List), a.resolveReference, scope)
			a.address += encodedLen(n)
			return err
		}

		return nil
	})
}

//isExternalReference returns true if name represents a reference to an external module.
func isExternalReference(name string) bool {
	alias, _ := filepath.Split(name)
	return len(alias) > 0
}

// resolveReference finds the address or value for a given external reference.
// This can be a label or constant. Optionally searches for a scope-local match first.
// Returns an error if it can't be found.
func (a *assembler) resolveReference(scope parser.Scope, name string) (int, error) {
	if name == "$$" {
		return a.address, nil
	}

	// Is this an external reference? No need to do the scope tree search.
	if isExternalReference(name) {
		key := strings.ToLower(name)
		if addr, ok := a.symbols[key]; ok {
			return addr, nil
		}
		return 0, fmt.Errorf("reference to unresolved value %s", name)
	}

	// Look for the entry in the scope tree.
	key := scope.Join(name).String()
	key = strings.ToLower(key)

	if addr, ok := a.symbols[key]; ok {
		return addr, nil
	}

	for len(scope) > 0 {
		scope, _ = scope.Split()

		key = scope.Join(name).String()
		key = strings.ToLower(key)

		if addr, ok := a.symbols[key]; ok {
			return addr, nil
		}
	}

	return 0, fmt.Errorf("reference to unresolved value %s", name)
}

// resolveMacros finds the macro defintions and adds them to the macro table in the assembler context.
// The definition is then removed from the AST.
func (a *assembler) resolveMacros(nodes *parser.List, scope parser.Scope) error {
	for i := 0; i < nodes.Len(); i++ {
		n := nodes.At(i)

		if n.Type() == parser.ScopeBegin {
			scope = scope.Join(n.(*parser.Value).Value)
			continue
		}

		if n.Type() == parser.ScopeEnd {
			scope, _ = scope.Split()
			continue
		}

		if n.Type() != parser.Macro {
			continue
		}

		macro := n.(*parser.List)
		name := macro.At(0).(*parser.Value)
		name.Value = scope.Join(name.Value).String()

		if a.hasSymbol(name.Value) {
			return newError(name.Position(), "duplicate symbol definition %q", name.Value)
		}

		a.macros[strings.ToLower(name.Value)] = macro
		nodes.Remove(i)
		i--
	}
	return nil
}

// replaceMacroInvocations finds macro invications and replaces them with the respective macro's body.
func (a *assembler) replaceMacroInvocations(nodes *parser.List, scope parser.Scope) error {
	for i := 0; i < nodes.Len(); i++ {
		n := nodes.At(i)

		if n.Type() == parser.ScopeBegin {
			scope = scope.Join(n.(*parser.Value).Value)
			continue
		}

		if n.Type() == parser.ScopeEnd {
			scope, _ = scope.Split()
			continue
		}

		if n.Type() != parser.Instruction {
			continue
		}

		instr := n.(*parser.List)
		name := instr.At(0).(*parser.Value)

		macro := a.findMacro(name.Value, scope)
		if macro == nil {
			continue
		}

		// We are about to alter the body of this macro.
		// Do this on a copy, not the original.
		macro = macro.Copy().(*parser.List)

		values := instr.Slice()[1:]
		names := macroArgs(macro, 0)
		body := macro.Slice()[len(names)+1:]
		replaceMacroContents(body, values, names)

		nodes.ReplaceAt(i, body...)
	}
	return nil
}

// findMacro returns the macro with the given (scoped) name. Returns nil if it can't be found.
func (a *assembler) findMacro(name string, scope parser.Scope) *parser.List {
	key := strings.ToLower(name)

	if m, ok := a.macros[key]; ok {
		return m
	}

	// Check the local scope.
	key = scope.Join(key).String()
	if m, ok := a.macros[key]; ok {
		return m
	}

	// Search all the way up the scope tree.
	for len(scope) > 0 {
		scope, _ = scope.Split()

		key = scope.Join(name).String()
		key = strings.ToLower(key)

		if m, ok := a.macros[key]; ok {
			return m
		}
	}

	return nil
}

// replaceMacroContents traverses instructions in body and replaces occurrences of idents from names
// with their counterparts in values.
func replaceMacroContents(body, values []parser.Node, names []*parser.Value) {
	for _, n := range body {
		if n.Type() != parser.Instruction {
			continue
		}

		instr := n.(*parser.List)
		for j := 1; j < instr.Len(); j++ {
			expr := instr.At(j).(*parser.List)
			for k := 0; k < expr.Len(); k++ {
				x := indexOfIdent(names, expr.At(k))
				if x == -1 {
					continue
				}

				if values[x].Type() != parser.Expression {
					expr.ReplaceAt(k, values[x])
				} else {
					// unpack expression
					vexpr := values[x].(*parser.List)
					expr.ReplaceAt(k, vexpr.Slice()...)
				}
			}
		}
	}
}

// indexOfIdent returns the index of ident n in set.
// Returns -1 if n is not an ident or the value is not in set.
func indexOfIdent(set []*parser.Value, n parser.Node) int {
	if n.Type() != parser.Ident {
		return -1
	}

	nvalue := n.(*parser.Value).Value

	for i, v := range set {
		if strings.EqualFold(v.Value, nvalue) {
			return i
		}
	}

	return -1
}

// resolveLabels finds all label definitionsin the given set and resolves their addresses.
// The given scope name is prefixed to the label name. Label definitions are removed.
func (a *assembler) resolveLabels(nodes *parser.List, scope parser.Scope) error {
	a.address = 0

	for i := 0; i < nodes.Len(); i++ {
		n := nodes.At(i)

		if n.Type() == parser.ScopeBegin {
			scope = scope.Join(n.(*parser.Value).Value)
			continue
		}

		if n.Type() == parser.ScopeEnd {
			scope, _ = scope.Split()
			continue
		}

		if n.Type() != parser.Label {
			a.address += encodedLen(n)
			continue
		}

		lbl := n.(*parser.Value)
		lbl.Value = scope.Join(lbl.Value).String()

		if a.hasSymbol(lbl.Value) {
			return newError(lbl.Position(), "duplicate definition name %q", lbl.Value)
		}

		a.symbols[strings.ToLower(lbl.Value)] = a.address

		nodes.Remove(i)
		i--
	}

	return nil
}

// hasSymbol returns true if the given symbol is defined as either a label, constant or macro.
func (a *assembler) hasSymbol(name string) bool {
	name = strings.ToLower(name)
	_, ok1 := a.symbols[name]
	_, ok2 := a.macros[name]
	return ok1 || ok2
}

// compile compiles all given instructions.
func (a *assembler) compile(nodes *parser.List) error {
	a.address = 0

	return nodes.Each(func(_ int, n parser.Node) error {
		switch n.Type() {
		case parser.BreakPoint:
			a.flags |= ar.Breakpoint

		case parser.Instruction:
			code, err := a.encode(n.(*parser.List))
			if err != nil {
				return err
			}
			a.emit(n.Position(), code)
		}

		return nil
	})
}

// emit emits the given instruction.
// It optionally generates debug symbols.
func (a *assembler) emit(pos parser.Position, code []byte) {
	a.ar.Instructions = append(a.ar.Instructions, code...)

	address := a.address
	a.address += len(code)

	if !a.debug {
		return
	}

	fileindex := a.addDebugFile(pos.File)
	a.ar.Debug.Symbols = append(a.ar.Debug.Symbols, ar.DebugData{
		Address: address,
		File:    fileindex,
		Line:    pos.Line,
		Col:     pos.Col,
		Offset:  pos.Offset,
		Flags:   a.flags,
	})

	// Clear out one-shot flags.
	a.flags = 0
}

// addDebugFile adds a filename to the debug symbol table, provided it does
// not already exist. Returns the index of the entry in the file list.
func (a *assembler) addDebugFile(file string) int {
	for i, v := range a.ar.Debug.Files {
		if v == file {
			return i
		}
	}
	a.ar.Debug.Files = append(a.ar.Debug.Files, file)
	return len(a.ar.Debug.Files) - 1
}

// encode encodes the given instruction into its final binary form.
func (a *assembler) encode(instr *parser.List) ([]byte, error) {
	if size, ok := isDataDirective(instr); ok {
		return encodeDataDirective(instr, size), nil
	}

	name := instr.At(0).(*parser.Value)
	opcode, ok := arch.Opcode(name.Value)
	if !ok {
		return nil, newError(name.Position(), "unknown instruction %q", name.Value)
	}

	out := make([]byte, 1, encodedLen(instr))
	out[0] = byte(opcode)

	var mode byte
	var value *parser.Value

	for i := 1; i < instr.Len(); i++ {
		expr := instr.At(i).(*parser.List)

		mode = 1

		// Check if the expression includes an explicit address mode.
		if expr.Len() == 2 {
			switch expr.At(0).(*parser.Value).Value {
			case "r":
				mode = 2
			case "$":
				mode = 0
			}

			value = expr.At(1).(*parser.Value)
		} else {
			value = expr.At(0).(*parser.Value)
		}

		num, _ := parser.ParseNumber(value.Value)

		if mode == 2 {
			out = append(out, (mode<<6)|byte(num&0x3f))
		} else {
			out = append(out, mode<<6, byte(num>>8), byte(num))
		}
	}

	return out, nil
}

// encodeMacroInvocation replaces the given invocation with the specified macro contents and
// returns their encoded versions.
func encodeMacroInvocation(macro, invocation *parser.List) ([]byte, error) {
	argv := invocation.Slice()[1:]
	args := macroArgs(macro, len(argv))

	if len(argv) != len(args) {
		return nil, newError(invocation.Position(), "invalid number of arguments in macro invocation; expected %d, have %d", len(args), len(argv))
	}

	return nil, nil
}

// macroArgs returns the given macro's operands as a list of idents.
func macroArgs(macro *parser.List, size int) []*parser.Value {
	out := make([]*parser.Value, 0, size)

	for i := 1; i < macro.Len() && macro.At(i).Type() == parser.Expression; i++ {
		expr := macro.At(i).(*parser.List)
		out = append(out, expr.At(0).(*parser.Value))
	}

	return out
}

// encodeDataDirective encodes the operands for the given data directive.
func encodeDataDirective(instr *parser.List, size int) []byte {
	out := make([]byte, 0, (instr.Len()-1)*size)

	for i := 1; i < instr.Len(); i++ {
		expr := instr.At(i).(*parser.List)
		value := expr.At(0).(*parser.Value)

		if value.Type() == parser.String {
			for _, r := range value.Value {
				out = writeData(out, int64(r), size)
			}
		} else {
			num, _ := parser.ParseNumber(value.Value)
			out = writeData(out, num, size)
		}
	}

	return out
}

// writeData writes the given value to out as a sequence of bytes and returns the resulting byte slice.
func writeData(out []byte, v int64, size int) []byte {
	switch size {
	case 1:
		out = append(out, byte(v))
	case 2:
		out = append(out, byte(v>>8), byte(v))
	case 4:
		out = append(out, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	case 8:
		out = append(out, byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}
	return out
}

// encodedLen returns the byte size occupied by the given node's compiled version.
func encodedLen(n parser.Node) int {
	if size, ok := isDataDirective(n); ok {
		return encodedDataDirectiveLen(n.(*parser.List), size)
	}

	if n.Type() != parser.Instruction {
		return 0
	}

	instr := n.(*parser.List)
	name := instr.At(0).(*parser.Value)
	opcode, _ := arch.Opcode(name.Value)
	size := 1

	switch arch.Argc(opcode) {
	case 3:
		size += encodedExprLen(instr.At(3).(*parser.List))
		fallthrough
	case 2:
		size += encodedExprLen(instr.At(2).(*parser.List))
		fallthrough
	case 1:
		size += encodedExprLen(instr.At(1).(*parser.List))
	}

	return size
}

// encodedExprLen returns the byte size for the encoded version of the given expression.
func encodedExprLen(expr *parser.List) int {
	for i := 0; i < expr.Len(); i++ {
		n := expr.At(i)
		if n.Type() == parser.AddressMode && n.(*parser.Value).Value == "r" {
			return 1
		}
	}
	return 3
}

// encodedDataDirectiveLen computes the encoded length for the given data directive.
func encodedDataDirectiveLen(instr *parser.List, bytesize int) int {
	var size int

	for i := 1; i < instr.Len(); i++ {
		expr := instr.At(i).(*parser.List)

		expr.Each(func(_ int, n parser.Node) error {
			if n.Type() == parser.String {
				size += utf8.RuneCountInString(n.(*parser.Value).Value) * bytesize
			} else {
				size += bytesize
			}
			return nil
		})
	}

	return size
}

// isDataDirective returns true if n represents a data directive.
// If true, returns the byte size it represents.
func isDataDirective(n parser.Node) (int, bool) {
	if n.Type() != parser.Instruction {
		return 0, false
	}

	instr := n.(*parser.List)
	name := instr.At(0).(*parser.Value)

	switch strings.ToLower(name.Value) {
	case "d8":
		return 1, true
	case "d16":
		return 2, true
	case "d32":
		return 4, true
	case "d64":
		return 8, true
	}

	return 0, false
}
