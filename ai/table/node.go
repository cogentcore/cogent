package table

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"github.com/google/uuid"
)

//var _ RowData[Node[any]] = (*Node)(nil) //todo why not pass any type here?

const ContainerKeyPostfix = "_container"

type Node[T any] struct {
	ID       uuid.UUID `json:"id"`
	Data     T
	Type     string     `json:"type"`
	IsOpen   bool       `json:"open,omitempty"`     // Container only
	children []*Node[T] `json:"children,omitempty"` // Container only
	parent   *Node[T]
}

func NewNode[T any](typeKey string, isContainer bool, data T) *Node[T] {
	if isContainer {
		typeKey += ContainerKeyPostfix
	}
	return &Node[T]{
		ID:       NewUUID(),
		Data:     data,
		Type:     typeKey,
		IsOpen:   isContainer,
		children: make([]*Node[T], 0),
		parent:   nil,
	}
}

func NewUUID() uuid.UUID {
	id, err := uuid.NewRandom()
	if !mylog.Error(err) {
		return uuid.UUID{}
	}
	return id
}

func (n *Node[T]) UUID() uuid.UUID { return n.ID }
func (n *Node[T]) Container() bool { return strings.HasSuffix(n.Type, ContainerKeyPostfix) }
func (n *Node[T]) kind(base string) string {
	if n.Container() {
		return fmt.Sprintf("%s Container", base)
	}
	return base
}
func (n *Node[T]) Depth() int {
	count := 0
	p := n.parent
	for p != nil {
		count++
		p = p.parent
	}
	return count
}
func (n *Node[T]) GetType() string                 { return n.Type }
func (n *Node[T]) SetType(t string)                { n.Type = t }
func (n *Node[T]) Open() bool                      { return n.IsOpen && n.Container() }
func (n *Node[T]) SetOpen(open bool)               { n.IsOpen = open && n.Container() }
func (n *Node[T]) Parent() *Node[T]                { return n.parent }
func (n *Node[T]) SetParent(parent *Node[T])       { n.parent = parent }
func (n *Node[T]) HasChildren() bool               { return n.Container() && len(n.children) > 0 }
func (n *Node[T]) Children() []*Node[T]            { return n.children }
func (n *Node[T]) SetChildren(children []*Node[T]) { n.children = children }
func (n *Node[T]) clearUnusedFields() {
	if !n.Container() {
		n.children = nil
		n.IsOpen = false
	}
}

func (n *Node[T]) AddChild(child *Node[T]) {
	child.parent = n
	n.children = append(n.children, child)
}

func (n *Node[T]) Sort(cmp func(a, b T) bool) {
	sort.SliceStable(n.children, func(i, j int) bool {
		return cmp(n.children[i].Data, n.children[j].Data)
	})
	for _, child := range n.children {
		child.Sort(cmp)
	}
}

func (n *Node[T]) InsertItem(parentID uuid.UUID, data T) *Node[T] {
	parent := n.Find(parentID)
	if parent == nil {
		return n
	}
	child := NewNode(parent.Type, false, data)
	parent.AddChild(child)
	return child
}

func (n *Node[T]) CreateItem(parent *Node[T], data T) *Node[T] {
	child := NewNode(parent.Type, false, data)
	parent.AddChild(child)
	return n //todo test witch need return
}

func (n *Node[T]) RemoveChild(id uuid.UUID) {
	for i, child := range n.children {
		if child.ID == id {
			n.children = slices.Delete(n.children, i, i+1)
			break
		}
	}
}

func (n *Node[T]) Update(id uuid.UUID, data T) {
	node := n.Find(id)
	if node != nil {
		node.Data = data
	}
}

func (n *Node[T]) Find(id uuid.UUID) *Node[T] {
	if n.ID == id {
		return n
	}
	for _, child := range n.children {
		found := child.Find(id)
		if found != nil {
			return found
		}
	}
	return nil
}

func (n *Node[T]) Walk(callback func(node *Node[T])) { //this method can not be call reaped
	callback(n)
	for _, child := range n.children {
		child.Walk(callback)
	}
}

func (n *Node[T]) WalkContainer(callback func(node *Node[T])) { //this method can not be call reaped
	queue := []*Node[T]{n}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		callback(node)
		for _, child := range node.children {
			queue = append(queue, child)
		}
	}
}

func (n *Node[T]) Format(root *Node[T]) string {
	s := stream.New("")
	n.format(root, "", true, s)
	return s.String()
}

func (n *Node[T]) format(root *Node[T], prefix string, isLast bool, s *stream.Stream) { //todo add callback for format data
	s.WriteString(fmt.Sprintf("%s", prefix))
	if isLast {
		s.WriteString("└───")
		prefix += "    "
		s.WriteString(prefix)
	} else {
		s.WriteString("├───")
		prefix += "│   "
		s.WriteString(prefix)
	}
	//switch data := any(root.Data).(type) {
	//case EncodingFieldEditData:
	//	sprintf := fmt.Sprintf("%d. %s (%s): %v", data.Number, data.Name, data.Kind.String(), data.Value)
	//	s.WriteStringLn(sprintf)
	//}
	//sprintf := fmt.Sprintf("%d. %s (%s): %v", root.Data.Number, root.Data.Name, root.Data.Kind.String(), root.Data.Value)
	s.WriteStringLn(n.formatData(root.Data))

	for i := 0; i < len(root.children); i++ {
		n.format(root.children[i], prefix, i == len(root.children)-1, s)
	}
}

func (n *Node[T]) formatData(rowObjectStruct any) (rowData string) {
	data := FormatDataForEdit(rowObjectStruct)
	data[0] += "."
	return strings.Join(data, "")
}

func (n *Node[T]) Clone(newParent *Node[T], preserveID bool) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) CellData(columnID int, data any) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) String() string {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) Enabled() bool {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) CopyFrom(from *Node[T]) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ApplyTo(to *Node[T]) {
	//TODO implement me
	panic("implement me")
}
