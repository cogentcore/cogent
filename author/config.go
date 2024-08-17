// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package author

//go:generate core generate -add-types -add-funcs

type Formats int32 //enums:enum -transform lower

const (
	// PDF, via LaTeX, with full math support.
	PDF Formats = iota

	// HTML is a single standalone .html file.
	HTML

	// DOCX is a Microsoft Word compatible .docx file.
	DOCX

	// EPUB is a standard eBook .epub file.
	EPUB
)

// Config is the configuration information for the author cli.
type Config struct {

	// Output is the base name of the file or path to generate.
	// The appropriate extension will be added based on the output format.
	Output string `flag:"o,output"`

	// Formats are the list of formats for the generated output.
	Formats []Formats `default:"['pdf','html','docx','epub']" flag:"f,format"`
}
