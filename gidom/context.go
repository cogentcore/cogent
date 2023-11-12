// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

// Context contains context information needed for gidom calls.
type Context interface {
	// PageURL returns the URL of the current page, and "" if there
	// is no current page.
	PageURL() string
}
