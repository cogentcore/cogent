// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

// // ApplyStyle applies styling information to the given parent widget,
// // using the given context. This should only be called in [ReadHTMLNode]
// // after the widget has already been populated by the node tree.
// func ApplyStyle(ctx Context, par gi.Widget) error {
// 	src := ctx.GetStyle()

// 	ss, err := parser.Parse(src)
// 	if err != nil {
// 		return err
// 	}
// 	selcss.Parse()
// 	par.AsWidget().WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
// 		for _, rule := range ss.Rules {
// 			if MatchesRule(wb, rule) {
// 				fmt.Println(wb, "\nMATCHES\n", rule)
// 			}
// 		}
// 		return true
// 	})
// 	return nil
// }

// // MatchesRule returns whether the given widget matches any of the selectors of
// // the given [css.Rule].
// func MatchesRule(w *gi.WidgetBase, rule *css.Rule) bool {
// 	for _, sel := range rule.Selectors {
// 		if MatchesSelector(w, sel) {
// 			return true
// 		}
// 	}
// 	return false
// }

// // MatchesSelector returns whether the given widget matches the given CSS selector.
// func MatchesSelector(w *gi.WidgetBase, sel string) bool {
// 	var fields []string
// 	for {
// 		i := strings.IndexFunc(sel, func(r rune) bool {
// 			return r == ' ' || r == '#' || r == '.'
// 		})
// 		if i == -1 {
// 			if len(sel) > 0 {
// 				fields = append(fields, sel)
// 			}
// 			break
// 		}
// 		fields = append(fields, strings.TrimSpace(sel[:i+1]))
// 		sel = strings.TrimSpace(sel[i+1:])
// 	}

// 	for _, field := range fields {
// 		fmt.Println(field)
// 		switch {
// 		case strings.HasPrefix(field, "#"):
// 			if w.Name() != strings.TrimPrefix(field, "#") {
// 				return false
// 			}
// 		case strings.HasPrefix(field, "."):
// 			if !w.HasClass(strings.TrimPrefix(field, ".")) {
// 				return false
// 			}
// 		default:
// 			if w.Prop("tag") != field {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }
