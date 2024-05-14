// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"fmt"
	"path/filepath"
)

// OpenDataTab opens a tab with a table displaying the
func (br *Browser) OpenDataTab(path string) {
	tabs := br.Tabs()
	dt := tabs.NewTab(path)
	_ = dt
	// todo: read table
	dpath := filepath.Join(br.DataRoot, path)
	fmt.Println("opening data at:", dpath)
}
