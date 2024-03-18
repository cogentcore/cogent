package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"

	"cogentcore.org/cogent/ai/pkg/tree"
)

type Model struct {
	Name       string
	Size       float64
	Hash       string
	UpdateTime string

	//json only,when the treeTableView widget implemented,it will be removed,
	//and we need implement treeTableView's json marshal and unmarshal method
	Children []Model //json need only

	Description string
}

var ModelJson = &Model{
	Name:        "root",
	Size:        0,
	Hash:        "",
	UpdateTime:  "",
	Description: "",
	Children:    make([]Model, 0),
}

func QueryModelList() {
	resp, err := http.Get("https://ollama.com/library")
	if !mylog.Error(err) {
		return
	}
	defer func() { mylog.Error(resp.Body.Close()) }()
	if resp.StatusCode != 200 {
		mylog.Error(fmt.Sprintf("status code error: %d %s", resp.StatusCode, resp.Status))
		return
	}
	root := queryModelList(resp.Body)
	root.WalkContainer(func(node *tree.Node[Model]) {
		children := QueryModelTags(node.Data.Name, node) //node is every container of model node
		ModelJson.Children = children
	})
	indent, err := json.MarshalIndent(ModelJson, "", "  ")
	if !mylog.Error(err) {
		return
	}
	stream.WriteTruncate("models.json", indent)
}

func queryModelList(r io.Reader) (root *tree.Node[Model]) {
	root = tree.NewNode("root", true, Model{
		Name:        "root",
		Description: "",
		UpdateTime:  "",
		Hash:        "",
		Size:        0,
	})
	doc, err := goquery.NewDocumentFromReader(r)
	if !mylog.Error(err) {
		return
	}
	doc.Find("a.group").Each(func(i int, s *goquery.Selection) {
		name := s.Find("h2.mb-3").Text()
		name = unescape(name)
		description := s.Find("p.mb-4").First().Text()
		model := Model{
			Name:        name,
			Size:        0,
			Hash:        "",
			UpdateTime:  "",
			Description: description,
			Children:    make([]Model, 0),
		}

		//json only,when the treeTableView widget implemented,it will be removed,
		//and we need implement treeTableView's json marshal and unmarshal method
		ModelJson.Children = append(ModelJson.Children, model)

		parent := tree.NewNode(name, true, model)
		root.AddChild(parent)
	})
	return
}

func QueryModelTags(name string, parent *tree.Node[Model]) (children []Model) {
	url := "https://ollama.com/library/" + name + "/tags" //todo found a bug with name from log
	mylog.Warning("update model tags", url)
	defer func() { mylog.Success("update model tags done", url) }()
	resp, err := http.Get(url)
	if !mylog.Error(err) {
		return
	}
	defer func() { mylog.Error(resp.Body.Close()) }()
	return queryModelTags(resp.Body, parent)
}

func queryModelTags(r io.Reader, parent *tree.Node[Model]) (children []Model) {
	doc, err := goquery.NewDocumentFromReader(r)
	if !mylog.Error(err) {
		return
	}
	children = make([]Model, 0)
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
				mylog.Error("not find model name in tags")
				return
			}
		}

		fnFindModelName()

		modelInfo := s.Find("span").Text()
		lines, ok := stream.New(modelInfo).ToLines()
		if !ok {
			mylog.Error("modelInfo ToLines not ok")
			return
		}
		modelInfoSplit := strings.Split(lines[1], " • ")

		if strings.Contains(modelWithTag, parent.Data.Name) {
			//mylog.Trace("modelInfoSplit[1]", modelInfoSplit[1])
			//sizeValue := strings.TrimSuffix(modelInfoSplit[1], "GB") //todo bug, not all size is GB,it may be MB,need add unit to colum
			LenSizeStr := len(modelInfoSplit[1])
			sizeValue := modelInfoSplit[1][:LenSizeStr-2] //2 is len gb or mb
			//mylog.Trace("sizeValue", sizeValue)
			size, err := strconv.ParseFloat(sizeValue, 64)
			if !mylog.Error(err) {
				return
			}
			model := Model{
				//Name: parent.Data.Name + ":" + tag,
				Name:        modelWithTag,
				Description: parent.Data.Description,
				UpdateTime:  strings.TrimSpace(lines[2]),
				Hash:        strings.TrimSpace(modelInfoSplit[0]),
				Size:        size,
			}
			parent.AddChild(tree.NewNode(modelWithTag, false, model))
			//model.Description = ""//todo why not done? we only need show description in container node
			clone := model
			clone.Description = "" //not working,why? this is every child here
			//json only,when the treeTableView widget implemented,it will be removed,
			//and we need implement treeTableView's json marshal and unmarshal method
			children = append(children, clone)
		}
	})
	return
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
