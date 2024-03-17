package main

import (
	"fmt"
	"testing"

	"github.com/ddkwork/golibrary/stream"
	"github.com/stretchr/testify/assert"

	"cogentcore.org/cogent/ai/pkg/tree"
)

func Test_queryModelList(t *testing.T) {
	root := queryModelList(stream.NewReadFile("library.html"))
	root.SetFormatRowCallback(func(n *tree.Node[Model]) string {
		fmtCommand := "%-25s. %s %s %-18s |%s" //todo do not show Description,is it Container node only
		if n.Container() {
			fmtCommand = "%-25s. %s %s %s |%s" //todo change field type and calculate children size sum
		}
		sprintf := fmt.Sprintf(fmtCommand,
			n.Data.Name, //todo swap struct field location
			n.Data.Size,
			n.Data.Hash,
			n.Data.UpdateTime,
			n.Data.Description,
		)
		return sprintf
	})
	root.WalkContainer(func(node *tree.Node[Model]) {
		switch node.Data.Name {
		case "gemma":
			gemmaNode := tree.NewNode(node.Data.Name, true, Model{
				Name:        node.Data.Name,
				Description: node.Data.Description,
				UpdateTime:  "",
				Hash:        "",
				Size:        "",
			})
			queryModelTags(stream.NewReadFile("tags.html"), gemmaNode) //root children[0] gemmaNode not append child
			//json, err := gemmaNode.MarshalJSON()
			//assert.NoError(t, err)
			//mylog.Json("", string(json))
			root.AddChild(gemmaNode)
		case "llama2":
			llama2Node := tree.NewNode(node.Data.Name, true, Model{
				Name:        node.Data.Name,
				Description: node.Data.Description,
				UpdateTime:  "",
				Hash:        "",
				Size:        "",
			})
			queryModelTags(stream.NewReadFile("Tags · llama2.html"), llama2Node)
			root.AddChild(llama2Node)
		}
	})
	stream.WriteTruncate("modelsTree.txt", root.Format(root))

	out, err := ModelMap.MarshalJSON()
	assert.NoError(t, err)
	stream.WriteTruncate("models.json", out)
}
