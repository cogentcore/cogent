// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"runtime"
	"strings"

	"cogentcore.org/core/htmlcore"
)

// defineNavigator defines the values of navigator that come from Go:
// the user agent and languages of the requests of the page (sites check
// that they match), and the platform.
func (e *Engine) defineNavigator() {
	e.jsDOM.Set("userAgent", htmlcore.UserAgent)
	e.jsDOM.Set("platform", navigatorPlatform())
	e.jsDOM.Set("languages", navigatorLanguages(htmlcore.AcceptLanguage))
}

// navigatorPlatform returns navigator.platform for the platform that the
// app runs on, as browsers report it.
func navigatorPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "Win32"
	case "linux", "android":
		return "Linux x86_64"
	case "ios":
		return "iPhone"
	default:
		return "MacIntel"
	}
}

// navigatorLanguages returns navigator.languages for the given
// Accept-Language header: its languages, without their weights.
func navigatorLanguages(header string) []any {
	var langs []any
	for _, l := range strings.Split(header, ",") {
		l, _, _ = strings.Cut(l, ";")
		if l = strings.TrimSpace(l); l != "" {
			langs = append(langs, l)
		}
	}
	if len(langs) == 0 {
		langs = []any{"en-US"}
	}
	return langs
}
