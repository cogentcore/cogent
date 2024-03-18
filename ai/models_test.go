package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"testing"

	"cogentcore.org/core/grr"
	"github.com/stretchr/testify/assert"

	"github.com/ddkwork/golibrary/pkg/tree"
	"github.com/ddkwork/golibrary/stream"
)

func Test_queryModelList(t *testing.T) {
	queryModelList(stream.NewReadFile("testdata/library.html"))

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
				//i finally understand the method what is CellDataForSort
				//also,we need add a method as CellDataForSum for table widget to display sum value
				sum += ParseUnitStr2GB(node.Data.Size)
			})
			n.Data.Size = strconv.FormatFloat(sum, 'f', 2, 64) + "GB"
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
			queryModelTags(stream.NewReadFile("testdata/tags_gemma.html"), node)

		case "llama2":
			queryModelTags(stream.NewReadFile("testdata/Tags_llama2.html"), node)
		}
	})
	stream.WriteTruncate("modelsTree.txt", root.Format(root))

	indent, err := json.MarshalIndent(root, "", "  ")
	assert.NoError(t, err)
	stream.WriteTruncate(jsonName, indent)
}

func ParseUnitStr2GB(data string) (value float64) {
	if data == "" {
		return
	}
	v := data[:len(data)-2] //2 is len gb or mb
	size, err := strconv.ParseFloat(v, 64)
	if grr.Log(err) != nil {
		return
	}
	unitStr := data[len(data)-2:]
	switch unitStr {
	case "GB":
		return size
	case "MB":
		return size / 1024
	default:
		slog.Error("unit is not GB or MB")
		return
	}
}

func TestParseFloatGB(t *testing.T) {
	assert.Equal(t, 1.5, ParseUnitStr2GB("1.5GB"))
	assert.Equal(t, 1.5/1024, ParseUnitStr2GB("1.5MB"))
	assert.Equal(t, 180.0, ParseUnitStr2GB("180GB"))
}
