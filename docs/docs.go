// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command docs provides documentation of the Cogent apps,
// hosted at https://cogentcore.org/cogent.
package main

import "cogentcore.org/core/core"

func main() {
	b := core.NewBody("Cogent Apps").AddTitle("Cogent Apps")
	b.RunMainWindow()
}
