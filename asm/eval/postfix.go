package eval

import "github.com/hexaflex/svm/asm/parser"

// ref: https://en.wikipedia.org/wiki/Shunting-yard_algorithm

// toPostfix uses Dijkstra's Shunting Yard algorithm to convert the given
// infix expression into postfix notation. This makes it a lot easier to
// evaluate later on.
//
// There should be no more parentheses once this call is finished.
func toPostfix(n *parser.List) ([]parser.Node, error) {
	out := make([]parser.Node, 0, n.Len())
	ops := make([]parser.Node, 0, n.Len()/2)

	if err := n.Each(func(i int, n parser.Node) error {
		var err error

		switch n.Type() {
		case parser.Operator:
			out, ops, err = postfixHandleOp(out, ops, n)
		default:
			out = append(out, n)
		}

		return err
	}); err != nil {
		return nil, err
	}

	for i := len(ops) - 1; i >= 0; i-- {
		if hasValue(ops[i], "(") {
			return nil, NewError(ops[i].Position(), "mismatched opening parenthesis")
		}
		out = append(out, ops[i])
	}

	return out, nil
}

// postfixHandleOp handles the given operator according to the Shunting yard algorithm rules.
func postfixHandleOp(out, ops []parser.Node, n parser.Node) ([]parser.Node, []parser.Node, error) {
	if hasValue(n, "(") {
		ops = append(ops, n)
		return out, ops, nil
	}

	if hasValue(n, ")") {
		var haveParen bool

		for len(ops) > 0 {
			top := ops[len(ops)-1]
			ops = ops[:len(ops)-1]
			if hasValue(top, "(") {
				haveParen = true
				break
			}
			out = append(out, top)
		}

		if !haveParen {
			return nil, nil, NewError(n.Position(), "mismatched closing parenthesis")
		}

		return out, ops, nil
	}

	if len(ops) == 0 || hasValue(ops[len(ops)-1], "(") {
		ops = append(ops, n)
		return out, ops, nil
	}

	top := ops[len(ops)-1]
	np, nleft := opProperties(n)
	tp, _ := opProperties(top)

	if (np > tp) || (np == tp && !nleft) {
		ops = append(ops, n)
		return out, ops, nil
	}

	for len(ops) > 0 {
		top := ops[len(ops)-1]
		np, nleft := opProperties(n)
		tp, _ := opProperties(top)

		if (np < tp) || (np == tp && nleft) {
			out = append(out, top)
			ops = ops[:len(ops)-1]
		} else {
			break
		}
	}

	ops = append(ops, n)
	return out, ops, nil
}

// opProperties treats n as an operator and returns its precedence,
// as well as true if it is left-associative.
func opProperties(n parser.Node) (int, bool) {
	switch n.(*parser.Value).Value {
	case "(", ")":
		return 1, true
	case "u+", "u-", "u^":
		return 3, false
	case "+", "-":
		return 3, true
	case "*", "/", "%":
		return 4, true
	case ">>", "<<":
		return 5, true
	case "<=", ">=", "<", ">":
		return 6, true
	case "!=", "==":
		return 7, true
	case "&":
		return 8, true
	case "^":
		return 9, false
	case "|":
		return 10, true
	}

	panic("eval: unknown operator " + n.(*parser.Value).Value)
}
