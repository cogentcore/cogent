package tree

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"github.com/google/uuid"
)

type (
	Provider[T any] interface {
		Walk(callback func(node *Node[T]))
		WalkContainer(callback func(node *Node[T]))
		Clone(newParent *Node[T], preserveID bool) *Node[T]
		CopyFrom(from *Node[T])
		ApplyTo(to *Node[T])
		UUID() uuid.UUID
		kind(base string) string
		Depth() int
		Parent() *Node[T]
		SetParent(parent *Node[T])
		Sort(cmp func(a T, b T) bool)
		RemoveByID(id uuid.UUID)
		RemoveByName(id uuid.UUID)
		RemoveByIndex(id uuid.UUID)
		RemoveSelf(id uuid.UUID)
		RemoveByPath(id uuid.UUID)
		AddChild(child *Node[T])
		InsertItem(parentID uuid.UUID, data T) *Node[T]
		CreateItem(parent *Node[T], data T) *Node[T]
		InsertByID(id uuid.UUID) *Node[T]
		InsertByName(id uuid.UUID) *Node[T]
		InsertByIndex(id uuid.UUID) *Node[T]
		InsertByType(id uuid.UUID) *Node[T]
		InsertByPath(id uuid.UUID) *Node[T]
		FindByID(id uuid.UUID) *Node[T]
		FindByName(id uuid.UUID) *Node[T]
		FindByIndex(id uuid.UUID) *Node[T]
		FindByPath(id uuid.UUID) *Node[T]
		FindByType(id uuid.UUID) *Node[T]
		UpdateByID(id uuid.UUID, data T)
		UpdateByName(id uuid.UUID, data T)
		UpdateByIndex(id uuid.UUID, data T)
		UpdateByPath(id uuid.UUID, data T)
		UpdateByType(id uuid.UUID, data T)
		Container() bool
		HasChildren() bool
		Children() []*Node[T]
		SetChildren(children []*Node[T])
		clearUnusedFields()
		GetType() string
		SetType(t string)
		Open() bool
		SetOpen(open bool)
		ChildrenLen()
		ChildrenSwap()
		ChildrenMoveTo()
		ChildrenLastElem()
		ChildrenElemByID()
		ChildrenElemByName()
		ChildrenElemByIndex()
		ChildrenElemByType()
		ChildrenElemByPath()
		ChildrenReset()
		ChildrenRemoveByID()
		ChildrenRemoveByName()
		ChildrenRemoveByIndex()
		ChildrenRemoveByPath()
		ChildrenSum(parent *Node[T]) *Node[T]
		SetHeader(header []string)
		SetFormatRowCallback(formatRowCallback func(*Node[T]) string)
		Format(root *Node[T]) string
		format(root *Node[T], prefix string, isLast bool, s *stream.Stream)
		String() string
		Enabled() bool
		Marshaller[T]
		Unmarshaler[T]
	}
	Marshaller[T any] interface {
		xml.Marshaler
		json.Marshaler
		encoding.TextMarshaler
		encoding.BinaryMarshaler
		Marshal(objectPtr any) (Provider[T], error)
	}
	Unmarshaler[T any] interface {
		xml.Unmarshaler
		json.Unmarshaler
		encoding.TextUnmarshaler
		encoding.BinaryUnmarshaler
		Unmarshal(tree Provider[T]) (objectPtr any, err error)
	}
	Node[T any] struct {
		ID                uuid.UUID `json:"id"`
		Data              T
		Type              string     `json:"type"`
		IsOpen            bool       `json:"open,omitempty"`     // Container only
		children          []*Node[T] `json:"children,omitempty"` // Container only
		parent            *Node[T]
		formatRowCallback func(root *Node[T]) string
		header            []string
	}
)

const ContainerKeyPostfix = "_container"

func New[T any]() Provider[T] {
	return &Node[T]{}
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

func (n *Node[T]) Clone(newParent *Node[T], preserveID bool) *Node[T] {
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

func (n *Node[T]) UUID() uuid.UUID { return n.ID }

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

func (n *Node[T]) Parent() *Node[T]          { return n.parent }
func (n *Node[T]) SetParent(parent *Node[T]) { n.parent = parent }

func (n *Node[T]) Sort(cmp func(a, b T) bool) {
	sort.SliceStable(n.children, func(i, j int) bool {
		return cmp(n.children[i].Data, n.children[j].Data)
	})
	for _, child := range n.children {
		child.Sort(cmp)
	}
}

func (n *Node[T]) RemoveByID(id uuid.UUID) {
	for i, child := range n.children {
		if child.ID == id {
			n.children = slices.Delete(n.children, i, i+1)
			break
		}
	}
}

func (n *Node[T]) RemoveByName(id uuid.UUID) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) RemoveByIndex(id uuid.UUID) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) RemoveSelf(id uuid.UUID) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) RemoveByPath(id uuid.UUID) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) AddChild(child *Node[T]) {
	child.parent = n
	n.children = append(n.children, child)
}

func (n *Node[T]) InsertItem(parentID uuid.UUID, data T) *Node[T] {
	parent := n.FindByID(parentID)
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

func (n *Node[T]) InsertByID(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) InsertByName(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) InsertByIndex(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) InsertByType(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) InsertByPath(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) FindByID(id uuid.UUID) *Node[T] {
	if n.ID == id {
		return n
	}
	for _, child := range n.children {
		found := child.FindByID(id)
		if found != nil {
			return found
		}
	}
	return nil
}
func (n *Node[T]) FindByName(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) FindByIndex(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) FindByPath(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) FindByType(id uuid.UUID) *Node[T] {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) UpdateByID(id uuid.UUID, data T) {
	node := n.FindByID(id)
	if node != nil {
		node.Data = data
	}
}

func (n *Node[T]) UpdateByName(id uuid.UUID, data T) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) UpdateByIndex(id uuid.UUID, data T) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) UpdateByPath(id uuid.UUID, data T) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) UpdateByType(id uuid.UUID, data T) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) Container() bool { return strings.HasSuffix(n.Type, ContainerKeyPostfix) }

func (n *Node[T]) HasChildren() bool { return n.Container() && len(n.children) > 0 }

func (n *Node[T]) Children() []*Node[T] { return n.children }

func (n *Node[T]) SetChildren(children []*Node[T]) {

	n.children = children
	//if n.dataAsNode.Container() {
	//	n.dataAsNode.SetChildren(ExtractNodeDataFromList(children))
	//	n.children = nil
	//}

}
func (n *Node[T]) clearUnusedFields() {
	if !n.Container() {
		n.children = nil
		n.IsOpen = false
	}
}

func (n *Node[T]) GetType() string  { return n.Type }
func (n *Node[T]) SetType(t string) { n.Type = t }

func (n *Node[T]) Open() bool        { return n.IsOpen && n.Container() }
func (n *Node[T]) SetOpen(open bool) { n.IsOpen = open && n.Container() }

func (n *Node[T]) ChildrenLen() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenSwap() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenMoveTo() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenLastElem() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenElemByID() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenElemByName() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenElemByIndex() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenElemByType() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenElemByPath() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenReset() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenRemoveByID() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenRemoveByName() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenRemoveByIndex() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenRemoveByPath() {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) ChildrenSum(parent *Node[T]) *Node[T] {
	//range   all children node
	//sum    every elem  node data
	//make container rowData and return

	rowData := make([]string, 0) //todo return it for show Container node in table row
	if parent.Container() {
		typeOf := reflect.TypeOf(parent.children[0])
		valueOf := reflect.ValueOf(parent.children[0])
		fields := reflect.VisibleFields(typeOf)
		for i := range fields {
			v := valueOf.Field(i).Interface()
			switch t := v.(type) { //todo add callback for format data
			case string:
				rowData = append(rowData, t)
			case int64:
				rowData = append(rowData, fmt.Sprint(t))
			case int:
				rowData = append(rowData, fmt.Sprint(t))
			case time.Time:
				rowData = append(rowData, stream.FormatTime(t))
			case time.Duration:
				rowData = append(rowData, fmt.Sprint(t))
			case reflect.Kind:
				rowData = append(rowData, t.String())
			case bool: // todo 不应该支持？数据库是否会有这种情况？
				rowData = append(rowData, fmt.Sprint(t))
			default: // any
				rowData = append(rowData, fmt.Sprint(t))
			}
		}
	}
	return n
}

func (n *Node[T]) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) MarshalJSON() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) MarshalText() (text []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) MarshalBinary() (data []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) Marshal(objectPtr any) (Provider[T], error) {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) UnmarshalJSON(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) UnmarshalText(text []byte) error {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) UnmarshalBinary(data []byte) error {
	//TODO implement me
	panic("implement me")
}

func (n *Node[T]) Unmarshal(tree Provider[T]) (objectPtr any, err error) {
	//TODO implement me
	panic("implement me")
}
func (n *Node[T]) SetHeader(header []string) {
	n.header = header
}
func (n *Node[T]) SetFormatRowCallback(formatRowCallback func(*Node[T]) string) {
	n.formatRowCallback = formatRowCallback
}

func (n *Node[T]) Format(root *Node[T]) string {
	s := stream.New("")

	if n.header != nil {
		for i, head := range n.header {
			if i == 0 {
				s.Indent(25) //todo indent lest from max container depth
			}
			s.WriteString(head)
			s.Indent(14) //todo max column width
		}
		s.NewLine()
	}

	n.format(root, "", true, s)
	return s.String()
}

func (n *Node[T]) format(root *Node[T], prefix string, isLast bool, s *stream.Stream) {
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
	if n.formatRowCallback != nil {
		s.WriteStringLn(n.formatRowCallback(root))
	}
	for i := 0; i < len(root.children); i++ {
		n.format(root.children[i], prefix, i == len(root.children)-1, s)
	}
}

func (n *Node[T]) String() string {
	return n.Type
	return fmt.Sprintf("%s Container", n.Type)
}

func (n *Node[T]) Enabled() bool {
	//TODO implement me
	panic("implement me")
}

func ExtractNodeDataFromList[T *Node[T]](list []*Node[T]) []T {
	dataList := make([]T, 0, len(list))
	for _, child := range list {
		dataList = append(dataList, child.Data)
	}
	return dataList
}
