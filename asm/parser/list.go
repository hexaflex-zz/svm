package parser

import "strings"

// List defines a generic collection of nodes and is itself a node.
type List struct {
	*nodeBase
	children []Node
}

// NewList creates a new, empty list.
func NewList(pos Position, ntype Type) *List {
	return &List{
		nodeBase: newNodeBase(pos, ntype),
	}
}

// String returns a human readable string representation of the list.
func (l *List) String() string {
	var sb strings.Builder
	dumpNode(&sb, l, "")
	return sb.String()
}

// Len returns the number of nodes in the list.
func (l *List) Len() int {
	return len(l.children)
}

// Clear empties the list.
func (l *List) Clear() {
	l.children = l.children[:0]
}

// Append adds the given nodes to the end of the list.
func (l *List) Append(set ...Node) {
	l.ReplaceAt(l.Len(), set...)
}

// At returns the node at index n.
func (l *List) At(x int) Node {
	return l.children[x]
}

// Slice returns the list of child elements as a slice.
func (l *List) Slice() []Node {
	return l.children
}

// RemoveRange removes a range of nodes from the list.
func (l *List) RemoveRange(start, end int) {
	copy(l.children[start:], l.children[end+1:])

	end = len(l.children) - (end - start) - 1

	for i := end; i < len(l.children); i++ {
		l.children[i] = nil
	}

	l.children = l.children[:end]
}

// Remove removes the node at index from the list.
func (l *List) Remove(index int) {
	l.ReplaceAt(index)
}

// Copy returns a deep copy of this list and its contents.
func (l *List) Copy() Node {
	nn := NewList(l.pos, l.ntype)
	nn.children = make([]Node, len(l.children))

	for i := range l.children {
		nn.children[i] = l.children[i].Copy()
	}

	return nn
}

// IterFunc defines a handler used for every list element called by List.Each
type IterFunc func(int, Node) error

// Each calls f for each element in the list.
// Iteration stops of f returns an error.
func (l *List) Each(f IterFunc) error {
	for i, v := range l.children {
		if err := f(i, v); err != nil {
			return err
		}
	}
	return nil
}

// FilterFunc defines a handler used for every list element called by List.Filter
type FilterFunc func(int, Node) bool

// Filter calls f for each element in the list and removes the element
// from the list if f returns false.
func (l *List) Filter(f FilterFunc) {
	for i := 0; i < len(l.children); i++ {
		if f(i, l.children[i]) {
			continue
		}

		copy(l.children[i:], l.children[i+1:])
		l.children[len(l.children)-1] = nil
		l.children = l.children[:len(l.children)-1]
		i--
	}
}

// ReplaceFunc is used when replacing nodes in a list.
type ReplaceFunc func(int, Node) ([]Node, error)

// Replace calls f for each node in the list.
// If f returns a non-nil set of nodes, the current node is replaced
// with those which have been returned.
func (l *List) Replace(f ReplaceFunc) error {
	set := make([]Node, 0, l.Len())

	for i, v := range l.children {
		new, err := f(i, v)
		if err != nil {
			return err
		}

		if new != nil {
			set = append(set, new...)
		} else {
			set = append(set, v)
		}
	}

	l.children = set
	return nil
}

// ReplaceAt replaces the node at index with the given set.
// ReplaceAt(x, nil) is equivalent to Remove(x).
// ReplaceAt(len(List), set) is equivalent to Append(set...).
func (l *List) ReplaceAt(index int, set ...Node) {
	switch {
	case len(set) == 0:
		copy(l.children[index:], l.children[index+1:])
		l.children[len(l.children)-1] = nil
		l.children = l.children[:len(l.children)-1]

	case index >= l.Len():
		l.children = append(l.children, set...)

	case len(set) == 1:
		l.children[index] = set[0]

	default:
		out := append(l.children, set[1:]...)
		copy(out[index+len(set):], out[index+1:])
		copy(out[index:], set)
		l.children = out
	}
}

// InsertAt inserts one or more nodes at the given index.
func (l *List) InsertAt(index int, set ...Node) {
	if len(set) == 0 {
		return
	}

	out := append(l.children, set...)
	copy(out[index+len(set):], out[index:])
	copy(out[index:], set)
	l.children = out
}
