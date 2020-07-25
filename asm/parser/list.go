package parser

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

func (n *List) Len() int {
	return len(n.children)
}

func (n *List) Clear() {
	n.children = n.children[:0]
}

func (n *List) Append(set ...Node) {
	n.ReplaceAt(n.Len(), set...)
}

// At returns the node at index n.
func (n *List) At(x int) Node {
	return n.children[x]
}

// Slice returns the list of child elements as a slice.
func (n *List) Slice() []Node {
	return n.children
}

// RemoveRange removes a range of nodes from the list.
func (n *List) RemoveRange(start, end int) {
	copy(n.children[start:], n.children[end+1:])

	end = len(n.children) - (end - start) - 1

	for i := end; i < len(n.children); i++ {
		n.children[i] = nil
	}

	n.children = n.children[:end]
}

// Remove removes the node at index from the list.
func (n *List) Remove(index int) {
	n.ReplaceAt(index)
}

// Copy returns a deep copy of this list and its contents.
func (n *List) Copy() Node {
	nn := NewList(n.pos, n.ntype)
	nn.children = make([]Node, len(n.children))

	for i := range n.children {
		nn.children[i] = n.children[i].Copy()
	}

	return nn
}

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
func (n *List) Replace(f ReplaceFunc) error {
	set := make([]Node, 0, n.Len())

	for i, v := range n.children {
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

	n.children = set
	return nil
}

// ReplaceAt replaces the node at index with the given set.
// ReplaceAt(x, nil) is equivalent to Remove(x).
// ReplaceAt(len(List), set) is equivalent to Append(set...).
func (n *List) ReplaceAt(index int, set ...Node) {
	switch {
	case len(set) == 0:
		copy(n.children[index:], n.children[index+1:])
		n.children[len(n.children)-1] = nil
		n.children = n.children[:len(n.children)-1]

	case index >= n.Len():
		n.children = append(n.children, set...)

	case len(set) == 1:
		n.children[index] = set[0]

	default:
		out := append(n.children, set[1:]...)
		copy(out[index+len(set):], out[index+1:])
		copy(out[index:], set)
		n.children = out
	}
}
