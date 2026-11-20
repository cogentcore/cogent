// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command site is the website of the Cogent apps, cogentcore.org/cogent,
// built with the content package: its home page has a grid of the apps,
// each linking to the app on the web and to its documentation, which
// is on the website too, at docs/<app> (the same documentation that the
// Help of the app menu opens in the app).
package main

import (
	"embed"
	"io/fs"

	codedocs "cogentcore.org/cogent/code/docs"
	filesdocs "cogentcore.org/cogent/files/docs"
	listsdocs "cogentcore.org/cogent/lists/docs"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
)

//go:embed content
var contentFS embed.FS

//go:embed icons
var iconsFS embed.FS

// webURL is the URL of the website, under which each app is on the web
// at its Path.
const webURL = "https://cogentcore.org/cogent/"

// App is a Cogent app on the website.
type App struct {

	// Name is the name of the app, without "Cogent".
	Name string

	// Path is the path of the app on the website, under [webURL], and of its
	// documentation, under docs/.
	Path string

	// About is a short description of the app.
	About string

	// Docs is the documentation of the app, whose home page is home.md and
	// which has its icon.svg, or nil if it has none yet; the icon is then
	// <Path>.svg in the icons directory of the site.
	Docs fs.FS
}

// Apps are the apps on the website, in order.
var Apps = []*App{
	{Name: "Code", Path: "code", About: "A code editor", Docs: codedocs.FS},
	{Name: "Canvas", Path: "canvas", About: "A vector graphics editor"},
	{Name: "Lists", Path: "lists", About: "All of your lists", Docs: listsdocs.FS},
	{Name: "Files", Path: "files", About: "A file explorer", Docs: filesdocs.FS},
	{Name: "Chess", Path: "chess", About: "Play chess"},
	{Name: "Marbles", Path: "marbles", About: "Marbles bouncing on graphs of equations"},
}

// URL returns the URL of the app on the web.
func (a *App) URL() string { return webURL + a.Path }

// DocsURL returns the URL of the documentation of the app on the website,
// relative to its root, or "" if it has none.
func (a *App) DocsURL() string {
	if a.Docs == nil {
		return ""
	}
	return "docs/" + a.Path
}

// Icon returns the SVG icon of the app.
func (a *App) Icon() string {
	var b []byte
	var err error
	if a.Docs != nil {
		b, err = fs.ReadFile(a.Docs, "icon.svg")
	} else {
		b, err = fs.ReadFile(iconsFS, "icons/"+a.Path+".svg")
	}
	errors.Log(err)
	return string(b)
}

// Source returns the source of the content of the website: its own pages,
// with the documentation of each app mounted at its [App.DocsURL].
func Source() fs.FS {
	src := content.NewMountFS(fsx.Sub(contentFS, "content"))
	for _, a := range Apps {
		if a.Docs != nil {
			src.Mount(a.DocsURL(), a.Docs)
		}
	}
	return src
}

// NewSite makes the website in the given body, and returns its content.
func NewSite(b *core.Body) *content.Content {
	ct := content.NewContent(b)
	ct.Context.ElementHandlers["cogent-apps"] = func(ctx *htmlcore.Context) bool {
		appsGrid(ctx)
		return true
	}
	ct.SetSource(Source())
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(ct.MakeToolbar)
		tb.Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				ct.Context.LinkButton(w, "https://github.com/cogentcore/cogent")
				w.SetText("GitHub").SetIcon(icons.GitHub)
			})
			tree.Add(p, func(w *core.Button) {
				ct.Context.LinkButton(w, "https://cogentcore.org/core")
				w.SetText("Cogent Core").SetIcon(icons.Home)
			})
		})
	})
	return ct
}

// appsGrid makes the grid of the [Apps], for the <cogent-apps> element of
// the home page: for each app, its icon and name, which link to the app on
// the web, its description, and a link to its documentation, if it has any.
func appsGrid(ctx *htmlcore.Context) {
	grid := core.NewFrame(ctx.BlockParent)
	grid.Styler(func(s *styles.Style) {
		s.Wrap = true
		s.Grow.Set(1, 0)
		s.Justify.Content = styles.Center
		s.Gap.Set(units.Em(1))
		s.Margin.Set(units.Em(1), units.Zero())
	})
	for _, a := range Apps {
		tile := core.NewFrame(grid)
		tile.SetName(a.Path)
		tile.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Align.Items = styles.Center
			s.Min.X.Em(14) // all the same width
			s.Max.X.Em(14)
			s.Padding.Set(units.Em(1))
			s.Gap.Set(units.Em(0.25))
			s.Border.Radius = styles.BorderRadiusLarge
			s.Background = colors.Scheme.SurfaceContainerLow
		})
		icon := core.NewButton(tile).SetType(core.ButtonAction).SetIcon(icons.Icon(a.Icon()))
		ctx.LinkButton(icon, a.URL())
		icon.SetTooltip("Run Cogent " + a.Name + " on the web")
		icon.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Dp(96))
		})
		name := core.NewButton(tile).SetType(core.ButtonText).SetText("Cogent " + a.Name)
		ctx.LinkButton(name, a.URL())
		name.Styler(func(s *styles.Style) {
			s.Font.Size = units.Dp(20)
			s.Font.Weight = rich.Semibold
		})
		core.NewText(tile).SetType(core.TextBodyMedium).SetText(a.About).Styler(func(s *styles.Style) {
			s.Color = colors.Scheme.OnSurfaceVariant
			s.Text.Align = text.Center
		})
		if du := a.DocsURL(); du != "" {
			docs := core.NewButton(tile).SetType(core.ButtonTonal).SetText("Docs").SetIcon(icons.Book)
			ctx.LinkButton(docs, du)
			docs.SetTooltip("The documentation of Cogent " + a.Name)
		}
	}
}

func main() {
	b := core.NewBody("Cogent Apps")
	content.OfflineURL = "https://cogentcore.org/cogent"
	NewSite(b)
	b.RunMainWindow()
}
