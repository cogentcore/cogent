// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/highlighting"
	"github.com/stretchr/testify/assert"
)

func TestParserLanguages(t *testing.T) {
	// standard: parser for markdown and tex only
	assert.NotNil(t, highlighting.ParseLanguage(fileinfo.Markdown))
	assert.NotNil(t, highlighting.ParseLanguage(fileinfo.TeX))
	assert.Nil(t, highlighting.ParseLanguage(fileinfo.Go))

	// parser backend: all parse languages
	parserBackend.Store(true)
	assert.NotNil(t, highlighting.ParseLanguage(fileinfo.Go))
	parserBackend.Store(false)

	// per-language option
	AvailableLanguages[fileinfo.Go].Parser = true
	updateParserLangs()
	assert.NotNil(t, highlighting.ParseLanguage(fileinfo.Go))
	AvailableLanguages[fileinfo.Go].Parser = false
	updateParserLangs()
	assert.Nil(t, highlighting.ParseLanguage(fileinfo.Go))
}
