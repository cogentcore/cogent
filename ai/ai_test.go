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
		sprintf := fmt.Sprintf("%s. %s %s %s %s",
			n.Data.Name,
			n.Data.Description,
			n.Data.UpdateTime,
			n.Data.Hash,
			n.Data.Size,
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
