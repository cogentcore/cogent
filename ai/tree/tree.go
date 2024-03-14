package tree

//todo when this project passed, move this package to golibrary

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"
	"unsafe"

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
			//重新申请内存空间将杜绝nil pointer，但是会浪费内存和gc
			//但是为了减少后续代码频繁的判断node是nil影响可读性，我宁愿使用更多的内存和让gc更加繁忙
			//n.Children = append(n.Children[:i], n.Children[i+1:]...) //maybe has memory leaks,we need a unsafe method change Children'len

			origLen := len(n.Children)
			slices.Delete(n.Children, i, i+1) //clear for gc
			//no memory leaks,but gc has not been triggered
			//But gc's execution frequency doesn't immediately recall it,
			//so you need to make Len decremented so that subsequent code doesn't trigger the already-nil sliced member.
			sh := (*reflect.SliceHeader)(unsafe.Pointer(&n.Children)) //无法封装，返回后len没改变，只能所有地方都这样重复的写
			sh.Len = origLen - 1
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
		found := child.Find(id)
		if found != nil {
			return found
		}
	}
	return nil
}

func (n *Node[T]) Sort(cmp func(a, b T) bool) {
	sort.SliceStable(n.Children, func(i, j int) bool {
		return cmp(n.Children[i].Data, n.Children[j].Data)
	})
	for _, child := range n.Children {
		child.Sort(cmp)
	}
}

func (n *Node[T]) WalkDepth(callback func(node *Node[T])) { //this method can not be call reaped
	callback(n)
	for _, child := range n.Children {
		child.WalkDepth(callback)
	}
}

// WalkBranch Breadth is branching in the context of data structures.
func (n *Node[T]) WalkBranch(callback func(node *Node[T])) { //this method can not be call reaped
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
