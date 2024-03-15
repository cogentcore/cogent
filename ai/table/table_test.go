package table

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTable3(t *testing.T) {
	//github.com/richardwilkes/gcs/v5
	type demoRow struct {
		parent       *demoRow
		data         string
		children     []*demoRow
		doubleHeight bool
	}

	root := NewNode("tableDemo", true, &demoRow{
		parent:       nil,
		data:         fmt.Sprintf("Row %d", 1),
		children:     make([]*demoRow, 0),
		doubleHeight: false,
	})

	for i := range 100 {
		if i%10 == 3 {
			for j := range 5 {
				child := NewNode("root", false, &demoRow{
					parent:       nil,
					data:         fmt.Sprintf("Sub Row %d", j+1),
					children:     make([]*demoRow, 0),
					doubleHeight: false,
				})
				root.AddChild(child)

				if j < 2 {
					child.SetOpen(true)
					for k := range child.children {
						child.AddChild(NewNode("", false, &demoRow{
							parent: nil,
							data:   fmt.Sprintf("Sub Sub Row %d", k+1),
						}))
					}
				}
			}
		}
	}
	mylog.Struct(root.Children())
}

func TestTable2(t *testing.T) {
	const topLevelRowsToMake = 100
	type demoRow struct {
		parent       *demoRow
		id           uuid.UUID
		text         string
		children     []*demoRow
		container    bool
		open         bool
		doubleHeight bool
	}

	rows := make([]*demoRow, topLevelRowsToMake)
	for i := range rows {
		row := &demoRow{
			id:   uuid.New(),
			text: fmt.Sprintf("Row %d", i+1),
		}
		if i%10 == 3 {
			if i == 3 {
				row.doubleHeight = true
			}
			row.container = true
			row.open = true
			row.children = make([]*demoRow, 5)
			for j := range row.children {
				child := &demoRow{
					parent: row,
					id:     uuid.New(),
					text:   fmt.Sprintf("Sub Row %d", j+1),
				}
				row.children[j] = child
				if j < 2 {
					child.container = true
					child.open = true
					child.children = make([]*demoRow, 2)
					for k := range child.children {
						child.children[k] = &demoRow{
							parent: child,
							id:     uuid.New(),
							text:   fmt.Sprintf("Sub Sub Row %d", k+1),
						}
					}
				}
			}
		}
		rows[i] = row
	}
	//table.SetRootRows(rows)
	mylog.Struct(rows)

}

func TestTable(t *testing.T) { //第三个就是更改抓包程序的数据存储格式为table
	type (
		Packed2 struct { // 1 2 ...
			Varint1 uint64 // 1
			Binary2 string // 2
			Binary3 string // 3
			Binary4 string // 4
		} // 1 2 ...
		Message2 struct { // 2
			Packed2 []Packed2 // 2
			Binary3 string    // 3  todo bug
		} // 2
		Group1 struct { // 1
			Binary1  string   // 1
			Message2 Message2 // 2
			Varint3  uint64   // 3
			Binary4  string   // 4
			Varint5  uint64   // 5
		} // 1
	)

	var Message = Group1{ //root container node
		Binary1: "game/system/session/info",
		Message2: Message2{ //container node,one child
			Packed2: []Packed2{ //container node two child
				{
					Varint1: 0,
					Binary2: "d3048a459417e6c0b7d39c971b99a58029f2720f7b2a70c992c826ce48184069",
					Binary3: "6593D03B-92BC-4BC6-BF54-D16BB0271AF9",
					Binary4: "Apple-iPhone10,3",
				},
				{
					Varint1: 1,
					Binary2: "",
					Binary3: "EAAIe5YPC68wBAPGE5l7JEtIE9BfPQmbcpQ92b0c9fD29vKn5ZCwHkutEjpX2PEcvyBLDqo15gNi1x0VN7U6d26QDaABEDaVzu3vuZBYKtvHH130O9Kna4742s6B8dtr1aKJUw7HuuyNWObpWYZCqBDGvypB7Js93oBISkwWrjbuYZAooY5vHLHFPyuIcLAcV8ZAX9sRFn3un18KZA0MEuDtgekU4ZBTJ1doEQmQ4DCLXwZDZD",
					Binary4: "_", // todo
				},
			},
			Binary3: "6593D03B-92BC-4BC6-BF54-D16BB0271AF9",
		},
		Varint3: 0,
		Binary4: "",
		Varint5: 0,
	}

	//https://521github.com/segmentio/encoding
	//https://github.com/golang/protobuf
	//https://521github.com/protocolbuffers/protobuf-go
	type (
		Field struct {
			Kind   reflect.Kind
			Value  reflect.Value
			number int
			child  []Field
		}
	)

	//Group1Node := NewNode("Group1", true, Field{
	//	Kind:  reflect.Struct,
	//	Value: reflect.ValueOf(Message),
	//	child: make([]Field, 0),
	//})
	fields := reflect.VisibleFields(reflect.TypeOf(Message))
	assert.Equal(t, 5, len(fields))
	//for i := range len(fields) {
	//	Group1Node.AddChild(NewNode("Group1", false, Field{
	//		Kind:  reflect.String,
	//		Value: reflect.ValueOf("game/system/session/info"),
	//		child: nil,
	//	}))
	//	node := NewNode("", true, Field{
	//		Kind:  reflect.Struct,
	//		Value: reflect.ValueOf(0),
	//		child: nil,
	//	})
	//	for i := range 2 { //todo assert it
	//		node.AddChild()
	//	}
	//
	//	Group1Node.AddChild(NewNode("Group1", false, Field{
	//		Kind:  reflect.Int,
	//		Value: reflect.ValueOf(""),
	//		child: nil,
	//	}))
	//	Group1Node.AddChild(NewNode("Group1", false, Field{
	//		Kind:  reflect.String,
	//		Value: reflect.Value{},
	//		child: nil,
	//	}))
	//	Group1Node.AddChild(NewNode("Group1", false, Field{
	//		Kind:  reflect.Int,
	//		Value: reflect.ValueOf(0),
	//		child: nil,
	//	}))
	//}
	//
	//Group1NodeEnd := NewNode("Group1End", true, Field{
	//	Kind:  reflect.Struct,
	//	Value: reflect.ValueOf(Message),
	//	child: make([]Field, 0),
	//})
	//
	//mylog.Struct(Message)

}

func Test_mock(t *testing.T) {
	type (
		obj struct {
			Index int
			Name  string
		}
	)
	o := obj{
		Index: 9,
		Name:  "ppp",
	}
	root := NewNode("", true, o)

	child1 := NewNode("", false, o)
	child2 := NewNode("", false, o)
	child3 := NewNode("", false, o)

	root.AddChild(child1)
	root.AddChild(child2)
	root.AddChild(child3)

	grandchild1 := NewNode("", false, o)
	grandchild2 := NewNode("", false, o)

	child1.AddChild(grandchild1)
	child1.AddChild(grandchild2)

	fmt.Println("tree:")
	root.Format(root)

	fmt.Println("Depth First Traversal:")
	root.Walk(func(node *Node[obj]) { //深度遍历
		fmt.Println(node.Data)
	})

	fmt.Println("Breadth First Traversal:")
	root.WalkContainer(func(node *Node[obj]) { //广度遍历
		fmt.Println(node.Data)
	})

	root.Sort(func(a, b obj) bool {
		return a.Index < b.Index
	})

	fmt.Println("Sorted tree:")
	root.Format(root)

	root.RemoveChild(child2.ID)

	fmt.Println("tree after removing child2:")
	root.Format(root)

	root.Update(grandchild1.ID, o)

	fmt.Println("tree after updating grandchild1:")
	root.Format(root)
}

func Test_main(t *testing.T) {
	root := NewNode("root", true, "root")

	child1 := NewNode("", false, "child1")
	child2 := NewNode("", false, "child2")
	child3 := NewNode("", false, "child3")

	root.AddChild(child1)
	root.AddChild(child2)
	root.AddChild(child3)

	grandchild1 := NewNode("", false, "grandchild1")
	grandchild2 := NewNode("", false, "grandchild2")

	child1.AddChild(grandchild1)
	child1.AddChild(grandchild2)

	fmt.Println("tree:")
	root.Format(root)

	fmt.Println("Depth First Traversal:")
	root.Walk(func(node *Node[string]) {
		fmt.Println(node.Data)
	})

	fmt.Println("Breadth First Traversal:")
	root.WalkContainer(func(node *Node[string]) {
		fmt.Println(node.Data)
	})

	root.Sort(func(a, b string) bool {
		return a < b
	})

	fmt.Println("Sorted tree:")
	root.Format(root)

	root.RemoveChild(child2.ID)

	fmt.Println("tree after removing child2:")
	root.Format(root)

	root.Update(grandchild1.ID, "updated_grandchild1")

	fmt.Println("tree after updating grandchild1:")
	println(root.Format(root))

	// ToGo(root)
}

func Test_mock2(t *testing.T) {
	type field struct {
		Name   string
		Number int
		Depth  int
		K      reflect.Kind
		Value  any
	}
	root := NewNode("", true, field{
		Name:   "x",
		Number: 0,
		Depth:  0,
		K:      0,
		Value:  nil,
	})
	Binary1 := NewNode("", false, field{
		Name:   "y",
		Number: 0,
		Depth:  0,
		K:      0,
		Value:  "game/system/session/info",
	})
	root.AddChild(Binary1)

	println(root.Format(root))
}

type ExplorerColumnId struct {
	Name    string
	Size    int
	Type    string
	ModTime time.Time
}
