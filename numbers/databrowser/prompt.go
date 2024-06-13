// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
)

// PromptOKCancel prompts the user for whether to do something,
// calling the given function if the user clicks OK.
func PromptOKCancel(ctx core.Widget, prompt string, fun func()) {
	d := core.NewBody().AddTitle(prompt)
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			if fun != nil {
				fun()
			}
		})
	})
	d.RunDialog(ctx)
}

// PromptString prompts the user for a string value (initial value given),
// calling the given function if the user clicks OK.
func PromptString(ctx core.Widget, str string, prompt string, fun func(s string)) {
	d := core.NewBody().AddTitle(prompt)
	tf := core.NewTextField(d).SetText(str)
	tf.Styler(func(s *styles.Style) {
		s.Min.X.Ch(60)
	})
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			if fun != nil {
				fun(tf.Text())
			}
		})
	})
	d.RunDialog(ctx)
}

// PromptStruct prompts the user for the values in given struct (pass a pointer),
// calling the given function if the user clicks OK.
func PromptStruct(ctx core.Widget, stru any, prompt string, fun func()) {
	d := core.NewBody().AddTitle(prompt)
	core.NewForm(d).SetStruct(stru)
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			if fun != nil {
				fun()
			}
		})
	})
	d.RunDialog(ctx)
}
