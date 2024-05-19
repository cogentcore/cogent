// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"fmt"
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorview"
)

// NewTabTable creates a tab with a table and a tableview
// func (br *Browser) NewTabTable(path string) (*table.Table, *tensorview.TableView) {
func (br *Browser) NewTabTable(path string) *table.Table {
	tabs := br.Tabs()
	tab := tabs.NewTab(path)
	dt := table.NewTable()
	tb := core.NewToolbar(tab)
	tv := tensorview.NewTableView(tab)
	tb.ConfigFuncs.Add(tv.ConfigToolbar)
	tv.SetTable(dt)

	dpath := filepath.Join(br.DataRoot, path)
	fmt.Println("opening data at:", dpath)
	br.FormatTableFromCSV(dt, filepath.Join(dpath, "dbformat.csv"))
	dt.SetNumRows(10)
	br.Update()
	return dt
	// return dt, tv
}

// FormatTableFromCSV
func (br *Browser) FormatTableFromCSV(dt *table.Table, format string) error {
	fmt.Println("Formatting data table from CSV file:", format)
	ft := table.NewTable()
	if err := errors.Log(ft.OpenCSV(core.Filename(format), table.Comma)); err != nil {
		return err
	}
	for i := range ft.Rows {
		name := ft.StringValue("Name", i)
		typ := ft.StringValue("Type", i)
		switch typ {
		case "string":
			dt.AddStringColumn(name)
		case "time":
			dt.AddIntColumn(name)
		}
	}
	return nil
}
