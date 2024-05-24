// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tensor/table"
)

// CleanCatTSV cleans a TSV file formed by
// concatenating multiple files together.
// Removes redundant headers and then sorts
// by given set of columns
func CleanCatTSV(filename string, sorts []string) error {
	str, err := os.ReadFile(filename)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	lns := strings.Split(string(str), "\n")
	if len(lns) == 0 {
		return nil
	}
	hdr := lns[0]
	f, err := os.Create(filename)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	for i, ln := range lns {
		if i > 0 && ln == hdr {
			continue
		}
		io.WriteString(f, ln)
		io.WriteString(f, "\n")
	}
	f.Close()
	dt := table.NewTable()
	err = dt.OpenCSV(core.Filename(filename), table.Detect)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	ix := table.NewIndexView(dt)
	ix.SortColumnNames(sorts, table.Ascending)
	st := ix.NewTable()
	err = st.SaveCSV(core.Filename(filename), table.Tab, true)
	if err != nil {
		slog.Error(err.Error())
	}
	return err
}
