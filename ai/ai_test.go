package main

import (
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"testing"
)

func Test_queryModelList(t *testing.T) {
	root := queryModelList(stream.NewReadFile("library.html"))

	gemmaNode := NewTreeNode(Model{
		Name:        "",
		Description: "",
		UpdateTime:  "",
		Hash:        "",
		Size:        "",
		Children:    nil,
	})

	queryModelTags(stream.NewReadFile("tags.html"), gemmaNode) //gemma bug

	llama2Node := NewTreeNode(Model{
		Name:        "",
		Description: "",
		UpdateTime:  "",
		Hash:        "",
		Size:        "",
		Children:    nil,
	})
	queryModelTags(stream.NewReadFile("Tags · llama2.html"), llama2Node) //llama2 todo this seem has a bug,need fix it
	println(root.Format(root))                                           //todo this need a treeTableView for show all tags in every model
	//todo save n-nar model tree to json, and when need update we should read from json file

	//todo need implement right format

	return
	resetModels()
	mylog.Struct(Models) //this is not well for show all tag,we should remove it
}
