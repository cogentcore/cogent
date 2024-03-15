package main

import (
	"testing"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"

	"cogentcore.org/cogent/ai/table"
)

func Test_queryModelList(t *testing.T) {
	root := queryModelList(stream.NewReadFile("library.html"))

	gemmaNode := table.NewNode("gemma", false, Model{
		Name:        "gemma",
		Description: "",
		UpdateTime:  "",
		Hash:        "",
		Size:        "",
		Children:    nil,
	})
	queryModelTags(stream.NewReadFile("tags.html"), gemmaNode)
	return

	llama2Node := table.NewNode("llama2", false, Model{
		Name:        "llama2",
		Description: "",
		UpdateTime:  "",
		Hash:        "",
		Size:        "",
		Children:    nil,
	})
	queryModelTags(stream.NewReadFile("Tags · llama2.html"), llama2Node)

	println(root.Format(root)) //todo this need a treeTableView for show all tags in every model
	//todo save n-nar model tree to json, and when need update we should read from json file

	//todo need implement right format

	return
	resetModels()
	mylog.Struct(Models) //this is not well for show all tag,we should remove it
}
