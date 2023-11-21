// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	_ "embed"
)

// UserAgentStyles contains the default user agent styles, as defined
// at https://chromium.googlesource.com/chromium/blink/+/refs/heads/main/Source/core/css/html.css.
//
//go:embed html.css
var UserAgentStyles string

// // ApplyStyle applies styling information to the given parent widget,
// // using the given context. This should only be called in [ReadHTMLNode]
// // after the widget has already been populated by the node tree.
// func ApplyStyle(ctx Context, par gi.Widget, n *html.Node) error {
// 	ss, sels := ctx.GetStyle()
// 	for i, r := range ss.Rules {
// 		sel := sels[i]
// 		matches := sel.Select(n)
// 		for _, match := range matches {
// 			w := ctx.WidgetForNode(match)
// 			// TODO(kai/styprops): need to go into text pseudo elements to stop these errors
// 			if w == nil {
// 				slog.Error("did not find widget for node", "type", match.Data, "id", GetAttr(match, "id"))
// 				continue
// 			}
// 			// fmt.Println("STYLE", w, ":\n", r)
// 			w.Style(func(s *styles.Style) {
// 				for _, decl := range r.Declarations {
// 					// TODO(kai/styprops): parent style and context
// 					s.StyleFromProp(s, decl.Property, decl.Value, colors.BaseContext(s.Color))
// 				}
// 			})
// 		}
// 	}
// 	return nil
// }
