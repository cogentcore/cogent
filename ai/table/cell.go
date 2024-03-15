/*
 * Copyright ©1998-2023 by Richard A. Wilkes. All rights reserved.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, version 2.0. If a copy of the MPL was not distributed with
 * this file, You can obtain one at http://mozilla.org/MPL/2.0/.
 *
 * This Source Code Form is "Incompatible With Secondary Licenses", as
 * defined by the Mozilla Public License, version 2.0.
 */

package table

// PageRefCellAlias is used an alias to request the page reference cell, if any.
const PageRefCellAlias = -10

// CellData holds data for creating a cell's visual representation.
type CellData struct {
	Type     Type
	Disabled bool
	Dim      bool
	Checked  bool
	//Alignment         Enum
	Primary           string
	Secondary         string
	Tooltip           string
	UnsatisfiedReason string
	TemplateInfo      string
	InlineTag         string
}

// ForSort returns a string that can be used to sort or search against for this data.
func (c *CellData) ForSort() string {
	switch c.Type {
	case Text:
		if c.Secondary != "" {
			return c.Primary + "\n" + c.Secondary
		}
		return c.Primary
	case Toggle:
		if c.Checked {
			return "√"
		}
	case PageRef, Tags, Markdown:
		return c.Primary
	}
	return ""
}
