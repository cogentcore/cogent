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

	//todo this need rename columnCellData callback
	//  add table header,columnIDs and cell width logic
	root.SetFormatRowCallback(func(n *tree.Node[Model]) string { //table row need all field set left align,and set too long filed as cut+...
		fmtCommand := "%-25s. %s %s %-18s %s" //todo do not show Description and name,is it Container node only
		if n.Container() {
			n.Data.Name = n.Type
			fmtCommand = "%-25s. %s %s %s %s" //todo change field type and calculate children elem Size field sum show in container node
		} else {
			n.Data.Description = ""
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
			queryModelTags(stream.NewReadFile("tags.html"), node)
		case "llama2":
			queryModelTags(stream.NewReadFile("Tags · llama2.html"), node)
		}
	})
	stream.WriteTruncate("modelsTree.txt", root.Format(root))
	out, err := ModelMap.MarshalJSON()
	assert.NoError(t, err)
	stream.WriteTruncate("models.json", out)
}
