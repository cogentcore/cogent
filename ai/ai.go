package main

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/coredom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/grows/jsons"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/xe"

	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/ddkwork/golibrary/mylog"
)

//go:embed models.json
var modelsJSON embed.FS

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

	grr.Log(jsons.OpenFS(ModelJSON, modelsJSON, "models.json"))
	giv.NewTableView(leftFrame).SetSlice(&ModelJSON.Children).SetReadOnly(true)

	newFrame := gi.NewFrame(leftFrame)
	newFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	gi.NewButton(newFrame).SetText("Update module list").OnClick(func(e events.Event) {
		mylog.Trace("start Update module list")
		QueryModelList()
		mylog.Success("Update module list finished")
	})

	gi.NewButton(newFrame).SetText("Run selected module").OnClick(func(e events.Event) {
		// model := Models[tableView.SelectedIndex]
		// cmd.RunArgs("ollama", "run", model.Name)//not need
	})
	gi.NewButton(newFrame).SetText("Stop selected module").OnClick(func(e events.Event) {
		// model := Models[tableView.SelectedIndex]
		// cmd.RunArgs("ollama", "stop",model.Name)//not need
	})

	rightFrame := gi.NewFrame(splits)
	rightFrame.Style(func(s *styles.Style) { s.Direction = styles.Column })
	splits.SetSplits(.2, .8)

	history := gi.NewFrame(rightFrame)
	history.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
	})

	prompt := gi.NewFrame(rightFrame)
	prompt.Style(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Align.Items = styles.Center
	})
	gi.NewButton(prompt).SetIcon(icons.Add)
	textField := gi.NewTextField(prompt).SetType(gi.TextFieldOutlined).SetPlaceholder("Enter a prompt here")
	textField.Style(func(s *styles.Style) { s.Max.X.Zero() })

	gi.NewButton(prompt).SetIcon(icons.Send).OnClick(func(e events.Event) {
		promptString := textField.Text()
		if promptString == "" {
			gi.MessageSnackbar(b, "Please enter a prompt")
			return
		}
		go func() {
			mylog.Warning("connect serve", "Send "+strconv.Quote(textField.Text())+" to the serve,please wait a while")
			// model := Models[tableView.SelectedIndex]
			resp, err := NewRequest(promptString, structs.Params{ // go1.22 Generic type constraints
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
				gi.ErrorSnackbar(b, err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				body := grr.Log1(io.ReadAll(resp.Body))
				gi.MessageSnackbar(b, fmt.Sprintf("Error getting response (%s): %s", resp.Status, body))
				return
			}
			scanner := bufio.NewScanner(resp.Body)
			allTokens := "**Cogent AI:** "

			history.AsyncLock()

			yourPrompt := gi.NewFrame(history)
			yourPrompt.Style(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
				s.Border.Radius = styles.BorderRadiusLarge
				s.Grow.Set(1, 0)
			})
			grr.Log(coredom.ReadMDString(coredom.NewContext(), yourPrompt, "**You:** "+promptString))

			answer := gi.NewFrame(history)
			answer.Style(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
				s.Border.Radius = styles.BorderRadiusLarge
				s.Grow.Set(1, 0)
			})

			history.Update()
			history.AsyncUnlock()

			for scanner.Scan() {
				token := HandleToken(scanner.Text())
				if token == "" {
					continue
				}
				allTokens += token

				answer.AsyncLock()
				answer.DeleteChildren()
				grr.Log(coredom.ReadMDString(coredom.NewContext(), answer, allTokens))
				answer.Update()
				answer.AsyncUnlock()
			}
			grr.Log(scanner.Err())
			grr.Log(resp.Body.Close())
		}()
	})

	b.RunMainWindow()
}
