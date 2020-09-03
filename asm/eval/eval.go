// Package eval facilitates compile-time evaluation of expressions.
// This covers arithmetic and comparison operations.
package eval

import (
	"strconv"
	"strings"

	"github.com/hexaflex/svm/asm/parser"
)

// referenceFunc finds the address or value for a given external reference.
// This can be a label or constant. Returns false if it can't be found.
type referenceFunc func(parser.Scope, string) (int, error)

// Evaluate evaluates the expressions in the given instruction.
func Evaluate(instr *parser.List, resolve referenceFunc, scope parser.Scope) error {
	return instr.Each(func(i int, n parser.Node) error {
		if i > 0 {
			err := evalExpression(n.(*parser.List), resolve, scope)
			if err != nil && strings.Index(err.Error(), "reference to undefined value") == -1 {
				return err
			}
		}
		return nil
	})
}

// evalExpression evaluates the given expression.
//
// The goal being to reduce it to its minimal representation.
//
// If successful, the expression will contain either one or two elements.
// In the case of two, the first is always an address mode marker.
// The value is either a number or an ident.
func evalExpression(n *parser.List, resolve referenceFunc, scope parser.Scope) error {
	postfix, err := toPostfix(n)
	if err != nil {
		return err
	}

	value, err := evalPostfix(postfix, resolve, scope)
	if err != nil {
		return err
	}

	// Replace the expression with the resulting value.
	// Make sure we preserve the address mode markers and type descriptors.

	n.Clear()

	for _, v := range postfix {
		if v.Type() != parser.TypeDescriptor && v.Type() != parser.AddressMode {
			n.Append(value)
			break
		}
		n.Append(v)
	}

	return nil
}

// evalPostfix evaluates the given postfix expression and returns its value if possible.
func evalPostfix(expr []parser.Node, resolve referenceFunc, scope parser.Scope) (parser.Node, error) {
	stack := make([]interface{}, 0, len(expr))

	var va, vb interface{}
	for _, n := range expr {
		switch n.Type() {
		case parser.AddressMode, parser.TypeDescriptor:
			/* nop */

		case parser.Ident:
			name := n.(*parser.Value)
			value, err := resolve(scope, strings.ToLower(name.Value))

			if err != nil {
				return nil, NewError(n.Position(), err.Error())
			}

			stack = append(stack, int64(value))

		case parser.Number, parser.String:
			ev, err := parseValue(n)
			if err != nil {
				return nil, err
			}
			stack = append(stack, ev)

		default:
			str := n.(*parser.Value).Value

			if len(stack) < 2 && n.Type() == parser.Operator {
				// We have a unary operation.

				if len(stack) < 1 {
					return nil, NewError(n.Position(), "missing operand for operation %q", str)
				}

				va = int64(0)
				vb = stack[len(stack)-1]
				stack = stack[:len(stack)-1]

			} else if len(stack) < 2 {
				return nil, NewError(n.Position(), "missing operands for operation %q", str)

			} else {
				va = stack[len(stack)-2]
				vb = stack[len(stack)-1]
				stack = stack[:len(stack)-2]
			}

			vc, err := apply(str, va, vb)
			if err != nil {
				return nil, err
			}

			stack = append(stack, vc)
		}
	}

	if len(stack) == 0 {
		return nil, NewError(expr[0].Position(), "invalid expression; no result")
	}

	if len(stack) > 1 {
		return nil, NewError(expr[0].Position(), "invalid expression; too many results")
	}

	var val *parser.Value
	pos := expr[0].Position()

	switch tv := stack[0].(type) {
	case int64:
		val = parser.NewValue(pos, parser.Number, strconv.FormatInt(tv, 10))
	case string:
		val = parser.NewValue(pos, parser.String, tv)
	default:
		return nil, NewError(expr[0].Position(), "expression evaluates to invalid type %T", tv)
	}

	return val, nil
}

// parseValue parses the given value into its real representation.
// E.g.: a numeric literal becomes the actual int64 or float64.
func parseValue(n parser.Node) (interface{}, error) {
	str := n.(*parser.Value).Value

	switch n.Type() {
	case parser.Number:
		nv, _ := parser.ParseNumber(str)
		return nv, nil

	case parser.String:
		return str, nil

	default:
		return nil, NewError(n.Position(), "invalid node type %s; expected number or string", n.Type().String())
	}
}

// hasValue returns true if the given node represents the given value
func hasValue(n parser.Node, v string) bool {
	tn, ok := n.(*parser.Value)
	return ok && strings.EqualFold(tn.Value, v)
}
