package main

import (
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"testing"
)

func Test_queryModelList(t *testing.T) {
	resetModels()
	//queryModelList(stream.NewReadFile("library.html"))
	queryModelTags(stream.NewReadFile("tags.html"))
	mylog.Struct(Models)
}
