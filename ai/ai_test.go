package main

import (
	"strings"
	"testing"

	"cogentcore.org/cogent/ai/tree"
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"github.com/stretchr/testify/assert"
)

func Test_queryModelList(t *testing.T) {
	root := queryModelList(stream.NewReadFile("library.html"))

	gemmaNode := tree.NewTreeNode(Model{
		Name:        "gemma",
		Description: "",
		UpdateTime:  "",
		Hash:        "",
		Size:        "",
		Children:    nil,
	})

	queryModelTags(stream.NewReadFile("tags.html"), gemmaNode) //gemma bug

	//llama2Node := NewTreeNode(Model{
	//	Name:        "llama2",
	//	Description: "",
	//	UpdateTime:  "",
	//	Hash:        "",
	//	Size:        "",
	//	Children:    nil,
	//})
	//queryModelTags(stream.NewReadFile("Tags · llama2.html"), llama2Node) //llama2 todo this seem has a bug,need fix it

	println(root.Format(root)) //todo this need a treeTableView for show all tags in every model
	//todo save n-nar model tree to json, and when need update we should read from json file

	//todo need implement right format

	return
	resetModels()
	mylog.Struct(Models) //this is not well for show all tag,we should remove it
}

func TestName(t *testing.T) {
	//assert.True(t, strings.Contains("gemma", "gemma:7b-text-fp16"))
	assert.True(t, strings.Contains("gemma:7b-text-fp16", "gemma"))
}
