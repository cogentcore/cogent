// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"net/url"
	"path"
	"strings"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
	"github.com/alecthomas/chroma/v3/lexers"

	"cogentcore.org/cogent/code/lsp"
)

// docsDialog shows the given documentation markdown from a language
// server (hover docs etc) in a new window with a read-only text editor,
// with syntax highlighted code blocks and clickable links, which open
// in the browser.
func docsDialog(ctx core.Widget, title, md string) {
	d := core.NewBody(title)
	ed := textcore.NewEditor(d)
	ed.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	buf := ed.Lines
	setDocsHighlighting(buf)
	buf.Settings.LineNumbers = false
	buf.AppendTextMarkup(docsMarkup(buf, md))
	buf.SetReadOnly(true)
	d.AddBottomBar(func(bar *core.Frame) {
		core.NewButton(bar).SetText("Copy to clipboard").SetIcon(icons.ContentCopy).
			OnClick(func(e events.Event) {
				d.Clipboard().Write(mimedata.NewText(lsp.MarkdownText(md)))
			})
		d.AddOK(bar)
	})
	d.RunWindowDialog(ctx)
}

// Window titles start with a standard kind name, followed by a colon,
// as the window geometry settings remember the window size and placement
// by the part of the title before the colon. These are the kind names
// for the language service document windows.
const (
	docsWindow        = "Documentation"
	signatureWindow   = "Signature"
	inlayHintsWindow  = "Inlay hints"
	assemblerWindow   = "Assembler"
	freeSymbolsWindow = "Free symbols"
	documentWindow    = "Document"
)

// documentTitle returns the window title for a page with the given title
// served by a language server at the given URL (see [LangService.showDocument]),
// starting with a standard kind name based on the gopls page type.
func documentTitle(uri, title string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return documentWindow + ": " + title
	}
	switch {
	case strings.Contains(u.Path, "/assembly"):
		if _, fn, ok := strings.Cut(title, " assembly for "); ok {
			title = fn
		}
		return assemblerWindow + ": " + title
	case strings.Contains(u.Path, "/pkg/"):
		_, pkg, _ := strings.Cut(u.Path, "/pkg/")
		return docsWindow + ": " + pkg
	case strings.Contains(u.Path, "/freesymbols"):
		if f := u.Query().Get("file"); f != "" {
			title = path.Base(f)
		}
		return freeSymbolsWindow + ": " + title
	}
	if title == "" {
		title = path.Base(u.Path)
	}
	return documentWindow + ": " + title
}

// setDocsHighlighting sets the highlighting style for docs in the buffer,
// from the settings, or the default if not set.
func setDocsHighlighting(buf *lines.Lines) {
	style := core.AppearanceSettings.Highlighting
	if style == "" {
		style = highlighting.StyleDefault
	}
	buf.SetHighlighting(style)
}

// docsMarkup returns the text lines and their markup for the given
// documentation markdown, for display in the given buffer.
func docsMarkup(buf *lines.Lines, md string) ([][]rune, []rich.Text) {
	sty := buf.FontStyle()
	bold := sty.Clone().SetWeight(rich.Bold)
	code := sty.Clone().SetBackground(colors.ToUniform(colors.Scheme.SurfaceContainerHigh))
	link := sty.Clone().SetLinkStyle()
	faint := sty.Clone().SetFillColor(colors.ToUniform(colors.Scheme.OutlineVariant))
	dls := lsp.ParseMarkdown(md)
	lns := make([][]rune, len(dls))
	mus := make([]rich.Text, len(dls))
	for i, dl := range dls {
		switch dl.Kind {
		case lsp.DocCode:
			lns[i] = []rune(dl.String())
			mus[i] = codeMarkup(buf, lns[i], dl.Lang)
		case lsp.DocRule:
			lns[i] = []rune(strings.Repeat("─", 40))
			mus[i] = rich.NewText(faint, lns[i])
		default:
			lns[i] = []rune(dl.String())
			txt := sty
			if dl.Kind == lsp.DocHeading {
				txt = bold
			}
			mu := rich.Text{}
			for _, sp := range dl.Spans {
				switch {
				case sp.URL != "":
					mu.AddLink(link, sp.URL, sp.Text)
				case sp.Code:
					mu.AddSpanString(code, sp.Text)
				default:
					mu.AddSpanString(txt, sp.Text)
				}
			}
			if len(mu) == 0 {
				mu = rich.NewText(sty, nil)
			}
			mus[i] = mu
		}
	}
	return lns, mus
}

// codeMarkup returns the syntax highlighting markup for the given line
// of code in the given language, using the highlighting style of the
// buffer, as for command output (see [MarkupCmdOutput]).
func codeMarkup(buf *lines.Lines, txt []rune, lang string) rich.Text {
	clex := lexers.Get(lang)
	if lang == "" || clex == nil {
		return rich.NewText(buf.FontStyle(), txt)
	}
	ctags, _ := highlighting.ChromaTagsLine(clex, string(txt))
	return highlighting.MarkupLineRich(buf.Highlighter.Style, buf.FontStyle(), txt, ctags, nil)
}
