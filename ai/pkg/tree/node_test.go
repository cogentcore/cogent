package tree_test

import (
	"fmt"
	"reflect"
	"testing"

	"cogentcore.org/cogent/ai/pkg/tree"
)

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
	root := tree.NewNode("", true, o)

	child1 := tree.NewNode("", false, o)
	child2 := tree.NewNode("", false, o)
	child3 := tree.NewNode("", false, o)

	root.AddChild(child1)
	root.AddChild(child2)
	root.AddChild(child3)

	grandchild1 := tree.NewNode("", false, o)
	grandchild2 := tree.NewNode("", false, o)

	child1.AddChild(grandchild1)
	child1.AddChild(grandchild2)

	fmt.Println("tree:")
	root.Format(root)

	fmt.Println("Depth First Traversal:")
	root.Walk(func(node *tree.Node[obj]) { //深度遍历
		fmt.Println(node.Data)
	})

	fmt.Println("Breadth First Traversal:")
	root.WalkContainer(func(node *tree.Node[obj]) { //广度遍历
		fmt.Println(node.Data)
	})

	root.Sort(func(a, b obj) bool {
		return a.Index < b.Index
	})

	fmt.Println("Sorted tree:")
	root.Format(root)

	root.RemoveByID(child2.ID)

	fmt.Println("tree after removing child2:")
	root.Format(root)

	root.UpdateByID(grandchild1.ID, o)

	fmt.Println("tree after updating grandchild1:")
	root.Format(root)
}

func Test_main(t *testing.T) {
	root := tree.NewNode("root", true, "root")

	child1 := tree.NewNode("", false, "child1")
	child2 := tree.NewNode("", false, "child2")
	child3 := tree.NewNode("", false, "child3")

	root.AddChild(child1)
	root.AddChild(child2)
	root.AddChild(child3)

	grandchild1 := tree.NewNode("", false, "grandchild1")
	grandchild2 := tree.NewNode("", false, "grandchild2")

	child1.AddChild(grandchild1)
	child1.AddChild(grandchild2)

	fmt.Println("tree:")
	root.Format(root)

	fmt.Println("Depth First Traversal:")
	root.Walk(func(node *tree.Node[string]) {
		fmt.Println(node.Data)
	})

	fmt.Println("Breadth First Traversal:")
	root.WalkContainer(func(node *tree.Node[string]) {
		fmt.Println(node.Data)
	})

	root.Sort(func(a, b string) bool {
		return a < b
	})

	fmt.Println("Sorted tree:")
	root.Format(root)

	root.RemoveByID(child2.ID)

	fmt.Println("tree after removing child2:")
	root.Format(root)

	root.UpdateByID(grandchild1.ID, "updated_grandchild1")

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
	root := tree.NewNode("", true, field{
		Name:   "x",
		Number: 0,
		Depth:  0,
		K:      0,
		Value:  nil,
	})
	Binary1 := tree.NewNode("", false, field{
		Name:   "y",
		Number: 0,
		Depth:  0,
		K:      0,
		Value:  "game/system/session/info",
	})
	root.AddChild(Binary1)

	println(root.Format(root))
}

func TestExtractNodeDataFromList(t *testing.T) {
}

func TestNew(t *testing.T) {
}

func TestNewNode(t *testing.T) {
}

func TestNewUUID(t *testing.T) {
}

func TestNode_AddChild(t *testing.T) {
}

func TestNode_ApplyTo(t *testing.T) {
}

func TestNode_CellData(t *testing.T) {
}

func TestNode_CellFromCellData(t *testing.T) {
}

func TestNode_Children(t *testing.T) {
}

func TestNode_ChildrenElemByID(t *testing.T) {
}

func TestNode_ChildrenElemByIndex(t *testing.T) {
}

func TestNode_ChildrenElemByName(t *testing.T) {
}
func TestNode_ChildrenElemByPath(t *testing.T) {
}

func TestNode_ChildrenElemByType(t *testing.T) {
}

func TestNode_ChildrenLastElem(t *testing.T) {
}

func TestNode_ChildrenLen(t *testing.T) {
}

func TestNode_ChildrenMoveTo(t *testing.T) {
}

func TestNode_ChildrenRemoveByID(t *testing.T) {
}

func TestNode_ChildrenRemoveByIndex(t *testing.T) {
}

func TestNode_ChildrenRemoveByName(t *testing.T) {
}

func TestNode_ChildrenRemoveByPath(t *testing.T) {
}

func TestNode_ChildrenReset(t *testing.T) {
}

func TestNode_ChildrenSum(t *testing.T) {
}

func TestNode_ChildrenSwap(t *testing.T) {
}

func TestNode_Clone(t *testing.T) {
}

func TestNode_Container(t *testing.T) {
}

func TestNode_CopyFrom(t *testing.T) {
}

func TestNode_CreateItem(t *testing.T) {
}

func TestNode_Depth(t *testing.T) {
}
func TestNode_Enabled(t *testing.T) {

}

func TestNode_FindByID(t *testing.T) {

}

func TestNode_FindByIndex(t *testing.T) {

}

func TestNode_FindByName(t *testing.T) {

}

func TestNode_FindByPath(t *testing.T) {

}

func TestNode_FindByType(t *testing.T) {

}

func TestNode_Format(t *testing.T) {

}

func TestNode_GetType(t *testing.T) {

}

func TestNode_HasChildren(t *testing.T) {

}

func TestNode_InsertByID(t *testing.T) {

}

func TestNode_InsertByIndex(t *testing.T) {

}

func TestNode_InsertByName(t *testing.T) {

}

func TestNode_InsertByPath(t *testing.T) {

}

func TestNode_InsertByType(t *testing.T) {

}

func TestNode_InsertItem(t *testing.T) {

}

func TestNode_MarshalBinary(t *testing.T) {

}

func TestNode_MarshalJSON(t *testing.T) {

}

func TestNode_MarshalStruct(t *testing.T) {

}

func TestNode_MarshalText(t *testing.T) {

}

func TestNode_MarshalXML(t *testing.T) {

}

func TestNode_Match(t *testing.T) {

}

func TestNode_Open(t *testing.T) {

}

func TestNode_Parent(t *testing.T) {

}

func TestNode_RemoveByID(t *testing.T) {

}

func TestNode_RemoveByIndex(t *testing.T) {

}

func TestNode_RemoveByName(t *testing.T) {

}

func TestNode_RemoveByPath(t *testing.T) {

}

func TestNode_RemoveSelf(t *testing.T) {

}

func TestNode_SetChildren(t *testing.T) {

}

func TestNode_SetOpen(t *testing.T) {

}

func TestNode_SetParent(t *testing.T) {

}

func TestNode_SetType(t *testing.T) {

}

func TestNode_Sort(t *testing.T) {

}

func TestNode_String(t *testing.T) {

}

func TestNode_UUID(t *testing.T) {

}

func TestNode_UnmarshalBinary(t *testing.T) {

}

func TestNode_UnmarshalJSON(t *testing.T) {

}

func TestNode_UnmarshalStruct(t *testing.T) {

}

func TestNode_UnmarshalText(t *testing.T) {

}

func TestNode_UnmarshalXML(t *testing.T) {

}

func TestNode_UpdateByID(t *testing.T) {

}

func TestNode_UpdateByIndex(t *testing.T) {

}

func TestNode_UpdateByName(t *testing.T) {
}

func TestNode_UpdateByPath(t *testing.T) {
}

func TestNode_UpdateByType(t *testing.T) {
}

func TestNode_Walk(t *testing.T) {
}

func TestNode_WalkContainer(t *testing.T) {
}

func TestNode_clearUnusedFields(t *testing.T) {
}

func TestNode_format(t *testing.T) {
}

func TestNode_formatData(t *testing.T) {

}

func TestNode_kind(t *testing.T) {

}
