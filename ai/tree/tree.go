package tree

//todo when this project passed, move this package to golibrary

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"github.com/google/uuid"
)

type Node[T any] struct {
	ID       uuid.UUID
	Data     T
	Parent   *Node[T]
	Children []*Node[T]
}

func NewTreeNode[T any](data T) *Node[T] {
	return &Node[T]{
		ID:       uuid.New(),
		Data:     data,
		Children: make([]*Node[T], 0),
	}
}

type Tree[T any] struct {
	Root *Node[T]
}

func New[T any](root T) *Tree[T] {
	return &Tree[T]{Root: NewTreeNode(root)}
}

func (t *Tree[T]) CreateItem(data T) *Node[T] {
	node := NewTreeNode(data)
	if reflect.TypeOf(data).Kind() == reflect.Slice {
		slice := reflect.ValueOf(data)
		for i := 0; i < slice.Len(); i++ {
			child := t.CreateItem(slice.Index(i).Interface().(T))
			node.AddChild(child)
		}
	}
	t.Root.AddChild(node)
	return node
}

func (t *Tree[T]) NodeSlice() []*Node[T] {
	var nodes []*Node[T]
	queue := []*Node[T]{t.Root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		nodes = append(nodes, node)
		for _, child := range node.Children {
			queue = append(queue, child)
		}
	}
	return nodes
}

func (n *Node[T]) AddChild(child *Node[T]) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

func (n *Node[T]) RemoveChild(id uuid.UUID) {
	for i, child := range n.Children {
		if child.ID == id {
			n.Children = append(n.Children[:i], n.Children[i+1:]...) //重新申请内存空间将杜绝nil pointer，但是会浪费内存和gc

			//Children := append(n.Children[:i], n.Children[i+1:]...)
			//n.Children = Children

			mylog.Warning("remove child,the child is nil pointer in memory")
			//slices.Delete(n.Children, i, i+1) //core style is nil maybe is this reason,we need check ki node's children is nil pointer
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
	for _, child := range n.Children {
		found := child.Find(id) //this is safe
		if found != nil {
			return found
		}
	}
	mylog.Error("node not found " + id.String())
	return nil
}

func (n *Node[T]) Sort(cmp func(a, b T) bool) {
	sort.SliceStable(n.Children, func(i, j int) bool {
		return cmp(n.Children[i].Data, n.Children[j].Data)
	})
	for _, child := range n.Children {
		if child == nil {
			mylog.Error("child == nil,maybe by RemoveChild method")
			continue
		}
		child.Sort(cmp)
	}
}

func (n *Node[T]) WalkDepth(callback func(node *Node[T])) { //this method can not be call reaped
	callback(n)
	for _, child := range n.Children {
		if child == nil {
			mylog.Error("child == nil,maybe by RemoveChild method")
			continue
		}
		child.WalkDepth(callback)
	}
}

func (n *Node[T]) WalkBreadth(callback func(node *Node[T])) { //this method can not be call reaped
	queue := []*Node[T]{n}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		callback(node)
		for _, child := range node.Children {
			queue = append(queue, child)
		}
	}
}

func (n *Node[T]) Format(root *Node[T]) string {
	s := stream.New("")
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
	//switch data := any(root.Data).(type) {
	//case EncodingFieldEditData:
	//	sprintf := fmt.Sprintf("%d. %s (%s): %v", data.Number, data.Name, data.Kind.String(), data.Value)
	//	s.WriteStringLn(sprintf)
	//}
	//sprintf := fmt.Sprintf("%d. %s (%s): %v", root.Data.Number, root.Data.Name, root.Data.Kind.String(), root.Data.Value)
	s.WriteStringLn(n.formatData(root.Data))

	for i := 0; i < len(root.Children); i++ {
		if root.Children[i] == nil {
			mylog.Error("root.Children[i] == nil,maybe by RemoveChild method")
			panic(111)
			continue
		}
		n.format(root.Children[i], prefix, i == len(root.Children)-1, s)
	}
}

func (n *Node[T]) formatData(rowObjectStruct any) (rowData string) {
	data := FormatDataForEdit(rowObjectStruct)
	data[0] += "."
	return strings.Join(data, "")
}

func (n *Node[T]) InsertItem(parentID uuid.UUID, data T) *Node[T] {
	parent := n.Find(parentID)
	if parent == nil {
		mylog.Error("parent node id not found")
		return n
	}
	child := NewTreeNode(data)
	parent.AddChild(child)
	return child
}

func (n *Node[T]) CreateItem(parent *Node[T], data T) *Node[T] {
	child := NewTreeNode(data)
	parent.AddChild(child)
	return n //todo test witch need return
}

func FormatDataForEdit(rowObjectStruct any) (rowData []string) {
	rowData = make([]string, 0)
	valueOf := reflect.ValueOf(rowObjectStruct)
	typeOf := reflect.Indirect(valueOf)
	if typeOf.Kind() != reflect.Struct {
		rowData = append(rowData, fmt.Sprint(rowObjectStruct))
		return
	}
	fields := reflect.VisibleFields(typeOf.Type())
	for i, field := range fields {
		field = field
		//mylog.Struct(field)
		v := valueOf.Field(i).Interface()
		switch t := v.(type) {
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
	return
}
