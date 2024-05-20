// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorview"
)

// NewTabTable creates a tab with a table and a tableview.
// Use tv.Table.Table to get the underlying *table.Table
// and tv.Table is the table.IndexView onto the table.
// Use tv.Table.Sequential to update the IndexView to view
// all of the rows when done updating the Table.
func (br *Browser) NewTabTable(path string) *tensorview.TableView {
	tabs := br.Tabs()
	tab := tabs.RecycleTab(path, true)
	if tab.HasChildren() {
		tv := tab.Child(1).(*tensorview.TableView)
		return tv
	}
	dt := table.NewTable()
	tb := core.NewToolbar(tab)
	tv := tensorview.NewTableView(tab)
	tb.ConfigFuncs.Add(tv.ConfigToolbar)
	tv.SetTable(dt)
	return tv
}

// FormatTableFromCSV formats the columns of the given table according to the
// Name, Type values in given format CSV file.
func (br *Browser) FormatTableFromCSV(dt *table.Table, format string) error {
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

// OpenTOML opens given .toml formatted file with name = value
// entries, as a map.
func (br *Browser) OpenTOML(filename string) (map[string]string, error) {
	md := make(map[string]string)
	err := tomlx.Open(&md, filename)
	errors.Log(err)
	return md, err
}
