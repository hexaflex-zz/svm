package eval

import (
	"fmt"
	"testing"
)

const False = int64(0)
const True = int64(-1)

type evalTest struct {
	A, B interface{}
	Want interface{}
}

type evalTestSet struct {
	Op   string
	list []evalTest
}

// makeTests returns tests using all possible data type combinations.
// The values in v are the answers we expect to get with each test.
// A value if nil indicates the given type combination is not acceptable.
func makeTests(v ...interface{}) []evalTest {
	return []evalTest{
		{int64(123), int64(456), v[0]},
		{int64(123), "456", v[1]},
		{"123", int64(456), v[2]},
		{"123", "456", v[3]},
	}
}

func TestApply(t *testing.T) {
	for i, v := range []evalTestSet{
		{"+", makeTests(int64(579), "{456", "123ǈ", "123456")},
		{"-", makeTests(int64(-333), nil, nil, nil)},
		{"*", makeTests(int64(56088), nil, nil, nil)},
		{"/", makeTests(int64(0), nil, nil, nil)},
		{"%", makeTests(int64(123), nil, nil, nil)},
		{"<<", makeTests(int64(0), nil, nil, nil)},
		{">>", makeTests(int64(0), nil, nil, nil)},
		{"&", makeTests(int64(72), nil, nil, nil)},
		{"|", makeTests(int64(507), nil, nil, nil)},
		{"^", makeTests(int64(435), nil, nil, nil)},
		{"==", makeTests(int64(0), nil, nil, int64(0))},
		{"<", makeTests(int64(-1), nil, nil, int64(-1))},
		{"<=", makeTests(int64(-1), nil, nil, int64(-1))},
		{">", makeTests(int64(0), nil, nil, int64(0))},
		{">=", makeTests(int64(0), nil, nil, int64(0))},
	} {
		for ii, vv := range v.list {
			have, err := apply(v.Op, vv.A, vv.B)
			if err != nil {
				if vv.Want != nil || !equals(err, makeErr(v.Op, vv.A, vv.B)) {
					t.Fatalf("test %d/%d (%T(%v) %s %T(%v)):\nwant: %v\nhave: %v",
						i+1, ii+1, vv.A, vv.A, v.Op, vv.B, vv.B, vv.Want, err)
				}
				continue
			}

			if !equals(have, vv.Want) {
				t.Fatalf("test %d/%d (%T(%v) %s %T(%v)):\nwant: %T(%v)\nhave: %T(%v)",
					i+1, ii+1, vv.A, vv.A, v.Op, vv.B, vv.B, vv.Want, vv.Want, have, have)
			}
		}
	}
}

// makeErr construct an error for the given values.
func makeErr(op string, a, b interface{}) error {
	return fmt.Errorf("can not evaluate %T %s %T", a, op, b)
}

// equals returns true if a equals b.
// This performs typed comparisons.
func equals(a, b interface{}) bool {
	switch va := a.(type) {
	case error:
		if vb, ok := b.(error); ok {
			return va.Error() == vb.Error()
		}
	case string:
		if vb, ok := b.(string); ok {
			return va == vb
		}
	case int64:
		if vb, ok := b.(int64); ok {
			return va == vb
		}
	case bool:
		if vb, ok := b.(bool); ok {
			return va == vb
		}
	}
	return false
}
