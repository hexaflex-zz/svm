package parser

import "strings"

// Value defines a generic single string value.
type Value struct {
	*nodeBase
	Value string
}

func NewValue(pos Position, ntype Type, value string) *Value {
	return &Value{
		nodeBase: newNodeBase(pos, ntype),
		Value:    value,
	}
}

// Copy returns a copy of this value.
func (n *Value) Copy() Node {
	return NewValue(n.pos, n.ntype, n.Value)
}

// String returns a human readable string representation of the value.
func (n *Value) String() string {
	var sb strings.Builder
	dumpNode(&sb, n, "")
	return sb.String()
}
