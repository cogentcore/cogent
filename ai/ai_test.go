package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"strings"
	"testing"
)

func Test_queryModelTags(t *testing.T) {
	//queryModelTags("gemma")
	doc, err := goquery.NewDocumentFromReader(stream.NewReadFile("tags.html"))
	if !mylog.Error(err) {
		return
	}

	Models = make([]Model, 0)

	doc.Find("a.group").Each(func(i int, s *goquery.Selection) {
		Name := s.Find(".break-all").Text()
		modelInfo := s.Find("span").Text()
		lines, ok := stream.New(modelInfo).ToLines()
		if !ok {
			return
		}
		modelInfoSplit := strings.Split(lines[1], " • ")
		Models = append(Models, Model{
			Name:        Name,
			Description: "", //todo merge description
			UpdateTime:  strings.TrimSpace(lines[2]),
			Hash:        strings.TrimSpace(modelInfoSplit[0]),
			Size:        modelInfoSplit[1],
		})
	})
	println()
}
