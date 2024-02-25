// Code generated by "core generate"; DO NOT EDIT.

package vector

import (
	"cogentcore.org/core/giv"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/units"
)

// AlignViewType is the [gti.Type] for [AlignView]
var AlignViewType = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/vector.AlignView", IDName: "align-view", Doc: "AlignView provides a range of alignment actions on selected objects.", Embeds: []gti.Field{{Name: "Layout"}}, Fields: []gti.Field{{Name: "AlignAnchorView"}, {Name: "VectorView", Doc: "the parent vectorview"}}, Instance: &AlignView{}})

// NewAlignView adds a new [AlignView] with the given name to the given parent:
// AlignView provides a range of alignment actions on selected objects.
func NewAlignView(par ki.Ki, name ...string) *AlignView {
	return par.NewChild(AlignViewType, name...).(*AlignView)
}

// KiType returns the [*gti.Type] of [AlignView]
func (t *AlignView) KiType() *gti.Type { return AlignViewType }

// New returns a new [*AlignView] value
func (t *AlignView) New() ki.Ki { return &AlignView{} }

// SetAlignAnchorView sets the [AlignView.AlignAnchorView]
func (t *AlignView) SetAlignAnchorView(v giv.EnumValue) *AlignView { t.AlignAnchorView = v; return t }

// SetVectorView sets the [AlignView.VectorView]:
// the parent vectorview
func (t *AlignView) SetVectorView(v *VectorView) *AlignView { t.VectorView = v; return t }

// SetTooltip sets the [AlignView.Tooltip]
func (t *AlignView) SetTooltip(v string) *AlignView { t.Tooltip = v; return t }

// SetStackTop sets the [AlignView.StackTop]
func (t *AlignView) SetStackTop(v int) *AlignView { t.StackTop = v; return t }

// PaintViewType is the [gti.Type] for [PaintView]
var PaintViewType = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/vector.PaintView", IDName: "paint-view", Doc: "PaintView provides editing of basic Stroke and Fill painting parameters\nfor selected items", Embeds: []gti.Field{{Name: "Layout"}}, Fields: []gti.Field{{Name: "StrokeType", Doc: "paint type for stroke"}, {Name: "StrokeStops", Doc: "name of gradient with stops"}, {Name: "FillType", Doc: "paint type for fill"}, {Name: "FillStops", Doc: "name of gradient with stops"}, {Name: "VectorView", Doc: "the parent vectorview"}}, Instance: &PaintView{}})

// NewPaintView adds a new [PaintView] with the given name to the given parent:
// PaintView provides editing of basic Stroke and Fill painting parameters
// for selected items
func NewPaintView(par ki.Ki, name ...string) *PaintView {
	return par.NewChild(PaintViewType, name...).(*PaintView)
}

// KiType returns the [*gti.Type] of [PaintView]
func (t *PaintView) KiType() *gti.Type { return PaintViewType }

// New returns a new [*PaintView] value
func (t *PaintView) New() ki.Ki { return &PaintView{} }

// SetStrokeType sets the [PaintView.StrokeType]:
// paint type for stroke
func (t *PaintView) SetStrokeType(v PaintTypes) *PaintView { t.StrokeType = v; return t }

// SetStrokeStops sets the [PaintView.StrokeStops]:
// name of gradient with stops
func (t *PaintView) SetStrokeStops(v string) *PaintView { t.StrokeStops = v; return t }

// SetFillType sets the [PaintView.FillType]:
// paint type for fill
func (t *PaintView) SetFillType(v PaintTypes) *PaintView { t.FillType = v; return t }

// SetFillStops sets the [PaintView.FillStops]:
// name of gradient with stops
func (t *PaintView) SetFillStops(v string) *PaintView { t.FillStops = v; return t }

// SetVectorView sets the [PaintView.VectorView]:
// the parent vectorview
func (t *PaintView) SetVectorView(v *VectorView) *PaintView { t.VectorView = v; return t }

// SetTooltip sets the [PaintView.Tooltip]
func (t *PaintView) SetTooltip(v string) *PaintView { t.Tooltip = v; return t }

// SetStackTop sets the [PaintView.StackTop]
func (t *PaintView) SetStackTop(v int) *PaintView { t.StackTop = v; return t }

var _ = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/vector.Preferences", IDName: "preferences", Doc: "Preferences is the overall Vector preferences", Directives: []gti.Directive{{Tool: "gti", Directive: "add"}}, Fields: []gti.Field{{Name: "Size", Doc: "default physical size, when app is started without opening a file"}, {Name: "Colors", Doc: "active color preferences"}, {Name: "ColorSchemes", Doc: "named color schemes -- has Light and Dark schemes by default"}, {Name: "ShapeStyle", Doc: "default shape styles"}, {Name: "TextStyle", Doc: "default text styles"}, {Name: "PathStyle", Doc: "default line styles"}, {Name: "LineStyle", Doc: "default line styles"}, {Name: "VectorDisp", Doc: "turns on the grid display"}, {Name: "SnapVector", Doc: "snap positions and sizes to underlying grid"}, {Name: "SnapGuide", Doc: "snap positions and sizes to line up with other elements"}, {Name: "SnapNodes", Doc: "snap node movements to align with guides"}, {Name: "SnapTol", Doc: "number of screen pixels around target point (in either direction) to snap"}, {Name: "SplitName", Doc: "named-split config in use for configuring the splitters"}, {Name: "EnvVars", Doc: "environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app"}, {Name: "Changed", Doc: "flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."}}})

var _ = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/vector.ColorPrefs", IDName: "color-prefs", Doc: "ColorPrefs for", Directives: []gti.Directive{{Tool: "gti", Directive: "add"}}, Fields: []gti.Field{{Name: "Background", Doc: "drawing background color"}, {Name: "Border", Doc: "border color of the drawing"}, {Name: "Vector", Doc: "grid line color"}}})

// SVGViewType is the [gti.Type] for [SVGView]
var SVGViewType = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/vector.SVGView", IDName: "svg-view", Doc: "SVGView is the element for viewing, interacting with the SVG", Embeds: []gti.Field{{Name: "SVG"}}, Fields: []gti.Field{{Name: "VectorView", Doc: "the parent vectorview"}, {Name: "Trans", Doc: "view translation offset (from dragging)"}, {Name: "Scale", Doc: "view scaling (from zooming)"}, {Name: "Vector", Doc: "grid spacing, in native ViewBox units"}, {Name: "VectorEff", Doc: "effective grid spacing given Scale level"}, {Name: "SetDragCursor", Doc: "has dragging cursor been set yet?"}, {Name: "BgPixels", Doc: "background pixels, includes page outline and grid"}, {Name: "bgTrans", Doc: "bg rendered translation"}, {Name: "bgScale", Doc: "bg rendered scale"}, {Name: "bgVectorEff", Doc: "bg rendered grid"}}, Instance: &SVGView{}})

// NewSVGView adds a new [SVGView] with the given name to the given parent:
// SVGView is the element for viewing, interacting with the SVG
func NewSVGView(par ki.Ki, name ...string) *SVGView {
	return par.NewChild(SVGViewType, name...).(*SVGView)
}

// KiType returns the [*gti.Type] of [SVGView]
func (t *SVGView) KiType() *gti.Type { return SVGViewType }

// New returns a new [*SVGView] value
func (t *SVGView) New() ki.Ki { return &SVGView{} }

// SetTooltip sets the [SVGView.Tooltip]
func (t *SVGView) SetTooltip(v string) *SVGView { t.Tooltip = v; return t }

// TreeViewType is the [gti.Type] for [TreeView]
var TreeViewType = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/vector.TreeView", IDName: "tree-view", Doc: "TreeView is a TreeView that knows how to operate on FileNode nodes", Embeds: []gti.Field{{Name: "TreeView"}}, Fields: []gti.Field{{Name: "VectorView", Doc: "the parent vectorview"}}, Instance: &TreeView{}})

// NewTreeView adds a new [TreeView] with the given name to the given parent:
// TreeView is a TreeView that knows how to operate on FileNode nodes
func NewTreeView(par ki.Ki, name ...string) *TreeView {
	return par.NewChild(TreeViewType, name...).(*TreeView)
}

// KiType returns the [*gti.Type] of [TreeView]
func (t *TreeView) KiType() *gti.Type { return TreeViewType }

// New returns a new [*TreeView] value
func (t *TreeView) New() ki.Ki { return &TreeView{} }

// SetVectorView sets the [TreeView.VectorView]:
// the parent vectorview
func (t *TreeView) SetVectorView(v *VectorView) *TreeView { t.VectorView = v; return t }

// SetTooltip sets the [TreeView.Tooltip]
func (t *TreeView) SetTooltip(v string) *TreeView { t.Tooltip = v; return t }

// SetText sets the [TreeView.Text]
func (t *TreeView) SetText(v string) *TreeView { t.Text = v; return t }

// SetIcon sets the [TreeView.Icon]
func (t *TreeView) SetIcon(v icons.Icon) *TreeView { t.Icon = v; return t }

// SetIconOpen sets the [TreeView.IconOpen]
func (t *TreeView) SetIconOpen(v icons.Icon) *TreeView { t.IconOpen = v; return t }

// SetIconClosed sets the [TreeView.IconClosed]
func (t *TreeView) SetIconClosed(v icons.Icon) *TreeView { t.IconClosed = v; return t }

// SetIconLeaf sets the [TreeView.IconLeaf]
func (t *TreeView) SetIconLeaf(v icons.Icon) *TreeView { t.IconLeaf = v; return t }

// SetIndent sets the [TreeView.Indent]
func (t *TreeView) SetIndent(v units.Value) *TreeView { t.Indent = v; return t }

// SetOpenDepth sets the [TreeView.OpenDepth]
func (t *TreeView) SetOpenDepth(v int) *TreeView { t.OpenDepth = v; return t }

// SetViewIdx sets the [TreeView.ViewIdx]
func (t *TreeView) SetViewIdx(v int) *TreeView { t.ViewIdx = v; return t }

// SetWidgetSize sets the [TreeView.WidgetSize]
func (t *TreeView) SetWidgetSize(v mat32.Vec2) *TreeView { t.WidgetSize = v; return t }

// SetRootView sets the [TreeView.RootView]
func (t *TreeView) SetRootView(v *giv.TreeView) *TreeView { t.RootView = v; return t }

// SetSelectedNodes sets the [TreeView.SelectedNodes]
func (t *TreeView) SetSelectedNodes(v ...giv.TreeViewer) *TreeView { t.SelectedNodes = v; return t }

// VectorViewType is the [gti.Type] for [VectorView]
var VectorViewType = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/vector.VectorView", IDName: "vector-view", Doc: "VectorView is the Vector SVG vector drawing program", Embeds: []gti.Field{{Name: "Frame"}}, Fields: []gti.Field{{Name: "Filename", Doc: "full path to current drawing filename"}, {Name: "EditState", Doc: "current edit state"}}, Instance: &VectorView{}})

// NewVectorView adds a new [VectorView] with the given name to the given parent:
// VectorView is the Vector SVG vector drawing program
func NewVectorView(par ki.Ki, name ...string) *VectorView {
	return par.NewChild(VectorViewType, name...).(*VectorView)
}

// KiType returns the [*gti.Type] of [VectorView]
func (t *VectorView) KiType() *gti.Type { return VectorViewType }

// New returns a new [*VectorView] value
func (t *VectorView) New() ki.Ki { return &VectorView{} }

// SetTooltip sets the [VectorView.Tooltip]
func (t *VectorView) SetTooltip(v string) *VectorView { t.Tooltip = v; return t }

// SetStackTop sets the [VectorView.StackTop]
func (t *VectorView) SetStackTop(v int) *VectorView { t.StackTop = v; return t }