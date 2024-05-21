// Code generated by "core generate"; DO NOT EDIT.

package databrowser

import (
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// BrowserType is the [types.Type] for [Browser]
var BrowserType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/numbers/databrowser.Browser", IDName: "browser", Doc: "Browser is a data browser, for browsing data typically organized into\nseparate directories, with .cosh Scripts as toolbar actions to perform\nregular tasks on the data.\nScripts are ordered alphabetically and any leading #- prefix is automatically\nremoved from the label, so you can use numbers to specify a custom order.", Methods: []types.Method{{Name: "UpdateFiles", Doc: "UpdateFiles Updates the file view with current files in DataRoot", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}}, Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "DataRoot", Doc: "DataRoot is the path to the root of the data to browse"}, {Name: "ScriptsDir", Doc: "ScriptsDir is the directory containing scripts for toolbar actions.\nIt defaults to DataDir/dbscripts"}, {Name: "Scripts", Doc: "Scripts"}, {Name: "ScriptInterp", Doc: "ScriptInterp is the interpreter to use for running Browser scripts"}}, Instance: &Browser{}})

// NewBrowser returns a new [Browser] with the given optional parent:
// Browser is a data browser, for browsing data typically organized into
// separate directories, with .cosh Scripts as toolbar actions to perform
// regular tasks on the data.
// Scripts are ordered alphabetically and any leading #- prefix is automatically
// removed from the label, so you can use numbers to specify a custom order.
func NewBrowser(parent ...tree.Node) *Browser { return tree.New[*Browser](parent...) }

// NodeType returns the [*types.Type] of [Browser]
func (t *Browser) NodeType() *types.Type { return BrowserType }

// New returns a new [*Browser] value
func (t *Browser) New() tree.Node { return &Browser{} }

// SetDataRoot sets the [Browser.DataRoot]:
// DataRoot is the path to the root of the data to browse
func (t *Browser) SetDataRoot(v string) *Browser { t.DataRoot = v; return t }

// SetScriptsDir sets the [Browser.ScriptsDir]:
// ScriptsDir is the directory containing scripts for toolbar actions.
// It defaults to DataDir/dbscripts
func (t *Browser) SetScriptsDir(v string) *Browser { t.ScriptsDir = v; return t }

// SetTooltip sets the [Browser.Tooltip]
func (t *Browser) SetTooltip(v string) *Browser { t.Tooltip = v; return t }