// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
)

// PreviewPanel is a widget that displays an interactive live preview of a
// MD, HTML, or SVG file currently open.
type PreviewPanel struct {
	core.Frame

	// code is the parent [Code].
	code *Code

	// text is the text editor whose content we are previewing.
	text *TextEditor
}

func (pp *PreviewPanel) Init() {
	pp.Frame.Init()
	pp.Updater(func() {
		if pp.text == nil {
			return
		}
		pp.DeleteChildren()
		switch pp.text.Buffer.Info.Known {
		case fileinfo.Markdown:
			htmlcore.ReadMD(htmlcore.NewContext(), pp, pp.text.Buffer.Bytes())
		default:
			core.NewText(pp).SetText("The current file cannot be previewed")
		}
		pp.Update()
	})
}
