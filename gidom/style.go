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
