package main

import (
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"testing"
)

func Test_queryModelList(t *testing.T) {
	queryModelList(stream.NewReadFile("library.html")) //root
	queryModelTags(stream.NewReadFile("tags.html"))    //todo for the test we need add another tag html file
	println(root.Format(root))                         //todo this need a treeTableView for show all tags in every model
	//todo save n-nar model tree to json, and when need update we should read from json file

	return
	resetModels()
	mylog.Struct(Models) //this is not well for show all tag,we should remove it
}
