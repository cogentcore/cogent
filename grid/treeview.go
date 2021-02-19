// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"image/color"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// TreeView is a TreeView that knows how to operate on FileNode nodes
type TreeView struct {
	giv.TreeView
	GridView *GridView `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
}

var KiT_TreeView = kit.Types.AddType(&TreeView{}, nil)

// AddNewTreeView adds a new filetreeview to given parent node, with given name.
func AddNewTreeView(parent ki.Ki, name string) *TreeView {
	tv := parent.AddNewChild(KiT_TreeView, name).(*TreeView)
	// tv.SetFlag(int(giv.TreeViewFlagUpdtRoot))
	tv.OpenDepth = 1
	return tv
}

func init() {
	kit.Types.SetProps(KiT_TreeView, TreeViewProps)
}

// TreeViewIsLayerFunc is an ActionUpdateFunc that activates if node is a Layer
var TreeViewIsLayerFunc = giv.ActionUpdateFunc(func(fni interface{}, act *gi.Action) {
	tv := fni.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
	sn := tv.SrcNode
	if sn != nil {
		act.SetInactiveState(!NodeIsLayer(sn))
	}
})

// ParGridView returns the parent GridView
func (tv *TreeView) ParGridView() *GridView {
	rtv := tv.RootView.Embed(KiT_TreeView).(*TreeView)
	return rtv.GridView
}

// SelectSVG
func (tv *TreeView) SelectSVG() {
	gv := tv.ParGridView()
	if gv != nil {
		gv.SelectNodeInSVG(tv.SrcNode, mouse.SelectOne)
	}
}

// LayerIsCurrent returns true if layer is the current active one for creating
func (tv *TreeView) LayerIsCurrent() bool {
	gv := tv.ParGridView()
	if gv != nil {
		return gv.IsCurLayer(tv.SrcNode.Name())
	}
	return false
}

// LayerSetCurrent sets this layer as the current layer name
func (tv *TreeView) LayerSetCurrent() {
	sn := tv.SrcNode
	gv := tv.ParGridView()
	if gv != nil {
		cur := gv.EditState.CurLayer
		if cur != "" {
			cli := tv.Par.ChildByName("tv_"+cur, 0)
			if cli != nil {
				cl := cli.Embed(KiT_TreeView).(*TreeView)
				cl.LayerClearCurrent()
			}
		}
		if LayerIsLocked(sn) {
			tv.LayerToggleLock()
		}
		if !LayerIsVisible(sn) {
			tv.LayerToggleVis()
		}
		gv.SetCurLayer(sn.Name())
		tv.SetFullReRender() // needed for icon updating
		tv.UpdateSig()
	}
}

// LayerClearCurrent clears this layer as the current layer if it was set as such.
func (tv *TreeView) LayerClearCurrent() {
	gv := tv.ParGridView()
	if gv != nil {
		gv.ClearCurLayer(tv.SrcNode.Name())
		tv.SetFullReRender() // needed for icon updating
		tv.UpdateSig()
	}
}

// NodeIsLayer returns true if given node is a layer
func NodeIsLayer(kn ki.Ki) bool {
	gm := kit.ToString(kn.Prop("groupmode"))
	return gm == "layer"
}

// NodeIsMetaData returns true if given node is a MetaData
func NodeIsMetaData(kn ki.Ki) bool {
	_, ismd := kn.(*gi.MetaData2D)
	return ismd
}

// LayerIsLocked returns true if layer is locked (insensitive = true)
func LayerIsLocked(kn ki.Ki) bool {
	cp := kit.ToString(kn.Prop("insensitive"))
	return cp == "true"
}

// LayerIsVisible returns true if layer is visible
func LayerIsVisible(kn ki.Ki) bool {
	cp := kit.ToString(kn.Prop("style"))
	return cp != "display:none"
}

// LayerToggleLock toggles whether layer is locked or not
func (tv *TreeView) LayerToggleLock() {
	sn := tv.SrcNode
	np := ""
	if LayerIsLocked(sn) {
		np = "false"
	} else {
		tv.LayerClearCurrent()
		np = "true"
	}
	sn.SetProp("insensitive", np)
	tv.SetFullReRenderIconLabel()
	tv.UpdateSig()
}

// LayerToggleVis toggles visibility of the layer
func (tv *TreeView) LayerToggleVis() {
	sn := tv.SrcNode
	np := ""
	if LayerIsVisible(sn) {
		np = "display:none"
		tv.LayerClearCurrent()
	} else {
		np = "display:inline"
	}
	sn.SetProp("style", np)
	tv.UpdateSig()
}

var TreeViewProps = ki.Props{
	"EnumType:Flag":    giv.KiT_TreeViewFlags,
	"indent":           units.NewCh(2),
	"spacing":          units.NewCh(.5),
	"border-width":     units.NewPx(0),
	"border-radius":    units.NewPx(0),
	"padding":          units.NewPx(0),
	"margin":           units.NewPx(1),
	"text-align":       gist.AlignLeft,
	"vertical-align":   gist.AlignTop,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": "inherit",
	"no-templates":     true,
	".svgnode": ki.Props{
		"font-weight": gist.WeightNormal,
		"font-style":  gist.FontNormal,
	},
	".layer": ki.Props{
		"font-weight": gist.WeightBold,
	},
	".invisible": ki.Props{
		"font-style": gist.FontItalic,
	},
	".locked": ki.Props{
		"color": "#ff4252",
	},
	"#icon": ki.Props{
		"width":   units.NewEm(1),
		"height":  units.NewEm(1),
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
		"fill":    &gi.Prefs.Colors.Icon,
		"stroke":  &gi.Prefs.Colors.Font,
	},
	"#branch": ki.Props{
		"icon":             "wedge-down",
		"icon-off":         "wedge-right",
		"margin":           units.NewPx(0),
		"padding":          units.NewPx(0),
		"background-color": color.Transparent,
		"max-width":        units.NewEm(.8),
		"max-height":       units.NewEm(.8),
	},
	"#space": ki.Props{
		"width": units.NewEm(.5),
	},
	"#label": ki.Props{
		"margin":    units.NewPx(0),
		"padding":   units.NewPx(0),
		"min-width": units.NewCh(16),
	},
	"#menu": ki.Props{
		"indicator": "none",
	},
	giv.TreeViewSelectors[giv.TreeViewActive]: ki.Props{},
	giv.TreeViewSelectors[giv.TreeViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
	giv.TreeViewSelectors[giv.TreeViewFocus]: ki.Props{
		"background-color": &gi.Prefs.Colors.Control,
	},
	"CtxtMenuActive": ki.PropSlice{
		{"SrcEdit", ki.Props{
			"label": "Edit",
		}},
		{"SelectSVG", ki.Props{
			"label": "Select",
		}},
		{"sep-edit", ki.BlankProp{}},
		{"SrcDuplicate", ki.Props{
			"label":    "Duplicate",
			"shortcut": gi.KeyFunDuplicate,
		}},
		{"Copy", ki.Props{
			"shortcut": gi.KeyFunCopy,
			"Args": ki.PropSlice{
				{"reset", ki.Props{
					"value": true,
				}},
			},
		}},
		{"Cut", ki.Props{
			"shortcut": gi.KeyFunCut,
			"updtfunc": giv.ActionUpdateFunc(func(tvi interface{}, act *gi.Action) {
				tv := tvi.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
				act.SetInactiveState(tv.IsRootOrField(""))
			}),
		}},
		{"Paste", ki.Props{
			"shortcut": gi.KeyFunPaste,
		}},
		{"sep-layer", ki.BlankProp{}},
		{"LayerSetCurrent", ki.Props{
			"label":    "Layer: Set Current",
			"updtfunc": TreeViewIsLayerFunc,
		}},
		{"LayerToggleLock", ki.Props{
			"label":    "Layer: Toggle Lock",
			"updtfunc": TreeViewIsLayerFunc,
		}},
		{"LayerToggleVis", ki.Props{
			"label":    "Layer: Toggle Visible",
			"updtfunc": TreeViewIsLayerFunc,
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenAll", ki.Props{}},
		{"CloseAll", ki.Props{}},
	},
}

func (tv *TreeView) Style2D() {
	sn := tv.SrcNode
	tv.Class = ""
	if sn != nil {
		if NodeIsLayer(sn) {
			tv.Icon = gi.IconName("blank")
			tv.AddClass("layer")
			if tv.LayerIsCurrent() {
				tv.Icon = gi.IconName("checkmark")
			}
			if LayerIsLocked(sn) {
				tv.AddClass("locked")
				tv.Icon = gi.IconName("close")
			}
			if !LayerIsVisible(sn) {
				tv.AddClass("invisible")
				tv.Icon = gi.IconName("close")
			}
			// todo: visibility and locked flags
		} else {
			tv.AddClass("svgnode")
			switch sn.(type) {
			case *svg.Circle:
				tv.Icon = gi.IconName("circlebutton-off")
			case *svg.Ellipse:
				tv.Icon = gi.IconName("circlebutton-off")
			case *svg.Rect:
				tv.Icon = gi.IconName("stop")
			case *svg.Path:
				tv.Icon = gi.IconName("color")
			case *svg.Image:
				tv.Icon = gi.IconName("file-image") // todo: image
			case *svg.Text:
				tv.Icon = gi.IconName("file-doc") // todo: A = text
			}
		}
		tv.StyleTreeView()
		tv.LayState.SetFromStyle(&tv.Sty.Layout) // also does reset
	}
}
