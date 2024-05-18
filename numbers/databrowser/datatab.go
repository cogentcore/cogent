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

// OpenDataTab opens a tab with a table displaying the
func (br *Browser) OpenDataTab(path string) {
	tabs := br.Tabs()
	dt := tabs.NewTab(path)
	_ = dt
	// todo: read table
	dpath := errors.Log1(filepath.Abs(path))
	fmt.Println("opening data at:", dpath)
	tab := table.NewTable()
	format := filepath.Join(dpath, "dbformat.csv")
	br.FormatTableFromCSV(tab, format)
	tab.SetNumRows(10)
	tv := tensorview.NewTableView(dt)
	tv.SetTable(tab)
}

// FormatTableFromCSV
func (br *Browser) FormatTableFromCSV(dt *table.Table, format string) error {
	ft := table.NewTable()
	if err := errors.Log(ft.OpenCSV(core.Filename(format), table.Comma)); err != nil {
		return err
	}
	fmt.Println("rows:", ft.Rows, ft.ColumnNames)
	for i := range ft.Rows {
		name := ft.StringValue("Name", i)
		typ := ft.StringValue("Type", i)
		fmt.Println(name, typ)
		switch typ {
		case "string":
			dt.AddStringColumn(name)
		case "time":
			dt.AddIntColumn(name)
		}
	}
	return nil
}
