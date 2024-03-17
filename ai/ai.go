package main

import (
	"bufio"
	"encoding/json"
	"strconv"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/xe"

	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
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
