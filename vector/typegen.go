// Code generated by "core generate"; DO NOT EDIT.

package vector

import (
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// AlignViewType is the [types.Type] for [AlignView]
var AlignViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/vector.AlignView", IDName: "align-view", Doc: "AlignView provides a range of alignment actions on selected objects.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Anchor", Doc: "Anchor is the alignment anchor"}, {Name: "Vector", Doc: "the parent vector"}}, Instance: &AlignView{}})

// NewAlignView returns a new [AlignView] with the given optional parent:
// AlignView provides a range of alignment actions on selected objects.
func NewAlignView(parent ...tree.Node) *AlignView { return tree.New[*AlignView](parent...) }

// NodeType returns the [*types.Type] of [AlignView]
func (t *AlignView) NodeType() *types.Type { return AlignViewType }

// New returns a new [*AlignView] value
func (t *AlignView) New() tree.Node { return &AlignView{} }

// SetAnchor sets the [AlignView.Anchor]:
// Anchor is the alignment anchor
func (t *AlignView) SetAnchor(v AlignAnchors) *AlignView { t.Anchor = v; return t }

// SetVector sets the [AlignView.Vector]:
// the parent vector
func (t *AlignView) SetVector(v *Vector) *AlignView { t.Vector = v; return t }

// PaintViewType is the [types.Type] for [PaintView]
var PaintViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/vector.PaintView", IDName: "paint-view", Doc: "PaintView provides editing of basic Stroke and Fill painting parameters\nfor selected items", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "StrokeType", Doc: "paint type for stroke"}, {Name: "StrokeStops", Doc: "name of gradient with stops"}, {Name: "FillType", Doc: "paint type for fill"}, {Name: "FillStops", Doc: "name of gradient with stops"}, {Name: "Vector", Doc: "the parent vector"}}, Instance: &PaintView{}})

// NewPaintView returns a new [PaintView] with the given optional parent:
// PaintView provides editing of basic Stroke and Fill painting parameters
// for selected items
func NewPaintView(parent ...tree.Node) *PaintView { return tree.New[*PaintView](parent...) }

// NodeType returns the [*types.Type] of [PaintView]
func (t *PaintView) NodeType() *types.Type { return PaintViewType }

// New returns a new [*PaintView] value
func (t *PaintView) New() tree.Node { return &PaintView{} }

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

// SetVector sets the [PaintView.Vector]:
// the parent vector
func (t *PaintView) SetVector(v *Vector) *PaintView { t.Vector = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/vector.PhysSize", IDName: "phys-size", Doc: "PhysSize specifies the physical size of the drawing, when making a new one", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "StandardSize", Doc: "select a standard size -- this will set units and size"}, {Name: "Portrait", Doc: "for standard size, use first number as width, second as height"}, {Name: "Units", Doc: "default units to use, e.g., in line widths etc"}, {Name: "Size", Doc: "drawing size, in Units"}, {Name: "Grid", Doc: "grid spacing, in units of ViewBox size"}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/vector.Settings", IDName: "settings", Doc: "Settings is the overall Vector settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "Size", Doc: "default physical size, when app is started without opening a file"}, {Name: "Colors", Doc: "active color settings"}, {Name: "ColorSchemes", Doc: "named color schemes -- has Light and Dark schemes by default"}, {Name: "ShapeStyle", Doc: "default shape styles"}, {Name: "TextStyle", Doc: "default text styles"}, {Name: "PathStyle", Doc: "default line styles"}, {Name: "LineStyle", Doc: "default line styles"}, {Name: "VectorDisp", Doc: "turns on the grid display"}, {Name: "SnapVector", Doc: "snap positions and sizes to underlying grid"}, {Name: "SnapGuide", Doc: "snap positions and sizes to line up with other elements"}, {Name: "SnapNodes", Doc: "snap node movements to align with guides"}, {Name: "SnapTol", Doc: "number of screen pixels around target point (in either direction) to snap"}, {Name: "SplitName", Doc: "named-split config in use for configuring the splitters"}, {Name: "EnvVars", Doc: "environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app"}, {Name: "Changed", Doc: "flag that is set by Form by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/vector.ColorSettings", IDName: "color-settings", Doc: "ColorSettings for", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "Background", Doc: "drawing background color"}, {Name: "Border", Doc: "border color of the drawing"}, {Name: "Vector", Doc: "grid line color"}}})

// SVGType is the [types.Type] for [SVG]
var SVGType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/vector.SVG", IDName: "svg", Doc: "SVG is the element for viewing and interacting with the SVG.", Embeds: []types.Field{{Name: "SVG"}}, Fields: []types.Field{{Name: "Vector", Doc: "the parent vector"}, {Name: "Trans", Doc: "view translation offset (from dragging)"}, {Name: "Scale", Doc: "view scaling (from zooming)"}, {Name: "Grid", Doc: "grid spacing, in native ViewBox units"}, {Name: "VectorEff", Doc: "effective grid spacing given Scale level"}, {Name: "BgPixels", Doc: "background pixels, includes page outline and grid"}, {Name: "bgTrans", Doc: "bg rendered translation"}, {Name: "bgScale", Doc: "bg rendered scale"}, {Name: "bgVectorEff", Doc: "bg rendered grid"}}, Instance: &SVG{}})

// NewSVG returns a new [SVG] with the given optional parent:
// SVG is the element for viewing and interacting with the SVG.
func NewSVG(parent ...tree.Node) *SVG { return tree.New[*SVG](parent...) }

// NodeType returns the [*types.Type] of [SVG]
func (t *SVG) NodeType() *types.Type { return SVGType }

// New returns a new [*SVG] value
func (t *SVG) New() tree.Node { return &SVG{} }

// TreeType is the [types.Type] for [Tree]
var TreeType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/vector.Tree", IDName: "tree", Doc: "Tree is a [core.Tree] that interacts properly with [Vector].", Embeds: []types.Field{{Name: "Tree"}}, Fields: []types.Field{{Name: "Vector", Doc: "the parent vector"}}, Instance: &Tree{}})

// NewTree returns a new [Tree] with the given optional parent:
// Tree is a [core.Tree] that interacts properly with [Vector].
func NewTree(parent ...tree.Node) *Tree { return tree.New[*Tree](parent...) }

// NodeType returns the [*types.Type] of [Tree]
func (t *Tree) NodeType() *types.Type { return TreeType }

// New returns a new [*Tree] value
func (t *Tree) New() tree.Node { return &Tree{} }

// SetVector sets the [Tree.Vector]:
// the parent vector
func (t *Tree) SetVector(v *Vector) *Tree { t.Vector = v; return t }

// VectorType is the [types.Type] for [Vector]
var VectorType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/vector.Vector", IDName: "vector", Doc: "Vector is the main widget of the Cogent Vector SVG vector graphics program.", Methods: []types.Method{{Name: "AddLayer", Doc: "AddLayer adds a new layer", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectGroup", Doc: "SelectGroup groups items together", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectUnGroup", Doc: "SelectUnGroup ungroups items from each other", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectRotateLeft", Doc: "SelectRotateLeft rotates the selection 90 degrees counter-clockwise", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectRotateRight", Doc: "SelectRotateRight rotates the selection 90 degrees clockwise", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectFlipHorizontal", Doc: "SelectFlipHorizontal flips the selection horizontally", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectFlipVertical", Doc: "SelectFlipVertical flips the selection vertically", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectRaiseTop", Doc: "SelectRaiseTop raises the selection to the top of the layer", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectRaise", Doc: "SelectRaise raises the selection by one level in the layer", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectLowerBottom", Doc: "SelectLowerBottom lowers the selection to the bottom of the layer", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SelectLower", Doc: "SelectLower lowers the selection by one level in the layer", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "DuplicateSelected", Doc: "DuplicateSelected duplicates selected items in SVG view, using Tree methods", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CopySelected", Doc: "CopySelected copies selected items in SVG view, using Tree methods", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CutSelected", Doc: "CutSelected cuts selected items in SVG view, using Tree methods", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "PasteClip", Doc: "PasteClip pastes clipboard, using cur layer etc", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "OpenDrawing", Doc: "OpenDrawing opens a new .svg drawing", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnm"}, Returns: []string{"error"}}, {Name: "PromptPhysSize", Doc: "PromptPhysSize prompts for the physical size of the drawing and sets it", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SaveDrawing", Doc: "SaveDrawing saves .svg drawing to current filename", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"error"}}, {Name: "SaveDrawingAs", Doc: "SaveDrawingAs saves .svg drawing to given filename", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fname"}, Returns: []string{"error"}}, {Name: "ExportPNG", Doc: "ExportPNG exports drawing to a PNG image (auto-names to same name\nwith .png suffix).  Calls inkscape -- needs to be on the PATH.\nspecify either width or height of resulting image, or nothing for\nphysical size as set.  Renders full current page -- do ResizeToContents\nto render just current contents.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"width", "height"}, Returns: []string{"error"}}, {Name: "ExportPDF", Doc: "ExportPDF exports drawing to a PDF file (auto-names to same name\nwith .pdf suffix).  Calls inkscape -- needs to be on the PATH.\nspecify DPI of resulting image for effects rendering.\nRenders full current page -- do ResizeToContents\nto render just current contents.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"dpi"}, Returns: []string{"error"}}, {Name: "ResizeToContents", Doc: "ResizeToContents resizes the drawing to just fit the current contents,\nincluding moving everything to start at upper-left corner,\npreserving the current grid offset, so grid snapping\nis preserved.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "AddImage", Doc: "AddImage adds a new image node set to the given image", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fname", "width", "height"}, Returns: []string{"error"}}, {Name: "UpdateAll", Doc: "UpdateAll updates the display", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Undo", Doc: "Undo undoes the last action", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"string"}}, {Name: "Redo", Doc: "Redo redoes the previously undone action", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"string"}}}, Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Filename", Doc: "full path to current drawing filename"}, {Name: "EditState", Doc: "current edit state"}}, Instance: &Vector{}})

// NewVector returns a new [Vector] with the given optional parent:
// Vector is the main widget of the Cogent Vector SVG vector graphics program.
func NewVector(parent ...tree.Node) *Vector { return tree.New[*Vector](parent...) }

// NodeType returns the [*types.Type] of [Vector]
func (t *Vector) NodeType() *types.Type { return VectorType }

// New returns a new [*Vector] value
func (t *Vector) New() tree.Node { return &Vector{} }
