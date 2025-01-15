// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
)

// PreviewPanel is a widget that displays an interactive live preview of a
// MD, HTML, or SVG file currently open.
type PreviewPanel struct {
	core.Frame

	// code is the parent [Code].
	code *Code
}

func (pp *PreviewPanel) Init() {
	pp.Frame.Init()
	pp.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	pp.Updater(func() {
		if pp.code == nil {
			return
		}
		pp.DeleteChildren()
		ed := pp.code.ActiveTextEditor()
		if ed == nil {
			core.NewText(pp).SetText("No open file")
			return
		}

		switch ed.Buffer.Info.Known {
		case fileinfo.Markdown:
			htmlcore.ReadMD(htmlcore.NewContext(), pp, ed.Buffer.Bytes())
		default:
			core.NewText(pp).SetText("The current file cannot be previewed")
		}
	})
}
