package table

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ddkwork/golibrary/stream"
	"github.com/google/uuid"
)

// RowData todo add depth method
type RowData[T any] interface {
	Clone(newParent *Node[T], preserveID bool) *Node[T]
	CellData(columnID int, data any)
	String() string
	Enabled() bool
	CopyFrom(from *Node[T])
	ApplyTo(to *Node[T])
	UUID() uuid.UUID
	Container() bool
	kind(base string) string
	GetType() string
	SetType(t string)
	Open() bool
	SetOpen(open bool)
	Parent() *Node[T]
	SetParent(parent *Node[T])
	HasChildren() bool
	Children() []*Node[T]
	SetChildren(children []*Node[T])
	clearUnusedFields()
	AddChild(child *Node[T])
	Sort(cmp func(a T, b T) bool)
	Format(root *Node[T]) string
	format(root *Node[T], prefix string, isLast bool, s *stream.Stream)
	InsertItem(parentID uuid.UUID, data T) *Node[T]
	CreateItem(parent *Node[T], data T) *Node[T]
	RemoveChild(id uuid.UUID)
	Update(id uuid.UUID, data T)
	Find(id uuid.UUID) *Node[T]
	Walk(callback func(node *Node[T]))
	WalkContainer(callback func(node *Node[T]))
	formatData(rowObjectStruct any) (rowData string)
}

type Model[T RowConstraint[T]] interface {
	RootRowCount() int
	RootRows() []T
	SetRootRows(rows []T)
}

type RowConstraint[T any] interface {
	comparable
	RowData[T]
}

type SimpleTableModel[T RowConstraint[T]] struct{ roots []T }

func (m *SimpleTableModel[T]) RootRowCount() int    { return len(m.roots) }
func (m *SimpleTableModel[T]) RootRows() []T        { return m.roots }
func (m *SimpleTableModel[T]) SetRootRows(rows []T) { m.roots = rows }

//func CollectUUIDsFromRow[T RowConstraint[T]](node T, ids map[uuid.UUID]bool) {
//	ids[node.UUID()] = true
//	for _, child := range node.NodeChildren() {
//		CollectUUIDsFromRow(child, ids)
//	}
//}

func AsNode[T any](in T) Node[T] { return any(in).(Node[T]) }

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

//func (t *Tree[T]) CreateItem(data T) *Node[T] {
//	node := NewTreeNode(data)
//	if reflect.TypeOf(data).Kind() == reflect.Slice {
//		slice := reflect.ValueOf(data)
//		for i := 0; i < slice.Len(); i++ {
//			child := t.CreateItem(slice.Index(i).Interface().(T))
//			node.AddChild(child)
//		}
//	}
//	t.Root.AddChild(node)
//	return node
//}
//
//func (t *Tree[T]) NodeSlice() []*Node[T] {
//	var nodes []*Node[T]
//	queue := []*Node[T]{t.Root}
//	for len(queue) > 0 {
//		node := queue[0]
//		queue = queue[1:]
//		nodes = append(nodes, node)
//		for _, child := range node.children {
//			queue = append(queue, child)
//		}
//	}
//	return nodes
//}
