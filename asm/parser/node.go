package parser

type Type int

// Known node types.
const (
	_ Type = iota
	AddressMode
	Ident
	String
	Number
	Operator
	Label
	ScopeBegin
	ScopeEnd
	Conditional
	Instruction
	Expression
	BreakPoint
	Constant
)

func (t Type) String() string {
	switch t {
	case Constant:
		return "Constant"
	case BreakPoint:
		return "BreakPoint"
	case AddressMode:
		return "AddressMode"
	case Ident:
		return "Ident"
	case String:
		return "String"
	case Number:
		return "Number"
	case Operator:
		return "Operator"
	case Label:
		return "Label"
	case ScopeBegin:
		return "ScopeBegin"
	case ScopeEnd:
		return "ScopeEnd"
	case Instruction:
		return "Instruction"
	case Expression:
		return "Expression"
	case Conditional:
		return "Conditional"
	}

	return ""
}

// Node represents a generic AST node.
type Node interface {
	Position() Position
	Type() Type
	Copy() Node
}

// nodeBase is embedded by concrete node types and ensures
// they qualify as a Node interface.
type nodeBase struct {
	pos   Position
	ntype Type
}

func newNodeBase(pos Position, ntype Type) *nodeBase {
	return &nodeBase{
		pos:   pos,
		ntype: ntype,
	}
}

func (n *nodeBase) Position() Position {
	return n.pos
}

func (n *nodeBase) Type() Type {
	return n.ntype
}
