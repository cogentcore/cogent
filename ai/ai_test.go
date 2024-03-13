package main

import (
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"testing"
)

func Test_queryModelList(t *testing.T) {
	queryModelList(stream.NewReadFile("library.html"))
	queryModelTags(stream.NewReadFile("tags.html"))
	println(root.Format(root)) //todo this need a treeTableView for show all tags in every model

	return
	resetModels()
	mylog.Struct(Models)
}
