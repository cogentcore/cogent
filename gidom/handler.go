// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/ki/v2"
	"golang.org/x/net/html"
)

// Handler is a function that can be used to describe the behavior
// of gidom parsing for a specific type of element. It takes the
// parent [ki.Ki] to add widgets to and the [*html.Node] to read from.
// If it returns nil, that indicates that the children of this node have
// already been handled (or will not be handled), and thus should not
// be handled further. If it returns a non-nil widget, then the children
// will be handled, with the returned widget as their parent.
type Handler func(par ki.Ki, n *html.Node) gi.Widget

// ElementHandlers is a map of [Handler] functions for each HTML element
// type (eg: "button", "input", "p"). It is initialized to contain appropriate
// handlers for all of the standard HTML elements, but can be extended or
// modified for anyone in need of different behavior.
var ElementHandlers = map[string]Handler{}
