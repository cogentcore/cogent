// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *KeyMaps) {
	if gi.ActivateExistingMainWindow(km) {
		return
	}
	sc := gi.NewScene("gide-key-maps")
	sc.Title = "Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map"
	sc.Lay = gi.LayoutVert
	sc.Data = km

	title := gi.NewLabel(sc, "title").SetText(sc.Title).SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Width.Ch(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := giv.NewTableView(sc).SetSlice(km)
	tv.SetStretchMax()

	AvailKeyMapsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailKeyMapsChanged = true
	})

	tb := tv.Toolbar()
	gi.NewSeparator(tb)
	sp := giv.NewFuncButton(tb, km.SavePrefs).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
	sp.SetUpdateFunc(func() {
		sp.SetEnabled(AvailKeyMapsChanged && km == &AvailKeyMaps)
	})
	oj := giv.NewFuncButton(tb, km.OpenJSON).SetText("Open from file").SetIcon(icons.Open).SetKey(keyfun.Open)
	oj.Args[0].SetTag("ext", ".json")
	sj := giv.NewFuncButton(tb, km.SaveJSON).SetText("Save to file").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
	sj.Args[0].SetTag("ext", ".json")
	gi.NewSeparator(tb)
	vs := giv.NewFuncButton(tb, km.ViewStd).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
	vs.SetUpdateFunc(func() {
		vs.SetEnabledUpdt(km != &StdKeyMaps)
	})
	rs := giv.NewFuncButton(tb, km.RevertToStd).SetConfirm(true).SetText("Revert to standard").SetIcon(icons.DeviceReset)
	rs.SetUpdateFunc(func() {
		rs.SetEnabledUpdt(km != &StdKeyMaps)
	})
	tb.OverflowMenu().SetMenu(func(m *gi.Scene) {
		giv.NewFuncButton(m, km.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
	})

	gi.NewWindow(sc).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  PrefsView

// PrefsView opens a view of user preferences, returns structview and window
func PrefsView(pf *Preferences) {
	if gi.ActivateExistingMainWindow(pf) {
		return
	}
	sc := gi.NewScene("gide-prefs")
	sc.Title = "Gide Preferences"
	sc.Lay = gi.LayoutVert
	sc.Data = pf

	title := gi.NewLabel(sc, "title").SetText(sc.Title).SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Width.Ch(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := giv.NewStructView(sc).SetStruct(pf)
	tv.SetStretchMax()

	tv.OnChange(func(e events.Event) {
		pf.Changed = true
	})

	tb := tv.Toolbar()
	gi.NewSeparator(tb)
	sp := giv.NewFuncButton(tb, pf.Save).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
	sp.SetUpdateFunc(func() {
		sp.SetEnabled(pf.Changed)
	})
	/*
		oj := giv.NewFuncButton(tb, pf.OpenJSON).SetText("Open from file").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := giv.NewFuncButton(tb, pf.SaveJSON).SetText("Save to file").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		gi.NewSeparator(tb)
		vs := giv.NewFuncButton(tb, pf.ViewStd).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
		vs.SetUpdateFunc(func() {
			vs.SetEnabledUpdt(pf != &StdKeyMaps)
		})
		rs := giv.NewFuncButton(tb, pf.RevertToStd).SetConfirm(true).SetText("Revert to standard").SetIcon(icons.DeviceReset)
		rs.SetUpdateFunc(func() {
			rs.SetEnabledUpdt(pf != &StdKeyMaps)
		})
		tb.OverflowMenu().SetMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pf.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	*/

	gi.NewWindow(sc).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  ProjPrefsView

// ProjPrefsView opens a view of project preferences (settings), returns structview and window
func ProjPrefsView(pf *ProjPrefs) { // (*giv.StructView, *gi.Window)
	if gi.ActivateExistingMainWindow(pf) {
		return
	}
	sc := gi.NewScene("gide-proj-prefs")
	sc.Title = "Gide Project Preferences"
	sc.Lay = gi.LayoutVert
	sc.Data = pf

	title := gi.NewLabel(sc, "title").SetText("Project preferences are saved in the project .gide file, along with other current state (open directories, splitter settings, etc) -- do Save Project to save.").SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Width.Ch(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := giv.NewStructView(sc).SetStruct(pf)
	tv.SetStretchMax()

	tv.OnChange(func(e events.Event) {
		pf.Changed = true
	})

	/*
		tb := tv.Toolbar()
		gi.NewSeparator(tb)
		sp := giv.NewFuncButton(tb, pf.Save).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(pf.Changed)
		})
			oj := giv.NewFuncButton(tb, pf.OpenJSON).SetText("Open from file").SetIcon(icons.Open).SetKey(keyfun.Open)
			oj.Args[0].SetTag("ext", ".json")
			sj := giv.NewFuncButton(tb, pf.SaveJSON).SetText("Save to file").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
			sj.Args[0].SetTag("ext", ".json")
			gi.NewSeparator(tb)
			vs := giv.NewFuncButton(tb, pf.ViewStd).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
			vs.SetUpdateFunc(func() {
				vs.SetEnabledUpdt(pf != &StdKeyMaps)
			})
			rs := giv.NewFuncButton(tb, pf.RevertToStd).SetConfirm(true).SetText("Revert to standard").SetIcon(icons.DeviceReset)
			rs.SetUpdateFunc(func() {
				rs.SetEnabledUpdt(pf != &StdKeyMaps)
			})
			tb.OverflowMenu().SetMenu(func(m *gi.Scene) {
				giv.NewFuncButton(m, pf.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
			})
	*/

	gi.NewWindow(sc).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  KeyMapValueView

// ValueView registers KeyMapValueView as the viewer of KeyMapName
func (kn KeyMapName) ValueView() giv.Value {
	return &KeyMapValueView{}
}

// KeyMapValueView presents an action for displaying an KeyMapName and selecting
// from KeyMapChooserDialog
type KeyMapValueView struct {
	giv.ValueBase
}

func (vv *KeyMapValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *KeyMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none -- click to set)"
	}
	bt.SetText(txt)
}

func (vv *KeyMapValueView) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *KeyMapValueView) HasDialog() bool {
	return true
}

func (vv *KeyMapValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailKeyMaps.MapByName(KeyMapName(cur))
	d := gi.NewDialog(ctx).Title("Select a key map").Prompt(vv.Doc()).FullWindow(true)
	giv.NewTableView(d).SetSlice(&AvailKeyMaps).SetSelIdx(curRow).BindSelectDialog(d, &si)
	d.OnAccept(func(e events.Event) {
		if si >= 0 {
			km := AvailKeyMaps[si]
			vv.SetValue(km.Name)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(d)
		}
	}).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  LangsView

// LangsView opens a view of a languages options map
func LangsView(pt *Langs) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	sc := gi.NewScene("gide-langs")
	sc.Title = "Available Language Opts: Add or modify entries to customize options for language / file types"
	sc.Lay = gi.LayoutVert
	sc.Data = pt

	title := gi.NewLabel(sc, "title").SetText(sc.Title).SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Width.Ch(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := giv.NewMapView(sc).SetMap(pt)
	tv.SetStretchMax()

	AvailLangsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailLangsChanged = true
	})

	tb := tv.Toolbar()
	gi.NewSeparator(tb)
	sp := giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
	sp.SetUpdateFunc(func() {
		sp.SetEnabled(AvailLangsChanged && pt == &AvailLangs)
	})
	oj := giv.NewFuncButton(tb, pt.OpenJSON).SetText("Open from file").SetIcon(icons.Open).SetKey(keyfun.Open)
	oj.Args[0].SetTag("ext", ".json")
	sj := giv.NewFuncButton(tb, pt.SaveJSON).SetText("Save to file").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
	sj.Args[0].SetTag("ext", ".json")
	gi.NewSeparator(tb)
	vs := giv.NewFuncButton(tb, pt.ViewStd).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
	vs.SetUpdateFunc(func() {
		vs.SetEnabledUpdt(pt != &StdLangs)
	})
	rs := giv.NewFuncButton(tb, pt.RevertToStd).SetConfirm(true).SetText("Revert to standard").SetIcon(icons.DeviceReset)
	rs.SetUpdateFunc(func() {
		rs.SetEnabledUpdt(pt != &StdLangs)
	})
	tb.OverflowMenu().SetMenu(func(m *gi.Scene) {
		giv.NewFuncButton(m, pt.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
	})

	gi.NewWindow(sc).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  CmdsView

// CmdsView opens a view of a commands table
func CmdsView(pt *Commands) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	sc := gi.NewScene("gide-cmds")
	sc.Title = "Gide Commands"
	sc.Lay = gi.LayoutVert
	sc.Data = pt

	title := gi.NewLabel(sc, "title").SetText(sc.Title).SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Width.Ch(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := giv.NewTableView(sc).SetSlice(pt)
	tv.SetStretchMax()

	CustomCmdsChanged = false
	tv.OnChange(func(e events.Event) {
		CustomCmdsChanged = true
	})

	/*
		tb := tv.Toolbar()
		gi.NewSeparator(tb)
		sp := giv.NewFuncButton(tb, km.SavePrefs).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(CustomCmdsChanged && km == &CustomCmds)
		})
		oj := giv.NewFuncButton(tb, km.OpenJSON).SetText("Open from file").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := giv.NewFuncButton(tb, km.SaveJSON).SetText("Save to file").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		gi.NewSeparator(tb)
		vs := giv.NewFuncButton(tb, km.ViewStd).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
		vs.SetUpdateFunc(func() {
			vs.SetEnabledUpdt(km != &StdKeyMaps)
		})
		tb.OverflowMenu().SetMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, km.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	*/

	gi.NewWindow(sc).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  CmdValueView

// ValueView registers CmdValueView as the viewer of CmdName
func (kn CmdName) ValueView() giv.Value {
	return &CmdValueView{}
}

// CmdValueView presents an action for displaying an CmdName and selecting
type CmdValueView struct {
	giv.ValueBase
}

func (vv *CmdValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *CmdValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	bt.SetText(txt)
}

func (vv *CmdValueView) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *CmdValueView) HasDialog() bool {
	return true
}

func (vv *CmdValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailCmds.CmdByName(CmdName(cur), false)
	d := gi.NewDialog(ctx).Title("Select a command").Prompt(vv.Doc()).FullWindow(true)
	giv.NewTableView(d).SetSlice(&AvailCmds).SetSelIdx(curRow).BindSelectDialog(d, &si)
	d.OnAccept(func(e events.Event) {
		if si >= 0 {
			pt := AvailCmds[si]
			vv.SetValue(CommandName(pt.Cat, pt.Name))
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(d)
		}
	}).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  SplitsView

// SplitsView opens a view of a splits table
func SplitsView(pt *Splits) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	sc := gi.NewScene("gide-splits")
	sc.Title = "Guide Splitters"
	sc.Lay = gi.LayoutVert
	sc.Data = pt

	title := gi.NewLabel(sc, "title").SetText("Available Splitter Settings: Can duplicate an existing (using Ctxt Menu) as starting point for new one").SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Width.Ch(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := giv.NewTableView(sc).SetSlice(pt)
	tv.SetStretchMax()

	AvailSplitsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailSplitsChanged = true
	})

	gi.NewWindow(sc).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  SplitValueView

// ValueView registers SplitValueView as the viewer of SplitName
func (kn SplitName) ValueView() giv.Value {
	return &SplitValueView{}
}

// SplitValueView presents an action for displaying an SplitName and selecting
type SplitValueView struct {
	giv.ValueBase
}

func (vv *SplitValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *SplitValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	bt.SetText(txt)
}

func (vv *SplitValueView) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *SplitValueView) HasDialog() bool {
	return true
}

func (vv *SplitValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	curRow := -1
	if cur != "" {
		_, curRow, _ = AvailSplits.SplitByName(SplitName(cur))
	}
	d := gi.NewDialog(ctx).Title("Select a Named Splitter Config").Prompt(vv.Doc()).FullWindow(true)
	giv.NewTableView(d).SetSlice(&AvailSplits).SetSelIdx(curRow).BindSelectDialog(d, &si)
	d.OnAccept(func(e events.Event) {
		if si >= 0 {
			pt := AvailSplits[si]
			vv.SetValue(pt.Name)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(d)
		}
	}).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  RegistersView

// RegistersView opens a view of a commands table
func RegistersView(pt *Registers) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	sc := gi.NewScene("gide-registers")
	sc.Title = "Guide Registers"
	sc.Lay = gi.LayoutVert
	sc.Data = pt

	title := gi.NewLabel(sc, "title").SetText("Available Registers: Can duplicate an existing (using Ctxt Menu) as starting point for new one").SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Width.Ch(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := giv.NewTableView(sc).SetSlice(pt)
	tv.SetStretchMax()

	AvailRegistersChanged = false
	tv.OnChange(func(e events.Event) {
		AvailRegistersChanged = true
	})

	gi.NewWindow(sc).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  RegisterValueView

// ValueView registers RegisterValueView as the viewer of RegisterName
func (kn RegisterName) ValueView() giv.Value {
	return &RegisterValueView{}
}

// RegisterValueView presents an action for displaying an RegisterName and selecting
type RegisterValueView struct {
	giv.ValueBase
}

func (vv *RegisterValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *RegisterValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	bt.SetText(txt)
}

func (vv *RegisterValueView) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *RegisterValueView) HasDialog() bool {
	return true
}

func (vv *RegisterValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	d := gi.NewDialog(ctx).Title("Select a command").Prompt(vv.Doc()).FullWindow(true)
	ch := gi.NewChooser(d).ItemsFromStringList(AvailRegisterNames, false, 30)
	ch.SetCurVal(cur)
	d.OnAccept(func(e events.Event) {
		rnm := ch.CurVal.(string)
		if ci := strings.Index(rnm, ":"); ci > 0 {
			rnm = rnm[:ci]
		}
		vv.SetValue(rnm)
		vv.UpdateWidget()
		if fun != nil {
			fun(d)
		}
	}).Run()
}
