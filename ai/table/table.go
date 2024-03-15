package table

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ddkwork/golibrary/stream"
	"github.com/google/uuid"
)

// RowData T 是内部node类型和其他所有外部的node类型，然而为避免样板代码，不需要将接口的类型约束为N个node的实现类型，
// 只需要在每个自定义node类型只需要实例化内部node类型并操作它的方法即可，这种情况下，自定义node没有没有嵌套内部node类型实现RowData接口，
// 所以不需要为node是否实现RowData接口进行签名实现检查
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
	Children() []*Node[T] //todo add calc sum method from CellData
	SetChildren(children []*Node[T])
	clearUnusedFields()
	AddChild(child *Node[T])
	Sort(cmp func(a T, b T) bool)
	Depth() int //todo
	InsertItem(parentID uuid.UUID, data T) *Node[T]
	CreateItem(parent *Node[T], data T) *Node[T]
	RemoveChild(id uuid.UUID)
	Update(id uuid.UUID, data T)
	Find(id uuid.UUID) *Node[T]
	Walk(callback func(node *Node[T]))
	WalkContainer(callback func(node *Node[T]))
	Format(root *Node[T]) string
	format(root *Node[T], prefix string, isLast bool, s *stream.Stream)
	formatData(rowObjectStruct any) (rowData string)
}

type (
	RowConstraint[T any] interface {
		comparable
		RowData[T]
	}
	Model[T RowConstraint[T]] interface {
		RootRowCount() int
		RootRows() []T
		SetRootRows(rows []T)
	}
	SimpleModel[T RowConstraint[T]] struct{ roots []T }
)

func (m *SimpleModel[T]) RootRowCount() int    { return len(m.roots) }
func (m *SimpleModel[T]) RootRows() []T        { return m.roots }
func (m *SimpleModel[T]) SetRootRows(rows []T) { m.roots = rows }

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
