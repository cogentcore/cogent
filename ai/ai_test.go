package main

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/ddkwork/golibrary/mylog"
	"os"
	"strings"
	"testing"
)

func Test_queryModelTags(t *testing.T) {
	//queryModelTags("gemma")
	file, err := os.ReadFile("tags.html")
	if !mylog.Error(err) {
		return
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(file))
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
