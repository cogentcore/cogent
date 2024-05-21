// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/views"
)

// Prompt prompts the user for the values in given struct (pass a pointer),
// calling the given function if the user clicks OK.
func Prompt(ctx core.Widget, stru any, prompt string, fun func()) {
	d := core.NewBody().AddTitle(prompt)
	views.NewStructView(d).SetStruct(stru).Style(func(s *styles.Style) {
		s.Min.X.Ch(60)
	})
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			fun()
		})
	})
	d.RunDialog(ctx)
}
