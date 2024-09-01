// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command AI provides a GUI interface for running AI models locally.
package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"

	"github.com/aandrew-me/tgpt/v2/structs"
)

// //go:embed models.json
// var rootJson embed.FS

// var root = tree.NewNode("root", true, Model{
// 	Name:        "root",
// 	Description: "",
// 	UpdateTime:  "",
// 	Hash:        "",
// 	Size:        "",
// })

// const jsonName = "models.json"

func main() { // TODO(config)
	b := core.NewBody("Cogent AI")
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Install").SetIcon(icons.Download)
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Start server").SetIcon(icons.PlayArrow).OnClick(func(e events.Event) {
					core.ErrorSnackbar(b, exec.Verbose().Run("ollama", "serve"))
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Stop server").SetIcon(icons.Stop)
			})
		})
	})

	splits := core.NewSplits(b).SetSplits(0.2, 0.8) // TODO(config): use Plan in Cogent AI

	leftFrame := core.NewFrame(splits)
	leftFrame.Styler(func(s *styles.Style) { s.Direction = styles.Column })

	// errors.Log(jsonx.OpenFS(root, rootJson, jsonName))
	// core.NewTable(leftFrame).SetSlice(&root.Children).SetReadOnly(true)

	newFrame := core.NewFrame(leftFrame)
	newFrame.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	core.NewButton(newFrame).SetText("Update module list").OnClick(func(e events.Event) {
		// core.ErrorSnackbar(b, QueryModelList())
	})

	core.NewButton(newFrame).SetText("Run selected module").OnClick(func(e events.Event) {
		// model := Models[table.SelectedIndex]
		// cmd.RunArgs("ollama", "run", model.Name)//not need
	})
	core.NewButton(newFrame).SetText("Stop selected module").OnClick(func(e events.Event) {
		// model := Models[table.SelectedIndex]
		// cmd.RunArgs("ollama", "stop",model.Name)//not need
	})

	rightFrame := core.NewFrame(splits)
	rightFrame.Styler(func(s *styles.Style) { s.Direction = styles.Column })

	header := core.NewFrame(rightFrame)
	header.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.CenterAll()
	})

	core.NewText(header).SetType(core.TextDisplayLarge).SetText("Cogent AI")
	core.NewText(header).SetType(core.TextTitleLarge).SetText("Run powerful AI models locally")

	var send *core.Button
	var textField *core.TextField

	suggestionsFrame := core.NewFrame(header)
	suggestionsFrame.Styler(func(s *styles.Style) {
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
	history.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
	})

	prompt := core.NewFrame(rightFrame)
	prompt.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Align.Items = styles.Center
	})

	//todo we need change back "new topic" button

	textField = core.NewTextField(prompt).SetType(core.TextFieldOutlined).SetPlaceholder("Ask me anything")
	textField.Styler(func(s *styles.Style) { s.Max.X.Zero() })
	textField.OnKeyChord(func(e events.Event) {
		if keymap.Of(e.KeyChord()) == keymap.Enter {
			send.Send(events.Click, e)
		}
	})

	send = core.NewButton(prompt).SetIcon(icons.Send)
	send.OnClick(func(e events.Event) {
		promptString := textField.Text()
		if promptString == "" {
			core.MessageSnackbar(b, "Please enter a prompt")
			return
		}
		textField.SetText("")

		if header.This != nil {
			rightFrame.DeleteChild(header)
			rightFrame.Update()
		}

		yourPrompt := core.NewFrame(history)
		yourPrompt.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Background = colors.Scheme.SurfaceContainerLow
			s.Border.Radius = styles.BorderRadiusLarge
			s.Grow.Set(1, 0)
		})
		errors.Log(htmlcore.ReadMDString(htmlcore.NewContext(), yourPrompt, "**You:** "+promptString))

		answer := core.NewFrame(history)
		answer.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Background = colors.Scheme.SurfaceContainerLow
			s.Border.Radius = styles.BorderRadiusLarge
			s.Grow.Set(1, 0)
		})
		core.NewText(answer).SetText("<b>Cogent AI:</b> Loading...")

		history.Update()

		go func() {
			// model := Models[table.SelectedIndex]
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
				errors.Log(htmlcore.ReadMDString(htmlcore.NewContext(), answer, allTokens))
				answer.Update()
				history.ScrollDimToContentEnd(math32.Y)
				answer.AsyncUnlock()
			}
			errors.Log(scanner.Err())
			errors.Log(resp.Body.Close())
		}()
	})

	b.RunMainWindow()
}
