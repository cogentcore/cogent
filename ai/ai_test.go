package main

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/ddkwork/golibrary/mylog"
	"os"
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
	Models := make([]Model, 0)
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		title := s.Find("h2").Text()
		if title == "" {
			return
		}
		title = unescape(title)
		description := s.Find("p").First().Text()
		Models = append(Models, Model{
			Name:        title,
			Description: description,
		})
	})
}
