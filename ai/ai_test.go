package main

import (
	"testing"

	"github.com/ddkwork/golibrary/stream"
	"github.com/stretchr/testify/assert"

	"cogentcore.org/cogent/ai/table"
)

func Test_queryModelList(t *testing.T) {
	root := queryModelList(stream.NewReadFile("library.html"))
	root.WalkContainer(func(node *table.Node[Model]) {
		switch node.Data.Name {
		case "gemma":
			gemmaNode := table.NewNode(node.Data.Name, true, Model{
				Name:        node.Data.Name,
				Description: node.Data.Description,
				UpdateTime:  "",
				Hash:        "",
				Size:        "",
				Children:    nil,
			})
			gemmaNode.SetParent(root)
			queryModelTags(stream.NewReadFile("tags.html"), gemmaNode)
		case "llama2":
			llama2Node := table.NewNode(node.Data.Name, true, Model{
				Name:        node.Data.Name,
				Description: node.Data.Description,
				UpdateTime:  "",
				Hash:        "",
				Size:        "",
				Children:    nil,
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
