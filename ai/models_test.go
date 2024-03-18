package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"github.com/stretchr/testify/assert"

	"cogentcore.org/cogent/ai/pkg/tree"
)

func Test_queryModelList(t *testing.T) {
	root := queryModelList(stream.NewReadFile("testdata/library.html"))

	root.SetHeader([]string{ //todo need calc max column depth and indent left
		"Name",
		"Size",
		"Hash",
		"UpdateTime",
		"Description",
	})

	root.SetFormatRowCallback(func(n *tree.Node[Model]) string { //table row need all field set left align,and set too long filed as cut+...
		fmtCommand := "%-25s %-10s %-10s %-10s %-10s"
		if n.Container() {
			//In addition, in the case of ERP financial software or bookkeeping system, purchase,
			//sale and inventory, and Excel summation, we will display SUM in the container node,
			//which automatically calculates today's turnover, workers' wages, total number of days of attendance, etc.,
			//and we will abandon the traditional outdated data statistics model.
			sum := 0.0
			n.WalkContainer(func(node *tree.Node[Model]) {
				unitStr := node.Data.Size[len(node.Data.Size)-2:] //2 is len gb or mb
				switch unitStr {
				case "GB":
				case "MB":
				default:
					mylog.Error("unit is not GB or MB")
					return
				}
				sizeValue := node.Data.Size[:len(node.Data.Size)-2]
				size, err := strconv.ParseFloat(sizeValue, 64)
				if !mylog.Error(err) {
					return
				}
				sum += size
			})
			n.Data.Size = strconv.FormatFloat(sum, 'f', 2, 64)
			n.Data.Name = n.Type
			fmtCommand = "%-25s %s %s %s %s"
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
			children := queryModelTags(stream.NewReadFile("testdata/tags_gemma.html"), node)
			ModelJson.Children[0].Children = children

		case "llama2":
			children := queryModelTags(stream.NewReadFile("testdata/Tags_llama2.html"), node)
			ModelJson.Children[1].Children = children
		}
	})
	stream.WriteTruncate("modelsTree.txt", root.Format(root))

	indent, err := json.MarshalIndent(ModelJson, "", "  ")
	assert.NoError(t, err)
	stream.WriteTruncate("models.json", indent)
}
