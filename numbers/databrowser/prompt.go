// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"golang.org/x/exp/maps"
)

// Prompt prompts the user for the values in given name-value map, calling
// the given function if the user clicks OK
func Prompt(ctx core.Widget, vals map[string]string, prompt string, fun func()) {
	d := core.NewBody().AddTitle(prompt)
	keys := maps.Keys(vals)
	slices.Sort(keys)
	for _, k := range keys {
		lbl := TrimOrderPrefix(k)
		core.NewText(d).SetText(lbl)
		tf := core.NewTextField(d).SetText(vals[k])
		tf.Style(func(s *styles.Style) {
			s.Min.X.Ch(60)
		})
		tf.OnChange(func(e events.Event) {
			vals[k] = tf.Text()
		})
	}
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			fun()
		})
	})
	d.RunDialog(ctx)
}
