package main

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"net/http"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlview"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/views"
	"cogentcore.org/core/xe"
	"cogentcore.org/core/xio/jsons"
	"github.com/ddkwork/golibrary/pkg/tree"

	"github.com/aandrew-me/tgpt/v2/structs"
)

//go:embed models.json
var rootJson embed.FS

var root = tree.NewNode("root", true, Model{
	Name:        "root",
	Description: "",
	UpdateTime:  "",
	Hash:        "",
	Size:        "",
})

const jsonName = "models.json"

func main() {
	b := core.NewBody("Cogent AI")
	b.AddAppBar(func(tb *core.Toolbar) {
		core.NewButton(tb).SetText("Install") // todo set icon and merge ollama doc md files into s dom tree view
		core.NewButton(tb).SetText("Start server").OnClick(func(e events.Event) {
			core.ErrorSnackbar(b, xe.Verbose().Run("ollama", "serve"))
		})
		core.NewButton(tb).SetText("Stop server").OnClick(func(e events.Event) {
			// todo kill thread ?
			// netstat -aon|findstr 11434
		})
		core.NewButton(tb).SetText("Logs")                      // todo add a new windows show log and set ico
		core.NewButton(tb).SetText("About").SetIcon(icons.Info) // todo add a new windows show some info
	})

	splits := core.NewSplits(b).SetSplits(0.2, 0.8)

	leftFrame := core.NewFrame(splits)
	leftFrame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	errors.Log(jsons.OpenFS(root, rootJson, jsonName))
	views.NewTableView(leftFrame).SetSlice(&root.Children).SetReadOnly(true)

	newFrame := core.NewFrame(leftFrame)
	newFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	core.NewButton(newFrame).SetText("Update module list").OnClick(func(e events.Event) {
		core.ErrorSnackbar(b, QueryModelList())
	})

	core.NewButton(newFrame).SetText("Run selected module").OnClick(func(e events.Event) {
		// model := Models[tableView.SelectedIndex]
		// cmd.RunArgs("ollama", "run", model.Name)//not need
	})
	core.NewButton(newFrame).SetText("Stop selected module").OnClick(func(e events.Event) {
		// model := Models[tableView.SelectedIndex]
		// cmd.RunArgs("ollama", "stop",model.Name)//not need
	})

	rightFrame := core.NewFrame(splits)
	rightFrame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	header := core.NewFrame(rightFrame)
	header.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Justify.Content = styles.Center
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
		s.Text.Align = styles.Center
	})

	core.NewLabel(header).SetType(core.LabelDisplayLarge).SetText("Cogent AI")
	core.NewLabel(header).SetType(core.LabelTitleLarge).SetText("Run powerful AI models locally")

	var send *core.Button
	var textField *core.TextField

	suggestionsFrame := core.NewFrame(header)
	suggestionsFrame.Style(func(s *styles.Style) {
		s.Justify.Content = styles.Center
		s.Grow.Set(0, 0)
	})

	suggestions := []string{"How do you call a function in Go?", "What is a partial derivative?", "Are apples healthy?"}

	for _, suggestion := range suggestions {
		core.NewButton(suggestionsFrame).SetText(suggestion).SetType(core.ButtonTonal).OnClick(func(e events.Event) {
			textField.SetText(suggestion)
			send.Send(events.Click, e)
		})
	}

	history := core.NewFrame(rightFrame)
	history.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
	})

	prompt := core.NewFrame(rightFrame)
	prompt.Style(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Align.Items = styles.Center
	})

	//todo we need change back "new topic" button

	textField = core.NewTextField(prompt).SetType(core.TextFieldOutlined).SetPlaceholder("Ask me anything")
	textField.Style(func(s *styles.Style) { s.Max.X.Zero() })
	textField.OnKeyChord(func(e events.Event) {
		if keyfun.Of(e.KeyChord()) == keyfun.Enter {
			send.Send(events.Click, e)
		}
	})

	send = core.NewButton(prompt).SetIcon(icons.Send)
	send.OnClick(func(e events.Event) {
		//todo prompt list and let ai support access local file service and access network
		//seems model unknown what is NPU computer

		//Which laptop has better battery life, LPU or NPU? Faster to reply to tokens? Also, whether they have VMX characteristics or not
		//go1.22 Generic type constraints

		//todo implement function with return d:/app app list and send to ai,let he tell us which app is we need choose
		//please help me find some app in d:/app and they has feature seems like set env

		//language translate
		//code translate
		//code comment translate
		//network access for spider, alse we need a not headless browser and api

		//Help me clean up the junk files on the C drive. The AI should let the user confirm whether to delete the junk files found or not
		//Help me remove all code comments in the xxx directory and remove blank lines
		//Help me find the download address of xxoo music or video
		//Tell me the code snippets that I use a lot
		//What is the lunar and new calendar today
		//XX days to oo days apart by a few days
		//Find out the cause of the memory leak and thread blocking caused by this code and suggest a solution to fix it
		//How to make the buttons in this GUI library have a CSS 3D button-like animation effect, please improve the button source code in the xx position
		//Help me translate all java files in the xxoo directory into Go language, and write the translated files in the corresponding directory, note that the suffix should be changed
		//...

		//todo gen png,also md can be show png canvas? and need test save png to local file

		//maybe these should let model do not us

		promptString := textField.Text()
		if promptString == "" {
			core.MessageSnackbar(b, "Please enter a prompt")
			return
		}
		textField.SetText("")

		if header.This() != nil {
			rightFrame.DeleteChild(header)
			rightFrame.Update()
		}

		yourPrompt := core.NewFrame(history)
		yourPrompt.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
			s.Border.Radius = styles.BorderRadiusLarge
			s.Grow.Set(1, 0)
		})
		errors.Log(htmlview.ReadMDString(htmlview.NewContext(), yourPrompt, "**You:** "+promptString))

		answer := core.NewFrame(history)
		answer.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
			s.Border.Radius = styles.BorderRadiusLarge
			s.Grow.Set(1, 0)
		})
		core.NewLabel(answer).SetText("<b>Cogent AI:</b> Loading...")

		history.Update()

		go func() {
			// model := Models[tableView.SelectedIndex]
			resp, err := NewRequest(promptString, structs.Params{
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
			if err != nil {
				core.ErrorSnackbar(b, err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				body := errors.Log1(io.ReadAll(resp.Body))
				core.MessageSnackbar(b, fmt.Sprintf("Error getting response (%s): %s", resp.Status, body))
				return
			}
			scanner := bufio.NewScanner(resp.Body)
			allTokens := "**Cogent AI:** "

			for scanner.Scan() {
				token, err := HandleToken(scanner.Text())
				if err != nil {
					core.ErrorSnackbar(b, err)
					continue
				}
				if token == "" {
					continue
				}
				allTokens += token

				answer.AsyncLock()
				answer.DeleteChildren()
				errors.Log(htmlview.ReadMDString(htmlview.NewContext(), answer, allTokens))
				answer.Update()
				history.ScrollDimToContentEnd(mat32.Y)
				answer.AsyncUnlock()
			}
			errors.Log(scanner.Err())
			errors.Log(resp.Body.Close())
		}()
	})

	b.RunMainWindow()
}
