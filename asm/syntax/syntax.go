// Package syntax performs syntax verification on an AST to ensure it is in a sane state.
// Additionally performs some mutations to simplify or amend structure where needed.
package syntax

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/hexaflex/svm/arch"
	"github.com/hexaflex/svm/asm/parser"
)

// Verify performs syntax verification on the given AST to ensure it has a sane state.
// Additionally performs some mutations to simplify or amend structure where needed.
func Verify(ast *parser.AST) error {
	if err := translateNames(ast.Nodes()); err != nil {
		return err
	}

	if err := fixScopeNames(ast.Nodes()); err != nil {
		return err
	}

	if err := translateConst(ast.Nodes()); err != nil {
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

// translateNames finds all idents and replaces dots with path separators.
// Since that is the form in which symbols with scope paths are stored by the assembler.
// E.g.: `gp14.ButtonA` becomes `gp14/ButtonA`.
func translateNames(nodes *parser.List) error {
	const Sep = string(os.PathSeparator)

	for i := 0; i < nodes.Len(); i++ {
		node := nodes.At(i)

		if list, ok := node.(*parser.List); ok {
			if err := translateNames(list); err != nil {
				return err
			}
			continue
		}

		if node.Type() != parser.Ident {
			continue
		}

		ident := node.(*parser.Value)
		ident.Value = strings.ReplaceAll(ident.Value, ".", Sep)
	}
	return nil
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

// fixScopeNames finds all empty ScopeBegin nodes and generates unique names for them,
// or assigns user-provided names to them if applicable. This is the case when a ScopeBegin
// node is immediately preceeded by a label. The label becomes the scope's name.
func fixScopeNames(nodes *parser.List) error {
	for i := 0; i < nodes.Len(); i++ {
		node := nodes.At(i)

		if list, ok := node.(*parser.List); ok {
			if err := fixScopeNames(list); err != nil {
				return err
			}
			continue
		}

		if node.Type() != parser.ScopeBegin {
			continue
		}

		scope := node.(*parser.Value)

		if len(scope.Value) == 0 {
			if i > 0 && nodes.At(i-1).Type() == parser.Label {
				lbl := nodes.At(i - 1).(*parser.Value)
				scope.Value = lbl.Value
			} else {
				scope.Value = UniqueName()
			}
		}
	}
	return nil
}

// translateConst finds constant definitions and replaces them with
// simplified versions of themselves:
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

// UniqueName generates a unique name from an atomically incremented value.
var UniqueName = func() func() string {
	var value uint32
	return func() string {
		v := atomic.AddUint32(&value, 1)
		return fmt.Sprintf("$__%04x", v)
	}
}()

// createConditionalJump creates a JEZ instruction and a label to jump to.
func createConditionalJump(pos1, pos2 parser.Position) (*parser.List, *parser.Value) {
	labelName := UniqueName()

	expr := parser.NewList(pos1, parser.Expression)
	expr.Append(parser.NewValue(pos1, parser.AddressMode, "$"))
	expr.Append(parser.NewValue(pos1, parser.Ident, labelName))

	jmp := parser.NewList(pos1, parser.Instruction)
	jmp.Append(parser.NewValue(pos1, parser.Ident, "jez"), expr)

	lbl := parser.NewValue(pos2, parser.Label, labelName)
	return jmp, lbl
}

// createCompareInstr creates a compare instruction from the given condition expression.
func createCompareInstr(expr *parser.List) (*parser.List, error) {
	var name string

	index := indexOfType(expr, parser.Operator)
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

// indexOfType returns the index of the first element in the
// given expression with the specified type. Returns -1 if it
// can not be found.
func indexOfType(expr *parser.List, ntype parser.Type) int {
	for i := 0; i < expr.Len(); i++ {
		if expr.At(i).Type() == ntype {
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

		// Remove empty expression nodes. These can occur in some edge cases like having
		// a zero-operand instruction with a trailing code comment.
		//
		// Also ensure that, if there is a type descriptor in the expression, it is the
		// first element in the expression.
		for i := 1; i < instr.Len(); i++ {
			expr := instr.At(i).(*parser.List)
			if expr.Len() == 0 {
				instr.Remove(i)
				i--
			}

			if idx := indexOfType(expr, parser.TypeDescriptor); idx > 0 {
				return NewError(expr.At(idx).Position(), "a type descriptor must be the first element in an expression")
			}
		}

		// Check if the instruction is known. If not, this is not an error. We may be dealing
		// with a macro reference or an assembler directive.
		name := instr.At(0).(*parser.Value)
		if opcode, ok := arch.Opcode(name.Value); ok {
			argc := arch.Argc(opcode)
			if argc != instr.Len()-1 {
				return NewError(name.Position(), "invalid operand count for instruction %q; expected %d", name.Value, argc)
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

// isIdent returns true if n is an Ident with the given value.
func isIdent(n parser.Node, value string) bool {
	return n.Type() == parser.Ident &&
		strings.EqualFold(value, n.(*parser.Value).Value)
}
