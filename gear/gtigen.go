// Code generated by "core generate -add-types"; DO NOT EDIT.

package gear

import (
	"cogentcore.org/core/gti"
	"cogentcore.org/core/ki"
)

// AppType is the [gti.Type] for [App]
var AppType = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/gear.App", IDName: "app", Doc: "App is a GUI view of a gear command.", Embeds: []gti.Field{{Name: "Frame"}}, Fields: []gti.Field{{Name: "Cmd", Doc: "Cmd is the root command associated with this app."}, {Name: "CurCmd", Doc: "CurCmd is the current root command being typed in."}, {Name: "Dir", Doc: "Dir is the current directory of the app."}}, Instance: &App{}})

// NewApp adds a new [App] with the given name to the given parent:
// App is a GUI view of a gear command.
func NewApp(par ki.Ki, name ...string) *App {
	return par.NewChild(AppType, name...).(*App)
}

// KiType returns the [*gti.Type] of [App]
func (t *App) KiType() *gti.Type { return AppType }

// New returns a new [*App] value
func (t *App) New() ki.Ki { return &App{} }

// SetCmd sets the [App.Cmd]:
// Cmd is the root command associated with this app.
func (t *App) SetCmd(v *Cmd) *App { t.Cmd = v; return t }

// SetCurCmd sets the [App.CurCmd]:
// CurCmd is the current root command being typed in.
func (t *App) SetCurCmd(v string) *App { t.CurCmd = v; return t }

// SetDir sets the [App.Dir]:
// Dir is the current directory of the app.
func (t *App) SetDir(v string) *App { t.Dir = v; return t }

// SetTooltip sets the [App.Tooltip]
func (t *App) SetTooltip(v string) *App { t.Tooltip = v; return t }

// SetStackTop sets the [App.StackTop]
func (t *App) SetStackTop(v int) *App { t.StackTop = v; return t }

var _ = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/gear.Cmd", IDName: "cmd", Doc: "Cmd contains all of the data for a parsed command line command.", Fields: []gti.Field{{Name: "Cmd", Doc: "Cmd is the actual name of the command (eg: \"git\", \"go build\")"}, {Name: "Name", Doc: "Name is the formatted name of the command (eg: \"Git\", \"Go build\")"}, {Name: "Doc", Doc: "Doc is the documentation for the command (eg: \"compile packages and dependencies\")"}, {Name: "Flags", Doc: "Flags contains the flags for the command"}, {Name: "Cmds", Doc: "Cmds contains the subcommands of the command"}}})

var _ = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/gear.Flag", IDName: "flag", Doc: "Flag contains the information for a parsed command line flag.", Fields: []gti.Field{{Name: "Name", Doc: "Name is the canonical (longest) name of the flag.\nIt includes the leading dashes of the flag."}, {Name: "Names", Doc: "Names are the different names the flag can go by.\nThey include the leading dashes of the flag."}, {Name: "Type", Doc: "Type is the type or value hint for the flag."}, {Name: "Doc", Doc: "Doc is the documentation for the flag."}}})

var _ = gti.AddType(&gti.Type{Name: "cogentcore.org/cogent/gear.ParseBlock", IDName: "parse-block", Doc: "ParseBlock is a block of parsed content containing the name of something and\nthe documentation for it.", Fields: []gti.Field{{Name: "Name"}, {Name: "Doc"}}})
