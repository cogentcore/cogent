package main

import (
	"fmt"
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
	doc.Find(".flex.px-4.py-3").Each(func(i int, s *goquery.Selection) {
		modelName := s.Find("href").Text()
		modelInfo := s.Find("span").Text()
		modelInfoSplit := strings.Split(modelInfo, " • ")
		modelHash := modelInfoSplit[0]
		modelSize := modelInfoSplit[1]
		modelUpdateTime := modelInfoSplit[2]

		fmt.Printf("Model Name: %s\n", modelName)
		fmt.Printf("Model Hash: %s\n", modelHash)
		fmt.Printf("Model Size: %s\n", modelSize)
		fmt.Printf("Model Update Time: %s\n", modelUpdateTime)
	})
}
