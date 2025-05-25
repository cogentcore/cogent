// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command docs provides documentation of the Cogent apps,
// hosted at https://cogentcore.org/cogent.
package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
)

func main() {
	b := core.NewBody("Cogent Apps")
	htmlcore.ReadMDString(htmlcore.NewContext(), b, `# Cogent Apps
* [Cogent Code](https://cogentcore.org/cogent/code)
* [Cogent Canvas](https://cogentcore.org/cogent/canvas)
* [Cogent Chess](https://cogentcore.org/cogent/chess)
* [Cogent Marbles](https://cogentcore.org/cogent/marbles)

See other Cogent Apps at https://github.com/cogentcore/cogent`)
	b.RunMainWindow()
}
