// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package author

import (
	"runtime"

	"cogentcore.org/core/base/exec"
)

//go:generate core generate -add-types -add-funcs

type Formats int32 //enums:enum -transform lower

const (
	// HTML is a single standalone .html file.
	HTML Formats = iota

	// PDF, via LaTeX, with full math support.
	PDF

	// DOCX is a Microsoft Word compatible .docx file.
	DOCX

	// EPUB is a standard eBook .epub file.
	EPUB

	// LaTeX is a latex file, which can be further customized.
	LaTeX
)

// Config is the configuration information for the author cli.
type Config struct {

	// Output is the base name of the file or path to generate.
	// The appropriate extension will be added based on the output format.
	Output string `flag:"o,output"`

	// Formats are the list of formats for the generated output.
	Formats []Formats `default:"['html','pdf','docx','epub']" flag:"f,format"`
}

// Setup runs commands to install the necessary pandoc files using
// platform specific install commands.
func Setup(c *Config) error {
	switch runtime.GOOS {
	case "darwin":
		_, err := exec.Output("brew", "install", "pandoc", "pandoc-crossref")
		return err
	case "linux":
		_, err := exec.Output("sudo", "apt-get", "install", "-f", "-y", "pandoc", "pandoc-crossref")
		return err
	case "windows":
	}
	return nil
}
