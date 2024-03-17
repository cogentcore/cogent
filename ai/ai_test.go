package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ddkwork/golibrary/stream"
	"github.com/stretchr/testify/assert"

	"cogentcore.org/cogent/ai/pkg/tree"
)

func Test_queryModelList(t *testing.T) {
	root := queryModelList(stream.NewReadFile("library.html"))

	root.SetHeader([]string{ //todo need calc max column depth and indent left
		"Name",
		"Size(GB)",
		"Hash",
		"UpdateTime",
		"Description",
	})

	root.SetFormatRowCallback(func(n *tree.Node[Model]) string { //table row need all field set left align,and set too long filed as cut+...
		fmtCommand := "%-25s. %-10.1f %-10s %-10s %-10s"
		if n.Container() {
			sum := 0.0
			n.WalkContainer(func(node *tree.Node[Model]) {
				sum += node.Data.Size
			})
			n.Data.Size = sum
			n.Data.Name = n.Type
			fmtCommand = "%-25s. %.1f %s %s %s"
		} else {
			n.Data.Description = ""
		}
		sprintf := fmt.Sprintf(fmtCommand,
			n.Data.Name,
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
			queryModelTags(stream.NewReadFile("tags.html"), node)
		case "llama2":
			queryModelTags(stream.NewReadFile("Tags · llama2.html"), node)
		}
	})
	stream.WriteTruncate("modelsTree.txt", root.Format(root))

	indent, err := json.MarshalIndent(ModelJson, "", "  ")
	assert.NoError(t, err)
	stream.WriteTruncate("root.json", indent)

	out, err := ModelMap.MarshalJSON()
	assert.NoError(t, err)
	stream.WriteTruncate("models.json", out)
}
