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
			gemmaNode.SetParent(root)
			queryModelTags(stream.NewReadFile("tags.html"), gemmaNode)
		case "llama2":
			llama2Node := tree.NewNode(node.Data.Name, true, Model{
				Name:        node.Data.Name,
				Description: node.Data.Description,
				UpdateTime:  "",
				Hash:        "",
				Size:        "",
			})
			llama2Node.SetParent(root)
			queryModelTags(stream.NewReadFile("Tags · llama2.html"), llama2Node)
		}
	})
	out, err := ModelMap.MarshalJSON()
	assert.NoError(t, err)
	stream.WriteTruncate("models.json", out) //todo test save all models to json file
	println(root.Format(root))
}
