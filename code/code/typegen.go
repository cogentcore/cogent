// Code generated by "core generate"; DO NOT EDIT.

package code

import (
	"image"
	"regexp"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/syms"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"cogentcore.org/core/units"
	"cogentcore.org/core/views"
)

// DebugViewType is the [types.Type] for [DebugView]
var DebugViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.DebugView", IDName: "debug-view", Doc: "DebugView is the debugger", Embeds: []types.Field{{Name: "Layout"}}, Fields: []types.Field{{Name: "Sup", Doc: "supported file type to determine debugger"}, {Name: "ExePath", Doc: "path to executable / dir to debug"}, {Name: "DbgTime", Doc: "time when dbg was last restarted"}, {Name: "Dbg", Doc: "the debugger"}, {Name: "State", Doc: "all relevant debug state info"}, {Name: "CurFileLoc", Doc: "current ShowFile location -- cleared before next one or run"}, {Name: "BBreaks", Doc: "backup breakpoints list -- to track deletes"}, {Name: "OutputBuffer", Doc: "output from the debugger"}, {Name: "Code", Doc: "parent code project"}}, Instance: &DebugView{}})

// NewDebugView adds a new [DebugView] with the given name to the given parent:
// DebugView is the debugger
func NewDebugView(parent tree.Node, name ...string) *DebugView {
	return parent.NewChild(DebugViewType, name...).(*DebugView)
}

// NodeType returns the [*types.Type] of [DebugView]
func (t *DebugView) NodeType() *types.Type { return DebugViewType }

// New returns a new [*DebugView] value
func (t *DebugView) New() tree.Node { return &DebugView{} }

// SetSup sets the [DebugView.Sup]:
// supported file type to determine debugger
func (t *DebugView) SetSup(v fileinfo.Known) *DebugView { t.Sup = v; return t }

// SetExePath sets the [DebugView.ExePath]:
// path to executable / dir to debug
func (t *DebugView) SetExePath(v string) *DebugView { t.ExePath = v; return t }

// SetDbgTime sets the [DebugView.DbgTime]:
// time when dbg was last restarted
func (t *DebugView) SetDbgTime(v time.Time) *DebugView { t.DbgTime = v; return t }

// SetTooltip sets the [DebugView.Tooltip]
func (t *DebugView) SetTooltip(v string) *DebugView { t.Tooltip = v; return t }

// StackViewType is the [types.Type] for [StackView]
var StackViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.StackView", IDName: "stack-view", Doc: "StackView is a view of the stack trace", Embeds: []types.Field{{Name: "Layout"}}, Fields: []types.Field{{Name: "FindFrames", Doc: "if true, this is a find frames, not a regular stack"}}, Instance: &StackView{}})

// NewStackView adds a new [StackView] with the given name to the given parent:
// StackView is a view of the stack trace
func NewStackView(parent tree.Node, name ...string) *StackView {
	return parent.NewChild(StackViewType, name...).(*StackView)
}

// NodeType returns the [*types.Type] of [StackView]
func (t *StackView) NodeType() *types.Type { return StackViewType }

// New returns a new [*StackView] value
func (t *StackView) New() tree.Node { return &StackView{} }

// SetFindFrames sets the [StackView.FindFrames]:
// if true, this is a find frames, not a regular stack
func (t *StackView) SetFindFrames(v bool) *StackView { t.FindFrames = v; return t }

// SetTooltip sets the [StackView.Tooltip]
func (t *StackView) SetTooltip(v string) *StackView { t.Tooltip = v; return t }

// BreakViewType is the [types.Type] for [BreakView]
var BreakViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.BreakView", IDName: "break-view", Doc: "BreakView is a view of the breakpoints", Embeds: []types.Field{{Name: "Layout"}}, Instance: &BreakView{}})

// NewBreakView adds a new [BreakView] with the given name to the given parent:
// BreakView is a view of the breakpoints
func NewBreakView(parent tree.Node, name ...string) *BreakView {
	return parent.NewChild(BreakViewType, name...).(*BreakView)
}

// NodeType returns the [*types.Type] of [BreakView]
func (t *BreakView) NodeType() *types.Type { return BreakViewType }

// New returns a new [*BreakView] value
func (t *BreakView) New() tree.Node { return &BreakView{} }

// SetTooltip sets the [BreakView.Tooltip]
func (t *BreakView) SetTooltip(v string) *BreakView { t.Tooltip = v; return t }

// ThreadViewType is the [types.Type] for [ThreadView]
var ThreadViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.ThreadView", IDName: "thread-view", Doc: "ThreadView is a view of the threads", Embeds: []types.Field{{Name: "Layout"}}, Instance: &ThreadView{}})

// NewThreadView adds a new [ThreadView] with the given name to the given parent:
// ThreadView is a view of the threads
func NewThreadView(parent tree.Node, name ...string) *ThreadView {
	return parent.NewChild(ThreadViewType, name...).(*ThreadView)
}

// NodeType returns the [*types.Type] of [ThreadView]
func (t *ThreadView) NodeType() *types.Type { return ThreadViewType }

// New returns a new [*ThreadView] value
func (t *ThreadView) New() tree.Node { return &ThreadView{} }

// SetTooltip sets the [ThreadView.Tooltip]
func (t *ThreadView) SetTooltip(v string) *ThreadView { t.Tooltip = v; return t }

// TaskViewType is the [types.Type] for [TaskView]
var TaskViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.TaskView", IDName: "task-view", Doc: "TaskView is a view of the threads", Embeds: []types.Field{{Name: "Layout"}}, Instance: &TaskView{}})

// NewTaskView adds a new [TaskView] with the given name to the given parent:
// TaskView is a view of the threads
func NewTaskView(parent tree.Node, name ...string) *TaskView {
	return parent.NewChild(TaskViewType, name...).(*TaskView)
}

// NodeType returns the [*types.Type] of [TaskView]
func (t *TaskView) NodeType() *types.Type { return TaskViewType }

// New returns a new [*TaskView] value
func (t *TaskView) New() tree.Node { return &TaskView{} }

// SetTooltip sets the [TaskView.Tooltip]
func (t *TaskView) SetTooltip(v string) *TaskView { t.Tooltip = v; return t }

// VarsViewType is the [types.Type] for [VarsView]
var VarsViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.VarsView", IDName: "vars-view", Doc: "VarsView is a view of the variables", Embeds: []types.Field{{Name: "Layout"}}, Fields: []types.Field{{Name: "GlobalVars", Doc: "if true, this is global vars, not local ones"}}, Instance: &VarsView{}})

// NewVarsView adds a new [VarsView] with the given name to the given parent:
// VarsView is a view of the variables
func NewVarsView(parent tree.Node, name ...string) *VarsView {
	return parent.NewChild(VarsViewType, name...).(*VarsView)
}

// NodeType returns the [*types.Type] of [VarsView]
func (t *VarsView) NodeType() *types.Type { return VarsViewType }

// New returns a new [*VarsView] value
func (t *VarsView) New() tree.Node { return &VarsView{} }

// SetGlobalVars sets the [VarsView.GlobalVars]:
// if true, this is global vars, not local ones
func (t *VarsView) SetGlobalVars(v bool) *VarsView { t.GlobalVars = v; return t }

// SetTooltip sets the [VarsView.Tooltip]
func (t *VarsView) SetTooltip(v string) *VarsView { t.Tooltip = v; return t }

// VarViewType is the [types.Type] for [VarView]
var VarViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.VarView", IDName: "var-view", Doc: "VarView shows a debug variable in an inspector-like framework,\nwith sub-variables in a tree.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Var", Doc: "variable being edited"}, {Name: "SelectVar"}, {Name: "FrameInfo", Doc: "frame info"}, {Name: "DbgView", Doc: "parent DebugView"}}, Instance: &VarView{}})

// NewVarView adds a new [VarView] with the given name to the given parent:
// VarView shows a debug variable in an inspector-like framework,
// with sub-variables in a tree.
func NewVarView(parent tree.Node, name ...string) *VarView {
	return parent.NewChild(VarViewType, name...).(*VarView)
}

// NodeType returns the [*types.Type] of [VarView]
func (t *VarView) NodeType() *types.Type { return VarViewType }

// New returns a new [*VarView] value
func (t *VarView) New() tree.Node { return &VarView{} }

// SetDbgView sets the [VarView.DbgView]:
// parent DebugView
func (t *VarView) SetDbgView(v *DebugView) *VarView { t.DbgView = v; return t }

// SetTooltip sets the [VarView.Tooltip]
func (t *VarView) SetTooltip(v string) *VarView { t.Tooltip = v; return t }

// FileNodeType is the [types.Type] for [FileNode]
var FileNodeType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.FileNode", IDName: "file-node", Doc: "FileNode is Code version of FileNode for FileTree view", Methods: []types.Method{{Name: "ExecCmdFile", Doc: "ExecCmdFile pops up a menu to select a command appropriate for the given node,\nand shows output in MainTab with name of command", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditFiles", Doc: "EditFiles calls EditFile on selected files", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SetRunExecs", Doc: "SetRunExecs sets executable as the RunExec executable that will be run with Run / Debug buttons", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}}, Embeds: []types.Field{{Name: "Node"}}, Instance: &FileNode{}})

// NewFileNode adds a new [FileNode] with the given name to the given parent:
// FileNode is Code version of FileNode for FileTree view
func NewFileNode(parent tree.Node, name ...string) *FileNode {
	return parent.NewChild(FileNodeType, name...).(*FileNode)
}

// NodeType returns the [*types.Type] of [FileNode]
func (t *FileNode) NodeType() *types.Type { return FileNodeType }

// New returns a new [*FileNode] value
func (t *FileNode) New() tree.Node { return &FileNode{} }

// SetTooltip sets the [FileNode.Tooltip]
func (t *FileNode) SetTooltip(v string) *FileNode { t.Tooltip = v; return t }

// SetText sets the [FileNode.Text]
func (t *FileNode) SetText(v string) *FileNode { t.Text = v; return t }

// SetIcon sets the [FileNode.Icon]
func (t *FileNode) SetIcon(v icons.Icon) *FileNode { t.Icon = v; return t }

// SetIconOpen sets the [FileNode.IconOpen]
func (t *FileNode) SetIconOpen(v icons.Icon) *FileNode { t.IconOpen = v; return t }

// SetIconClosed sets the [FileNode.IconClosed]
func (t *FileNode) SetIconClosed(v icons.Icon) *FileNode { t.IconClosed = v; return t }

// SetIconLeaf sets the [FileNode.IconLeaf]
func (t *FileNode) SetIconLeaf(v icons.Icon) *FileNode { t.IconLeaf = v; return t }

// SetIndent sets the [FileNode.Indent]
func (t *FileNode) SetIndent(v units.Value) *FileNode { t.Indent = v; return t }

// SetOpenDepth sets the [FileNode.OpenDepth]
func (t *FileNode) SetOpenDepth(v int) *FileNode { t.OpenDepth = v; return t }

// SetViewIndex sets the [FileNode.ViewIndex]
func (t *FileNode) SetViewIndex(v int) *FileNode { t.ViewIndex = v; return t }

// SetWidgetSize sets the [FileNode.WidgetSize]
func (t *FileNode) SetWidgetSize(v math32.Vector2) *FileNode { t.WidgetSize = v; return t }

// SetRootView sets the [FileNode.RootView]
func (t *FileNode) SetRootView(v *views.TreeView) *FileNode { t.RootView = v; return t }

// SetSelectedNodes sets the [FileNode.SelectedNodes]
func (t *FileNode) SetSelectedNodes(v ...views.TreeViewer) *FileNode { t.SelectedNodes = v; return t }

// FindViewType is the [types.Type] for [FindView]
var FindViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.FindView", IDName: "find-view", Doc: "FindView is a find / replace widget that displays results in a TextEditor\nand has a toolbar for controlling find / replace process.", Embeds: []types.Field{{Name: "Layout"}}, Fields: []types.Field{{Name: "Code", Doc: "parent code project"}, {Name: "LangVV", Doc: "langs value view"}, {Name: "Time", Doc: "time of last find"}, {Name: "Re", Doc: "compiled regexp"}}, Instance: &FindView{}})

// NewFindView adds a new [FindView] with the given name to the given parent:
// FindView is a find / replace widget that displays results in a TextEditor
// and has a toolbar for controlling find / replace process.
func NewFindView(parent tree.Node, name ...string) *FindView {
	return parent.NewChild(FindViewType, name...).(*FindView)
}

// NodeType returns the [*types.Type] of [FindView]
func (t *FindView) NodeType() *types.Type { return FindViewType }

// New returns a new [*FindView] value
func (t *FindView) New() tree.Node { return &FindView{} }

// SetCode sets the [FindView.Code]:
// parent code project
func (t *FindView) SetCode(v Code) *FindView { t.Code = v; return t }

// SetLangVV sets the [FindView.LangVV]:
// langs value view
func (t *FindView) SetLangVV(v views.Value) *FindView { t.LangVV = v; return t }

// SetTime sets the [FindView.Time]:
// time of last find
func (t *FindView) SetTime(v time.Time) *FindView { t.Time = v; return t }

// SetRe sets the [FindView.Re]:
// compiled regexp
func (t *FindView) SetRe(v *regexp.Regexp) *FindView { t.Re = v; return t }

// SetTooltip sets the [FindView.Tooltip]
func (t *FindView) SetTooltip(v string) *FindView { t.Tooltip = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.SettingsData", IDName: "settings-data", Doc: "SettingsData is the data type for the overall user settings for Code.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Methods: []types.Method{{Name: "Apply", Doc: "Apply settings updates things according with settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditKeyMaps", Doc: "EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /\nload from other files, etc.  Current avail keymaps are saved and loaded\nwith settings automatically.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditLangOpts", Doc: "EditLangOpts opens the LangsView editor to customize options for each type of\nlanguage / data / file type.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditCmds", Doc: "EditCmds opens the CmdsView editor to customize commands you can run.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditSplits", Doc: "EditSplits opens the SplitsView editor to customize saved splitter settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditRegisters", Doc: "EditRegisters opens the RegistersView editor to customize saved registers", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}}, Embeds: []types.Field{{Name: "SettingsBase"}}, Fields: []types.Field{{Name: "Files", Doc: "file view settings"}, {Name: "EnvVars", Doc: "environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app"}, {Name: "KeyMap", Doc: "key map for code-specific keyboard sequences"}, {Name: "SaveKeyMaps", Doc: "if set, the current available set of key maps is saved to your settings directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords"}, {Name: "SaveLangOpts", Doc: "if set, the current customized set of language options (see Edit Lang Opts) is saved / loaded along with other settings -- if not set, then you always are using the default compiled-in standard set (which will be updated)"}, {Name: "SaveCmds", Doc: "if set, the current customized set of command parameters (see Edit Cmds) is saved / loaded along with other settings -- if not set, then you always are using the default compiled-in standard set (which will be updated)"}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.FileSettings", IDName: "file-settings", Doc: "FileSettings contains file view settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "DirsOnTop", Doc: "if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted"}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.ProjSettings", IDName: "proj-settings", Doc: "ProjSettings are the settings for saving for a project. This IS the project file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Methods: []types.Method{{Name: "Open", Doc: "Open open from file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}, Returns: []string{"error"}}, {Name: "Save", Doc: "Save save to file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}, Returns: []string{"error"}}}, Fields: []types.Field{{Name: "Files", Doc: "file view settings"}, {Name: "Editor", Doc: "editor settings"}, {Name: "SplitName", Doc: "current named-split config in use for configuring the splitters"}, {Name: "MainLang", Doc: "the language associated with the most frequently-encountered file\nextension in the file tree -- can be manually set here as well"}, {Name: "VersionControl", Doc: "the type of version control system used in this project (git, svn, etc).\nfilters commands available"}, {Name: "ProjectFilename", Doc: "current project filename for saving / loading specific Code\nconfiguration information in a .code file (optional)"}, {Name: "ProjectRoot", Doc: "root directory for the project. all projects must be organized within\na top-level root directory, with all the files therein constituting\nthe scope of the project. By default it is the path for ProjectFilename"}, {Name: "GoMod", Doc: "if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically"}, {Name: "BuildCmds", Doc: "command(s) to run for main Build button"}, {Name: "BuildDir", Doc: "build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands"}, {Name: "BuildTarg", Doc: "build target for main Build button, if relevant for your  BuildCmds"}, {Name: "RunExec", Doc: "executable to run for this project via main Run button -- called by standard Run Proj command"}, {Name: "RunCmds", Doc: "command(s) to run for main Run button (typically Run Proj)"}, {Name: "Debug", Doc: "custom debugger parameters for this project"}, {Name: "Find", Doc: "saved find params"}, {Name: "Symbols", Doc: "saved structure params"}, {Name: "Dirs", Doc: "directory properties"}, {Name: "Register", Doc: "last register used"}, {Name: "Splits", Doc: "current splitter splits"}}})

// SpellViewType is the [types.Type] for [SpellView]
var SpellViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.SpellView", IDName: "spell-view", Doc: "SpellView is a widget that displays results of spell check", Embeds: []types.Field{{Name: "Layout"}}, Fields: []types.Field{{Name: "Code", Doc: "parent code project"}, {Name: "Text", Doc: "texteditor that we're spell-checking"}, {Name: "Errs", Doc: "current spelling errors"}, {Name: "CurLn", Doc: "current line in text we're on"}, {Name: "CurIndex", Doc: "current index in Errs we're on"}, {Name: "UnkLex", Doc: "current unknown lex token"}, {Name: "UnkWord", Doc: "current unknown word"}, {Name: "Suggest", Doc: "a list of suggestions from spell checker"}, {Name: "LastAction", Doc: "last user action (ignore, change, learn)"}}, Instance: &SpellView{}})

// NewSpellView adds a new [SpellView] with the given name to the given parent:
// SpellView is a widget that displays results of spell check
func NewSpellView(parent tree.Node, name ...string) *SpellView {
	return parent.NewChild(SpellViewType, name...).(*SpellView)
}

// NodeType returns the [*types.Type] of [SpellView]
func (t *SpellView) NodeType() *types.Type { return SpellViewType }

// New returns a new [*SpellView] value
func (t *SpellView) New() tree.Node { return &SpellView{} }

// SetCode sets the [SpellView.Code]:
// parent code project
func (t *SpellView) SetCode(v Code) *SpellView { t.Code = v; return t }

// SetText sets the [SpellView.Text]:
// texteditor that we're spell-checking
func (t *SpellView) SetText(v *TextEditor) *SpellView { t.Text = v; return t }

// SetErrs sets the [SpellView.Errs]:
// current spelling errors
func (t *SpellView) SetErrs(v lexer.Line) *SpellView { t.Errs = v; return t }

// SetCurLn sets the [SpellView.CurLn]:
// current line in text we're on
func (t *SpellView) SetCurLn(v int) *SpellView { t.CurLn = v; return t }

// SetCurIndex sets the [SpellView.CurIndex]:
// current index in Errs we're on
func (t *SpellView) SetCurIndex(v int) *SpellView { t.CurIndex = v; return t }

// SetUnkLex sets the [SpellView.UnkLex]:
// current unknown lex token
func (t *SpellView) SetUnkLex(v lexer.Lex) *SpellView { t.UnkLex = v; return t }

// SetUnkWord sets the [SpellView.UnkWord]:
// current unknown word
func (t *SpellView) SetUnkWord(v string) *SpellView { t.UnkWord = v; return t }

// SetSuggest sets the [SpellView.Suggest]:
// a list of suggestions from spell checker
func (t *SpellView) SetSuggest(v ...string) *SpellView { t.Suggest = v; return t }

// SetLastAction sets the [SpellView.LastAction]:
// last user action (ignore, change, learn)
func (t *SpellView) SetLastAction(v *core.Button) *SpellView { t.LastAction = v; return t }

// SetTooltip sets the [SpellView.Tooltip]
func (t *SpellView) SetTooltip(v string) *SpellView { t.Tooltip = v; return t }

// SymbolsViewType is the [types.Type] for [SymbolsView]
var SymbolsViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.SymbolsView", IDName: "symbols-view", Doc: "SymbolsView is a widget that displays results of a file or package parse", Embeds: []types.Field{{Name: "Layout"}}, Fields: []types.Field{{Name: "Code", Doc: "parent code project"}, {Name: "SymParams", Doc: "params for structure display"}, {Name: "Syms", Doc: "all the symbols for the file or package in a tree"}, {Name: "Match", Doc: "only show symbols that match this string"}}, Instance: &SymbolsView{}})

// NewSymbolsView adds a new [SymbolsView] with the given name to the given parent:
// SymbolsView is a widget that displays results of a file or package parse
func NewSymbolsView(parent tree.Node, name ...string) *SymbolsView {
	return parent.NewChild(SymbolsViewType, name...).(*SymbolsView)
}

// NodeType returns the [*types.Type] of [SymbolsView]
func (t *SymbolsView) NodeType() *types.Type { return SymbolsViewType }

// New returns a new [*SymbolsView] value
func (t *SymbolsView) New() tree.Node { return &SymbolsView{} }

// SetCode sets the [SymbolsView.Code]:
// parent code project
func (t *SymbolsView) SetCode(v Code) *SymbolsView { t.Code = v; return t }

// SetSymParams sets the [SymbolsView.SymParams]:
// params for structure display
func (t *SymbolsView) SetSymParams(v SymbolsParams) *SymbolsView { t.SymParams = v; return t }

// SetSyms sets the [SymbolsView.Syms]:
// all the symbols for the file or package in a tree
func (t *SymbolsView) SetSyms(v *SymNode) *SymbolsView { t.Syms = v; return t }

// SetMatch sets the [SymbolsView.Match]:
// only show symbols that match this string
func (t *SymbolsView) SetMatch(v string) *SymbolsView { t.Match = v; return t }

// SetTooltip sets the [SymbolsView.Tooltip]
func (t *SymbolsView) SetTooltip(v string) *SymbolsView { t.Tooltip = v; return t }

// SymNodeType is the [types.Type] for [SymNode]
var SymNodeType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.SymNode", IDName: "sym-node", Doc: "SymNode represents a language symbol -- the name of the node is\nthe name of the symbol. Some symbols, e.g. type have children", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Symbol", Doc: "the symbol"}}, Instance: &SymNode{}})

// NewSymNode adds a new [SymNode] with the given name to the given parent:
// SymNode represents a language symbol -- the name of the node is
// the name of the symbol. Some symbols, e.g. type have children
func NewSymNode(parent tree.Node, name ...string) *SymNode {
	return parent.NewChild(SymNodeType, name...).(*SymNode)
}

// NodeType returns the [*types.Type] of [SymNode]
func (t *SymNode) NodeType() *types.Type { return SymNodeType }

// New returns a new [*SymNode] value
func (t *SymNode) New() tree.Node { return &SymNode{} }

// SetSymbol sets the [SymNode.Symbol]:
// the symbol
func (t *SymNode) SetSymbol(v syms.Symbol) *SymNode { t.Symbol = v; return t }

// SymTreeViewType is the [types.Type] for [SymTreeView]
var SymTreeViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.SymTreeView", IDName: "sym-tree-view", Doc: "SymTreeView is a TreeView that knows how to operate on FileNode nodes", Embeds: []types.Field{{Name: "TreeView"}}, Instance: &SymTreeView{}})

// NewSymTreeView adds a new [SymTreeView] with the given name to the given parent:
// SymTreeView is a TreeView that knows how to operate on FileNode nodes
func NewSymTreeView(parent tree.Node, name ...string) *SymTreeView {
	return parent.NewChild(SymTreeViewType, name...).(*SymTreeView)
}

// NodeType returns the [*types.Type] of [SymTreeView]
func (t *SymTreeView) NodeType() *types.Type { return SymTreeViewType }

// New returns a new [*SymTreeView] value
func (t *SymTreeView) New() tree.Node { return &SymTreeView{} }

// SetTooltip sets the [SymTreeView.Tooltip]
func (t *SymTreeView) SetTooltip(v string) *SymTreeView { t.Tooltip = v; return t }

// SetText sets the [SymTreeView.Text]
func (t *SymTreeView) SetText(v string) *SymTreeView { t.Text = v; return t }

// SetIcon sets the [SymTreeView.Icon]
func (t *SymTreeView) SetIcon(v icons.Icon) *SymTreeView { t.Icon = v; return t }

// SetIconOpen sets the [SymTreeView.IconOpen]
func (t *SymTreeView) SetIconOpen(v icons.Icon) *SymTreeView { t.IconOpen = v; return t }

// SetIconClosed sets the [SymTreeView.IconClosed]
func (t *SymTreeView) SetIconClosed(v icons.Icon) *SymTreeView { t.IconClosed = v; return t }

// SetIconLeaf sets the [SymTreeView.IconLeaf]
func (t *SymTreeView) SetIconLeaf(v icons.Icon) *SymTreeView { t.IconLeaf = v; return t }

// SetIndent sets the [SymTreeView.Indent]
func (t *SymTreeView) SetIndent(v units.Value) *SymTreeView { t.Indent = v; return t }

// SetOpenDepth sets the [SymTreeView.OpenDepth]
func (t *SymTreeView) SetOpenDepth(v int) *SymTreeView { t.OpenDepth = v; return t }

// SetViewIndex sets the [SymTreeView.ViewIndex]
func (t *SymTreeView) SetViewIndex(v int) *SymTreeView { t.ViewIndex = v; return t }

// SetWidgetSize sets the [SymTreeView.WidgetSize]
func (t *SymTreeView) SetWidgetSize(v math32.Vector2) *SymTreeView { t.WidgetSize = v; return t }

// SetRootView sets the [SymTreeView.RootView]
func (t *SymTreeView) SetRootView(v *views.TreeView) *SymTreeView { t.RootView = v; return t }

// SetSelectedNodes sets the [SymTreeView.SelectedNodes]
func (t *SymTreeView) SetSelectedNodes(v ...views.TreeViewer) *SymTreeView {
	t.SelectedNodes = v
	return t
}

// TextEditorType is the [types.Type] for [TextEditor]
var TextEditorType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/code.TextEditor", IDName: "text-editor", Doc: "TextEditor is the Code-specific version of the TextEditor, with support for\nsetting / clearing breakpoints, etc", Embeds: []types.Field{{Name: "Editor"}}, Fields: []types.Field{{Name: "Code"}}, Instance: &TextEditor{}})

// NewTextEditor adds a new [TextEditor] with the given name to the given parent:
// TextEditor is the Code-specific version of the TextEditor, with support for
// setting / clearing breakpoints, etc
func NewTextEditor(parent tree.Node, name ...string) *TextEditor {
	return parent.NewChild(TextEditorType, name...).(*TextEditor)
}

// NodeType returns the [*types.Type] of [TextEditor]
func (t *TextEditor) NodeType() *types.Type { return TextEditorType }

// New returns a new [*TextEditor] value
func (t *TextEditor) New() tree.Node { return &TextEditor{} }

// SetCode sets the [TextEditor.Code]
func (t *TextEditor) SetCode(v Code) *TextEditor { t.Code = v; return t }

// SetTooltip sets the [TextEditor.Tooltip]
func (t *TextEditor) SetTooltip(v string) *TextEditor { t.Tooltip = v; return t }

// SetCursorWidth sets the [TextEditor.CursorWidth]
func (t *TextEditor) SetCursorWidth(v units.Value) *TextEditor { t.CursorWidth = v; return t }

// SetLineNumberColor sets the [TextEditor.LineNumberColor]
func (t *TextEditor) SetLineNumberColor(v image.Image) *TextEditor { t.LineNumberColor = v; return t }

// SetSelectColor sets the [TextEditor.SelectColor]
func (t *TextEditor) SetSelectColor(v image.Image) *TextEditor { t.SelectColor = v; return t }

// SetHighlightColor sets the [TextEditor.HighlightColor]
func (t *TextEditor) SetHighlightColor(v image.Image) *TextEditor { t.HighlightColor = v; return t }

// SetCursorColor sets the [TextEditor.CursorColor]
func (t *TextEditor) SetCursorColor(v image.Image) *TextEditor { t.CursorColor = v; return t }

// SetLinkHandler sets the [TextEditor.LinkHandler]
func (t *TextEditor) SetLinkHandler(v func(tl *paint.TextLink)) *TextEditor {
	t.LinkHandler = v
	return t
}
