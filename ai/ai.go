package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/xe"

	"github.com/PuerkitoBio/goquery"
	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"

	"cogentcore.org/cogent/ai/pkg/tree"
)

//go:generate go install .

func main() {
	b := gi.NewBody("Cogent AI")
	b.AddAppBar(func(tb *gi.Toolbar) {
		gi.NewButton(tb).SetText("Install") // todo set icon and merge ollama doc md files into s dom tree view
		gi.NewButton(tb).SetText("Start server").OnClick(func(e events.Event) {
			gi.ErrorSnackbar(b, xe.Verbose().Run("ollama", "serve"))
		})
		gi.NewButton(tb).SetText("Stop server").OnClick(func(e events.Event) {
			// todo kill thread ?
			// netstat -aon|findstr 11434
		})
		gi.NewButton(tb).SetText("Logs")                      // todo add a new windows show log and set ico
		gi.NewButton(tb).SetText("About").SetIcon(icons.Info) // todo add a new windows show some info
	})

	splits := gi.NewSplits(b)

	leftFrame := gi.NewFrame(splits)
	leftFrame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	if !mylog.Error(json.Unmarshal(stream.NewReadFile("ai/models.json").Bytes(), ModelJson)) {
		return
	}
	mylog.Struct(ModelJson)
	tableView := giv.NewTableView(leftFrame).SetSlice(&ModelJson) //todo bug: NewTableView can not set struct
	tableView.SetReadOnly(true)

	newFrame := gi.NewFrame(leftFrame)
	newFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	gi.NewButton(newFrame).SetText("Update module list").OnClick(func(e events.Event) {
		QueryModelList()
	})

	gi.NewButton(newFrame).SetText("Run selected module").OnClick(func(e events.Event) {
		// model := Models[tableView.SelectedIndex]
		// cmd.RunArgs("ollama", "run", model.Name)//not need
	})
	gi.NewButton(newFrame).SetText("Stop selected module").OnClick(func(e events.Event) {
		// model := Models[tableView.SelectedIndex]
		// cmd.RunArgs("ollama", "stop",model.Name)//not need
	})

	rightSplits := gi.NewSplits(splits)
	splits.SetSplits(.2, .8)

	frame := gi.NewFrame(rightSplits)
	frame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	answer := gi.NewFrame(frame)
	answer.Style(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
	})

	prompt := gi.NewFrame(frame)
	prompt.Style(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Align.Items = styles.Center
	})
	gi.NewButton(prompt).SetIcon(icons.Add)
	textField := gi.NewTextField(prompt).SetType(gi.TextFieldOutlined).SetPlaceholder("Enter a prompt here")
	textField.Style(func(s *styles.Style) { s.Max.X.Zero() })

	gi.NewButton(prompt).SetIcon(icons.Send).OnClick(func(e events.Event) {
		if textField.Text() == "" {
			mylog.Error("textField.Text() == \"\"")
			return
		}
		go func() {
			mylog.Warning("connect serve", "Send "+strconv.Quote(textField.Text())+" to the serve,please wait a while")
			// model := Models[tableView.SelectedIndex]
			resp, err := NewRequest(textField.Text(), structs.Params{ // go1.22 Generic type constraints
				// ApiModel: model.Name,
				ApiModel:    "gemma:2b",
				ApiKey:      "",
				Provider:    "",
				Temperature: "",
				Top_p:       "",
				Max_length:  "1111111",
				Preprompt:   "",
				ThreadID:    "",
			}, "")
			if !mylog.Error(err) { // todo  timeout ? need set it
				return
			}
			scanner := bufio.NewScanner(resp.Body)
			allToken := ""
			for scanner.Scan() {
				token := HandleToken(scanner.Text())
				if token == "" {
					continue
				}
				print(token)

				answer.AsyncLock()
				answer.DeleteChildren() //todo can not save chat history

				//reset answer style
				answer.Style(func(s *styles.Style) {
					s.Direction = styles.Column
				})

				//  need save chat list layout for show chat history
				you := gi.NewFrame(answer)
				you.Style(func(s *styles.Style) {
					s.Direction = styles.Row
				})
				gi.NewLabel(you).SetText("you:").Style(func(s *styles.Style) { //todo NewLabel seems can not set svg icon
					s.Align.Self = styles.Start
				})
				youSend := gi.NewTextField(you).SetType(gi.TextFieldOutlined) //todo need more type
				youSend.SetText(textField.Text())                             //todo if we send code block or md need highlight it
				youSend.Style(func(s *styles.Style) {
					s.Align.Self = styles.End
				})
				//todo need support emoji  😅 😁 😍 🤥 https://www.emojiall.com/zh-hans/categories/A

				ai := gi.NewFrame(answer)
				ai.Style(func(s *styles.Style) {
					s.Direction = styles.Row
				})
				gi.NewLabel(ai).SetText("ai:").Style(func(s *styles.Style) {
					s.Align.Self = styles.Start
				})

				//now need given ReadMDString a NewFrame? and set s.Align.Self = styles.End ?
				mdFrame := gi.NewFrame(answer) //todo rename answer as chatPair
				mdFrame.Style(func(s *styles.Style) {
					s.Align.Self = styles.End
				})

				allToken += token

				if !mylog.Error(coredom.ReadMDString(coredom.NewContext(), mdFrame, allToken)) {
					return
				}
				answer.Update()
				answer.AsyncUnlock()
			}
			mylog.Error(scanner.Err())
			// mylog.Error(resp.Body.Close()) //not do,unknown reason
		}()
	})

	rightSplits.SetSplits(.6, .4)

	b.RunMainWindow()
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
		ModelJson.Children = append(ModelJson.Children, model)
		parent := tree.NewNode(name, true, model)
		root.AddChild(parent)
	})
	return
}

func QueryModelTags(name string, parent *tree.Node[Model]) (children []Model) {
	resp, err := http.Get("https://ollama.com/library/" + name + "/tags")
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
			sizeValue := strings.TrimSuffix(modelInfoSplit[1], "GB")
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
			children = append(children, model)
			//mylog.Struct(model)
			parent.AddChild(tree.NewNode(modelWithTag, false, model))
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

type (
	Model struct {
		Name        string
		Size        float64
		Hash        string
		UpdateTime  string
		Description string
		Children    []Model
	}
)

var ModelJson = &Model{
	Name:        "root",
	Size:        0,
	Hash:        "",
	UpdateTime:  "",
	Description: "",
	Children:    make([]Model, 0),
}
