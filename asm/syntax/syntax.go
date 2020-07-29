// Package syntax performs syntax verification on an AST to ensure it is in a sane state.
// Additionally performs some mutations to simplify or amend structure where needed.
package syntax

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/hexaflex/svm/arch"
	"github.com/hexaflex/svm/asm/parser"
)

// Verify performs syntax verification on the given AST to ensure it has a sane state.
// Additionally performs some mutations to simplify or amend structure where needed.
func Verify(ast *parser.AST) error {
	if err := translateConst(ast.Nodes()); err != nil {
		return err
	}

	if err := fixScopeNames(ast.Nodes()); err != nil {
		return err
	}

	aliases, err := parseImports(ast.Nodes())
	if err != nil {
		return err
	}

	if err := fixAliases(ast.Nodes(), aliases); err != nil {
		return err
	}

	if err := translateIf(ast.Nodes()); err != nil {
		return err
	}

	if err := testMacros(ast.Nodes()); err != nil {
		return err
	}

	if err := testInstructions(ast.Nodes()); err != nil {
		return err
	}

	return testNumbers(ast.Nodes())
}

// testMacros finds macro definitions and ensures they have a sane layout.
func testMacros(nodes *parser.List) error {
	return nodes.Each(func(_ int, n parser.Node) error {
		if n.Type() != parser.Macro {
			return nil
		}

		macro := n.(*parser.List)

		if macro.Len() < 1 {
			return NewError(macro.Position(), "invalid macro definition; missing name")
		}

		if macro.At(0).Type() != parser.Ident {
			return NewError(macro.At(0).Position(), "invalid macro name; expected ident")
		}

		for i := 1; i < macro.Len() && macro.At(i).Type() == parser.Expression; i++ {
			expr := macro.At(i).(*parser.List)
			if expr.Len() != 1 || expr.At(0).Type() != parser.Ident {
				return NewError(expr.At(0).Position(), "invalid macro operand; expected ident")
			}
		}

		return nil
	})
}

// fixScopeNames finds all empty ScopeBegin nodes and generates unique names for them.
func fixScopeNames(nodes *parser.List) error {
	return nodes.Each(func(_ int, n parser.Node) error {
		if list, ok := n.(*parser.List); ok {
			return fixScopeNames(list)
		}

		if n.Type() != parser.ScopeBegin {
			return nil
		}

		scope := n.(*parser.Value)

		if len(scope.Value) == 0 {
			scope.Value = uniqueName()
		}

		return nil
	})
}

// translateConst finds constant definitions, It looks for uses of these in the rest
// of the program and replaces those uses with the expression represented by the constant.
// The const nodes are replaced with simplified versions of themselves:
//
//    List{"const", Expr1, Expr2}
//
// where Expr1 contains the constant name and Expr2 the value, becomes:
//
//    List{Name, Expr2}
//
func translateConst(nodes *parser.List) error {
	for i := 0; i < nodes.Len(); i++ {
		n := nodes.At(i)

		if n.Type() == parser.Macro {
			if err := translateConst(n.(*parser.List)); err != nil {
				return err
			}
			continue
		}

		if n.Type() != parser.Constant {
			continue
		}

		constant := n.(*parser.List)

		if constant.Len() < 2 {
			return NewError(constant.Position(), "missing operands in const definition")
		}

		expr1 := constant.At(1).(*parser.List)
		if expr1.Len() < 1 {
			return NewError(expr1.Position(), "missing expression in const definition")
		}

		if expr1.At(0).Type() != parser.Ident {
			return NewError(expr1.Position(), "invalid expression name; expected ident")
		}

		name := expr1.At(0).(*parser.Value)
		expr2 := constant.At(2).(*parser.List)

		newConst := parser.NewList(n.Position(), parser.Constant)
		newConst.Append(name, expr2)
		nodes.ReplaceAt(i, newConst)

		if err := replaceConst(nodes, name.Value, expr2.Slice()); err != nil {
			return err
		}
	}
	return nil
}

// replaceConst finds references to the given constant and replaces the referenec
// with the specified expression.
func replaceConst(nodes *parser.List, name string, expr []parser.Node) error {
	for i := 0; i < nodes.Len(); i++ {
		n := nodes.At(i)

		if n.Type() == parser.Macro {
			if err := replaceConst(n.(*parser.List), name, expr); err != nil {
				return err
			}
			continue
		}

		if (nodes.Type() == parser.Instruction && i < 1) ||
			(nodes.Type() == parser.Constant && i < 2) {
			continue
		}

		if instr, ok := n.(*parser.List); ok {
			if err := replaceConst(instr, name, expr); err != nil {
				return err
			}
		}

		if isIdent(n, name) {
			nodes.ReplaceAt(i, expr...)
			i--
		}
	}
	return nil
}

// translateIf finds If statements and translates them to correct assembly code.
//
//   if r0 < r1                   ; test if r0 is less than r1.
//      mul r0, r0, -1            ; Multiply r0 by -1 iff the condition is true.
//
// The above is translated into:
//
//   clt r0, r1                   ; RST/compare = 1 iff r0 < r1
//   jez _4239874274              ; Jump past the multiply iff RST/compare = 0
//   mul r0, r0, -1               ; Multiply r0 by -1.
//   :_4239874274
//
// The label is automatically generated to have a unique, unused name.
func translateIf(nodes *parser.List) error {
	for i := 0; i < nodes.Len(); i++ {
		n := nodes.At(i)

		if n.Type() == parser.Macro {
			if err := translateIf(n.(*parser.List)); err != nil {
				return err
			}
			continue
		}

		if n.Type() != parser.Conditional {
			continue
		}

		cond := n.(*parser.List)
		if cond.Len() != 2 {
			return NewError(cond.Position(), "if statement must have a condition expression and preceed an instruction")
		}

		cmp, err := createCompareInstr(cond.At(0).(*parser.List))
		if err != nil {
			return err
		}

		jmp, lbl := createConditionalJump(cond.At(0).Position(), cond.At(1).Position())
		nodes.ReplaceAt(i, cmp, jmp, cond.At(1), lbl)
	}
	return nil
}

// uniqueName generates a unique name from an atomically incremented value.
var uniqueName = func() func() string {
	var value uint32
	return func() string {
		v := atomic.AddUint32(&value, 1)
		return fmt.Sprintf("$__%04x", v)
	}
}()

// createConditionalJump creates a JEZ instruction and a label to jump to.
func createConditionalJump(pos1, pos2 parser.Position) (*parser.List, *parser.Value) {
	labelName := uniqueName()

	expr := parser.NewList(pos1, parser.Expression)
	expr.Append(parser.NewValue(pos1, parser.AddressMode, "$"))
	expr.Append(parser.NewValue(pos1, parser.Ident, labelName))

	jmp := parser.NewList(pos1, parser.Instruction)
	jmp.Append(parser.NewValue(pos1, parser.Ident, "jez"), expr)

	lbl := parser.NewValue(pos2, parser.Label, labelName)
	return jmp, lbl
}

// createCompareInstr creates a compare unstruction from the given condition expression.
func createCompareInstr(expr *parser.List) (*parser.List, error) {
	var name string

	index := indexOfOperator(expr)
	if index == -1 {
		return nil, NewError(expr.Position(), "invalid conditional expression; expected <value> <operator> <value>")
	}

	cmp := parser.NewList(expr.Position(), parser.Instruction)
	op := expr.At(index).(*parser.Value)

	switch op.Value {
	case "==":
		name = "ceq"
	case "!=":
		name = "cne"
	case "<":
		name = "clt"
	case "<=":
		name = "cle"
	case ">":
		name = "cgt"
	case ">=":
		name = "cge"
	default:
		return nil, NewError(op.Position(), "unexpected token %q; expected a compare operator", op.Value)
	}

	arg1 := expr.Copy().(*parser.List)
	arg1.RemoveRange(index, arg1.Len()-1)

	arg2 := expr.Copy().(*parser.List)
	arg2.RemoveRange(0, index)

	cmp.Append(parser.NewValue(expr.Position(), parser.Ident, name), arg1, arg2)
	return cmp, nil
}

func indexOfOperator(expr *parser.List) int {
	for i := 0; i < expr.Len(); i++ {
		if expr.At(i).Type() == parser.Operator {
			return i
		}
	}
	return -1
}

// testInstructions ensures instructions are properly formatted and refer to valid opcodes.
func testInstructions(nodes *parser.List) error {
	return nodes.Each(func(_ int, n parser.Node) error {
		if n.Type() == parser.Macro {
			return testInstructions(n.(*parser.List))
		}

		if n.Type() != parser.Instruction {
			return nil
		}

		instr := n.(*parser.List)

		if instr.At(0).Type() != parser.Ident {
			return NewError(instr.At(0).Position(), "invalid instruction name; expected ident")
		}

		// Check f the instruction is known. If not, this is not an error. We mau be dealing
		// with a macro reference of an assembler directive.
		name := instr.At(0).(*parser.Value)
		if opcode, ok := arch.Opcode(name.Value); ok {
			argc := arch.Argc(opcode)
			if argc != instr.Len()-1 {
				return NewError(name.Position(), "invalid operand count for instruction %q; expected %d", name.Value, argc)
			}
		}

		for i := 1; i < instr.Len(); i++ {
			expr := instr.At(i).(*parser.List)
			if expr.Len() == 0 {
				return NewError(expr.Position(), "unexpected empty expression in instruction operand")
			}
		}

		return nil
	})
}

// testNumbers finds numeric literals and ensures they can be parsed into
// actual integers without issue.
func testNumbers(nodes *parser.List) error {
	return nodes.Each(func(_ int, n parser.Node) error {
		if list, ok := n.(*parser.List); ok {
			return testNumbers(list)
		}

		if n.Type() != parser.Number {
			return nil
		}

		node := n.(*parser.Value)
		_, err := parser.ParseNumber(node.Value)
		if err != nil {
			return NewError(node.Position(), "invalid number: %v", err)
		}

		return nil
	})
}

// fixAliases finds uses of import aliases and replaces the alias part with full import paths.
func fixAliases(nodes *parser.List, aliases map[string]string) error {
	return nodes.Each(func(_ int, n parser.Node) error {
		if n.Type() == parser.Macro {
			return fixAliases(n.(*parser.List), aliases)
		}

		if n.Type() != parser.Instruction && n.Type() != parser.Constant {
			return nil
		}

		instr := n.(*parser.List)

		name := instr.At(0).(*parser.Value)
		fixAliasIdent(name, aliases)

		for i := 1; i < instr.Len(); i++ {
			expr := instr.At(i).(*parser.List)
			for j := 0; j < expr.Len(); j++ {
				n := expr.At(j)
				if n.Type() == parser.Ident {
					fixAliasIdent(n.(*parser.Value), aliases)
				}
			}
		}

		return nil
	})
}

// fixAliasIdent checks if the given ident defines a reference to an external symbol.
// The alias component is replaced with the import path if possible.
func fixAliasIdent(ident *parser.Value, aliases map[string]string) {
	if index := strings.Index(ident.Value, "."); index > -1 {
		alias := strings.ToLower(ident.Value[:index])
		if path, ok := aliases[alias]; ok {
			ident.Value = filepath.Join(path, ident.Value[index+1:])
		}
	}
}

// parseImports ensures import statements have the correct layout.
// Returns a mapping of aliases to path names. Import nodes are removed.
func parseImports(nodes *parser.List) (map[string]string, error) {
	aliases := make(map[string]string)
	return aliases, parseImportsRec(nodes, aliases)
}

func parseImportsRec(nodes *parser.List, aliases map[string]string) error {
	for i := 0; i < nodes.Len(); i++ {
		n := nodes.At(i)

		if n.Type() == parser.Macro {
			if err := parseImportsRec(n.(*parser.List), aliases); err != nil {
				return err
			}
		}

		instr, ok := isInstruction(n, "import")
		if !ok {
			continue
		}

		switch instr.Len() {
		case 2:
			expr := instr.At(1).(*parser.List)
			switch expr.Len() {
			case 1:
				if expr.At(0).Type() != parser.String {
					return NewError(expr.At(0).Position(), "invalid import path; expected string")
				}

				// create an alias from the last portion of the path.
				// e.g.: "path/to/thing" -> "thing"
				path := expr.At(0).(*parser.Value).Value
				_, alias := filepath.Split(path)
				alias = strings.ToLower(alias)
				aliases[alias] = path

			default:
				return NewError(expr.Position(), "import statement must have either one or two operands; did you forget a comma?")
			}

		case 3:
			expr := instr.At(1).(*parser.List)
			if expr.At(0).Type() != parser.Ident {
				return NewError(expr.At(0).Position(), "invalid import alias; expected ident")
			}

			alias := expr.At(0).(*parser.Value).Value
			alias = strings.ToLower(alias)

			expr = instr.At(2).(*parser.List)
			if expr.At(0).Type() != parser.String {
				return NewError(expr.At(0).Position(), "invalid import path; expected string")
			}

			path := expr.At(0).(*parser.Value).Value
			aliases[alias] = path

		default:
			return NewError(instr.Position(), "import statement must have either one or two operands")
		}

		nodes.Remove(i)
		i--
	}
	return nil
}

// isInstruction returns true if n is an instruction with the given name.
// If true, returns n as a List.
func isInstruction(n parser.Node, name string) (*parser.List, bool) {
	if n.Type() != parser.Instruction {
		return nil, false
	}
	instr := n.(*parser.List)
	return instr, isIdent(instr.At(0), name)
}

// isIdent returns true if n is an Ident with the given value.
func isIdent(n parser.Node, value string) bool {
	return n.Type() == parser.Ident &&
		strings.EqualFold(value, n.(*parser.Value).Value)
}
