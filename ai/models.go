package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/ddkwork/golibrary/pkg/tree"
	"github.com/ddkwork/golibrary/stream"
)

type Model struct {
	Name              string
	Size              string
	Hash              string
	UpdateTime        string
	tree.Node[*Model] //ContainerBase
	Description       string
}

func QueryModelList() error {
	resp, err := http.Get("https://ollama.com/library")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("error status code: %d (%s)", resp.StatusCode, resp.Status)
	}
	err = queryModelList(resp.Body)
	if err != nil {
		return err
	}
	root.WalkContainer(func(node *tree.Node[Model]) {
		//node is every container of model node
		e := QueryModelTags(node.Data.Name, node)
		if e != nil {
			err = e
		}
	})
	if err != nil {
		return err
	}
	indent, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	stream.WriteTruncate(jsonName, indent)
	return nil
}

func queryModelList(r io.Reader) error {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return err
	}
	doc.Find("a.group").Each(func(i int, s *goquery.Selection) {
		name := s.Find("h2.mb-3").Text()
		name = unescape(name)
		description := s.Find("p.mb-4").First().Text()
		model := Model{
			Name:        name,
			Size:        "",
			Hash:        "",
			UpdateTime:  "",
			Description: description,
		}
		parent := tree.NewNode(name, true, model)
		root.AddChild(parent)
	})
	return nil
}

func QueryModelTags(name string, parent *tree.Node[Model]) error {
	url := "https://ollama.com/library/" + name + "/tags" //todo bug skip root? why every model has run twice?
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return queryModelTags(resp.Body, parent)
}

func queryModelTags(r io.Reader, parent *tree.Node[Model]) error {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return err
	}
	doc.Find("a.group").Each(func(i int, s *goquery.Selection) {
		//tag := s.Find(".break-all").Text() //not need
		modelWithTag := ""
		fnFindModelName := func() {
			href, exists := s.Attr("href")
			if exists {
				// https://ollama.com/library/llama2:latest
				// /library/gemma:latest
				_, after, found := strings.Cut(href, "/library/")
				if !found {
					return
				}
				modelWithTag = after

			}
			if modelWithTag == "" {
				err = fmt.Errorf("did not find model name in tags")
				return
			}
		}

		fnFindModelName()

		modelInfo := s.Find("span").Text()
		lines, ok := stream.New(modelInfo).ToLines()
		if !ok {
			err = fmt.Errorf("modelInfo.ToLines not ok")
			return
		}
		modelInfoSplit := strings.Split(lines[1], " • ")

		if strings.Contains(modelWithTag, parent.Data.Name) {
			model := Model{
				//Name: parent.Data.Name + ":" + tag,
				Name:        modelWithTag,
				Description: parent.Data.Description,
				UpdateTime:  strings.TrimSpace(lines[2]),
				Hash:        strings.TrimSpace(modelInfoSplit[0]),
				Size:        modelInfoSplit[1],
			}
			parent.AddChild(tree.NewNode(modelWithTag, false, model))
		}
	})
	return err
}

func unescape(s string) string {
	return strings.NewReplacer(
		`\n`, "",
		"\n", "",
		` `, "",
		//`\\`, "",
		//`\n`, "\n",
		//`\r`, "\r",
		//`\t`, "\t",
		//`\"`, `"`,
		//`\u003e`, `>`,
		//`\u003c`, `<`,
		//`\u0026`, `&`,
	).Replace(s)
}
