// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLText(t *testing.T) {
	src := `<html><head><title>Doc: fmt</title><style>p {}</style><script>x()</script></head>
<body><h1>Package  fmt</h1>
<p>Call <code>fmt.<a href="#Println">Println</a></code> to   print &amp; more.</p>
<pre>	MOVD	$1, R0
	RET</pre>
<table><tr><td>a</td><td>b</td></tr></table></body></html>`
	title, text := htmlText(src)
	assert.Equal(t, "Doc: fmt", title)
	assert.Equal(t, "Package fmt\nCall fmt.Println to print & more.\n\tMOVD\t$1, R0\n\tRET\na\tb\n", text)
	assert.True(t, IsLocalURL("http://127.0.0.1:5000/gopls/x/pkg/fmt"))
	assert.False(t, IsLocalURL("https://github.com/golang/tools"))
}
