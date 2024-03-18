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
	root := queryModelList(stream.NewReadFile("testdata/library.html"))

	root.SetHeader([]string{ //todo need calc max column depth and indent left
		"Name",
		"Size", //this is not well,it maybe is mb, we need add a column to show unit
		"Hash",
		"UpdateTime",
		"Description",
	})

	root.SetFormatRowCallback(func(n *tree.Node[Model]) string { //table row need all field set left align,and set too long filed as cut+...
		fmtCommand := "%-25s %-10.1f %-10s %-10s %-10s"
		if n.Container() {
			//sum := 0.0
			n.WalkContainer(func(node *tree.Node[Model]) {
				/*
					//mylog.Trace("modelInfoSplit[1]", modelInfoSplit[1])
					//sizeValue := strings.TrimSuffix(modelInfoSplit[1], "GB") //todo bug, not all size is GB,it may be MB,need add unit to colum
					LenSizeStr := len(modelInfoSplit[1])
					sizeValue := modelInfoSplit[1][:LenSizeStr-2] //2 is len gb or mb
					//mylog.Trace("sizeValue", sizeValue)
					size, err := strconv.ParseFloat(sizeValue, 64)
					if !mylog.Error(err) {
						return
					}
				*/

				//sum += node.Data.Size //so this is not right in all model,need get unit is gb or mb
			})
			//n.Data.Size = sum
			n.Data.Name = n.Type
			fmtCommand = "%-25s %.1f %s %s %s"
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
