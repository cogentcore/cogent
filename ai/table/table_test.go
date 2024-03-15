package table

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestTable(t *testing.T) {
	//todo: add more test cases
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
