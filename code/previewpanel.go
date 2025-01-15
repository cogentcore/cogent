// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"bytes"

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

	// lastRendered is the content that was last rendered in the preview.
	lastRendered []byte
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
		ed := pp.code.ActiveTextEditor()
		if ed == nil {
			return
		}
		current := ed.Buffer.Bytes()
		if bytes.Equal(current, pp.lastRendered) {
			return
		}
		pp.lastRendered = current
		pp.DeleteChildren()

		switch ed.Buffer.Info.Known {
		case fileinfo.Markdown:
			htmlcore.ReadMD(htmlcore.NewContext(), pp, current)
		}
	})
}
