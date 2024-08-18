// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command author processes markdown input files into a range
// of different standard output formats.
package main

import (
	"cogentcore.org/cogent/author"
	"cogentcore.org/cogent/author/book"
	"cogentcore.org/core/cli"
	"cogentcore.org/core/cli/clicore"
)

func main() {
	opts := cli.DefaultOptions("Cogent Author", "Generates standard output formats from markdown input files, automatically handling references, figures and other advanced features.")
	opts.DefaultFiles = []string{"author.toml"}
	clicore.Run(opts, &author.Config{}, book.Book, author.Setup)
}
