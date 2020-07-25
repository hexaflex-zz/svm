package eval

import (
	"fmt"
)

// apply performs the given arithmetic or compare operation on operands a and b
// and returns the result. Returns an error if something went wrong.
//
// Operands are expected to be of the types int64, bool or string.
// This function performs implicit type conversions where applicable.
//
// Supported operations are: + - * / % << >> & | ^ == < <= > >=
// Not all data types are supported by all operations.
func apply(op string, a, b interface{}) (interface{}, error) {
	switch op {
	case "+":
		return add(a, b)
	case "-":
		return sub(a, b)
	case "*":
		return mul(a, b)
	case "/":
		return div(a, b)
	case "%":
		return mod(a, b)
	case "<<":
		return shl(a, b)
	case ">>":
		return shr(a, b)
	case "&":
		return and(a, b)
	case "|":
		return or(a, b)
	case "^":
		return xor(a, b)
	case "==":
		return eq(a, b)
	case "<":
		return lt(a, b)
	case "<=":
		return le(a, b)
	case ">":
		return gt(a, b)
	case ">=":
		return ge(a, b)
	default:
		return nil, fmt.Errorf("unrecognized operation %q", op)
	}
}

// ge returns true if a >= b
func ge(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return _bool(va >= vb), nil
		}
	case string:
		switch vb := b.(type) {
		case string:
			return _bool(va >= vb), nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T >= %T", a, b)
}

// gt returns true if a > b
func gt(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return _bool(va > vb), nil
		}
	case string:
		switch vb := b.(type) {
		case string:
			return _bool(va > vb), nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T > %T", a, b)
}

// le returns true if a <= b
func le(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return _bool(va <= vb), nil
		}
	case string:
		switch vb := b.(type) {
		case string:
			return _bool(va <= vb), nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T <= %T", a, b)
}

// lt returns true if a < b
func lt(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return _bool(va < vb), nil
		}
	case string:
		switch vb := b.(type) {
		case string:
			return _bool(va < vb), nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T < %T", a, b)
}

// eq returns true if a == b
func eq(a, b interface{}) (interface{}, error) {
	// Short path in case and b are the same object.
	if a == b {
		return true, nil
	}

	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return _bool(va == vb), nil
		}
	case string:
		if vb, ok := b.(string); ok {
			return _bool(va == vb), nil
		}
	}

	return false, fmt.Errorf("can not evaluate %T == %T", a, b)
}

// xor returns a ^ b
func xor(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va ^ vb, nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T ^ %T", a, b)
}

// or returns a | b
func or(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va | vb, nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T | %T", a, b)
}

// and returns a & b
func and(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va & vb, nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T & %T", a, b)
}

// shr returns a >> b
func shr(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va >> uint64(vb), nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T >> %T", a, b)
}

// shl returns a << b
func shl(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va << uint64(vb), nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T << %T", a, b)
}

// add returns a + b
func add(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va + vb, nil
		case string:
			return string(rune(va)) + vb, nil
		}
	case string:
		switch vb := b.(type) {
		case int64:
			return va + string(rune(vb)), nil
		case string:
			return va + vb, nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T + %T", a, b)
}

// sub returns a - b
func sub(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va - vb, nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T - %T", a, b)
}

// mul returns a * b
func mul(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va * vb, nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T * %T", a, b)
}

// div returns a / b
func div(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va / vb, nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T / %T", a, b)
}

// mod returns a % b
func mod(a, b interface{}) (interface{}, error) {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va % vb, nil
		}
	}
	return nil, fmt.Errorf("can not evaluate %T %% %T", a, b)
}

// _bool returns the canonical integer representation of the given bool.
func _bool(x bool) int64 {
	if x {
		return -1
	}
	return 0
}
