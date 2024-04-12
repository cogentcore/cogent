// Code generated by "core generate"; DO NOT EDIT.

package codev

import (
	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/core"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// CodeViewType is the [types.Type] for [CodeView]
var CodeViewType = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/codev.CodeView", IDName: "code-view", Doc: "CodeView is the core editor and tab viewer framework for the Code system.  The\ndefault view has a tree browser of files on the left, editor panels in the\nmiddle, and a tabbed viewer on the right.", Methods: []types.Method{{Name: "ConfigToolbar", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"tb"}}, {Name: "UpdateFiles", Doc: "UpdateFiles updates the list of files saved in project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "OpenRecent", Doc: "OpenRecent opens a recently-used file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}}, {Name: "OpenFile", Doc: "OpenFile opens file in an open project if it has the same path as the file\nor in a new window.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnm"}}, {Name: "OpenPath", Doc: "OpenPath creates a new project by opening given path, which can either be a\nspecific file or a folder containing multiple files of interest -- opens in\ncurrent CodeView object if it is empty, or otherwise opens a new window.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"path"}, Returns: []string{"CodeView"}}, {Name: "OpenProj", Doc: "OpenProj opens .code project file and its settings from given filename, in a standard\ntoml-formatted file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}, Returns: []string{"CodeView"}}, {Name: "NewProj", Doc: "NewProj creates a new project at given path, making a new folder in that\npath -- all CodeView projects are essentially defined by a path to a folder\ncontaining files.  If the folder already exists, then use OpenPath.\nCan also specify main language and version control type", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"path", "folder", "mainLang", "VersionControl"}, Returns: []string{"CodeView"}}, {Name: "NewFile", Doc: "NewFile creates a new file in the project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename", "addToVcs"}}, {Name: "SaveProj", Doc: "SaveProj saves project file containing custom project settings, in a\nstandard toml-formatted file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SaveProjAs", Doc: "SaveProjAs saves project custom settings to given filename, in a standard\ntoml-formatted file\nsaveAllFiles indicates if user should be prompted for saving all files\nreturns true if the user was prompted, false otherwise", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}, Returns: []string{"bool"}}, {Name: "ExecCmdNameActive", Doc: "ExecCmdNameActive calls given command on current active texteditor", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"cmdNm"}}, {Name: "ExecCmd", Doc: "ExecCmd pops up a menu to select a command appropriate for the current\nactive text view, and shows output in Tab with name of command", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Build", Doc: "Build runs the BuildCmds set for this project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Run", Doc: "Run runs the RunCmds set for this project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Commit", Doc: "Commit commits the current changes using relevant VCS tool.\nChecks for VCS setting and for unsaved files.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CursorToHistPrev", Doc: "CursorToHistPrev moves back to the previous history item.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"bool"}}, {Name: "CursorToHistNext", Doc: "CursorToHistNext moves forward to the next history item.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"bool"}}, {Name: "ReplaceInActive", Doc: "ReplaceInActive does query-replace in active file only", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CutRect", Doc: "CutRect cuts rectangle in active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CopyRect", Doc: "CopyRect copies rectangle in active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "PasteRect", Doc: "PasteRect cuts rectangle in active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "RegisterCopy", Doc: "RegisterCopy saves current selection in active text view to register of given name\nreturns true if saved", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"name"}, Returns: []string{"bool"}}, {Name: "RegisterPaste", Doc: "RegisterPaste pastes register of given name into active text view\nreturns true if pasted", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"name"}, Returns: []string{"bool"}}, {Name: "CommentOut", Doc: "CommentOut comments-out selected lines in active text view\nand uncomments if already commented\nIf multiple lines are selected and any line is uncommented all will be commented", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"bool"}}, {Name: "Indent", Doc: "Indent indents selected lines in active view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"bool"}}, {Name: "ReCase", Doc: "ReCase replaces currently selected text in current active view with given case", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"c"}, Returns: []string{"string"}}, {Name: "JoinParaLines", Doc: "JoinParaLines merges sequences of lines with hard returns forming paragraphs,\nseparated by blank lines, into a single line per paragraph,\nfor given selected region (full text if no selection)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "TabsToSpaces", Doc: "TabsToSpaces converts tabs to spaces\nfor given selected region (full text if no selection)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SpacesToTabs", Doc: "SpacesToTabs converts spaces to tabs\nfor given selected region (full text if no selection)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "DiffFiles", Doc: "DiffFiles shows the differences between two given files\nin side-by-side DiffView and in the console as a context diff.\nIt opens the files as file nodes and uses existing contents if open already.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnmA", "fnmB"}}, {Name: "DiffFileNode", Doc: "DiffFileNode shows the differences between given file node as the A file,\nand another given file as the B file,\nin side-by-side DiffView and in the console as a context diff.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fna", "fnmB"}}, {Name: "CountWords", Doc: "CountWords counts number of words (and lines) in active file\nreturns a string report thereof.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"string"}}, {Name: "CountWordsRegion", Doc: "CountWordsRegion counts number of words (and lines) in selected region in file\nif no selection, returns numbers for entire file.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"string"}}, {Name: "SaveActiveView", Doc: "SaveActiveView saves the contents of the currently-active texteditor", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SaveActiveViewAs", Doc: "SaveActiveViewAs save with specified filename the contents of the\ncurrently-active texteditor", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}}, {Name: "RevertActiveView", Doc: "RevertActiveView revert active view to saved version", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CloseActiveView", Doc: "CloseActiveView closes the buffer associated with active view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "NextViewFile", Doc: "NextViewFile sets the next text view to view given file name -- include as\nmuch of name as possible to disambiguate -- will use the first matching --\nif already being viewed, that is activated -- returns texteditor and its\nindex, false if not found", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnm"}, Returns: []string{"TextEditor", "int", "bool"}}, {Name: "ViewFile", Doc: "ViewFile views file in an existing TextEditor if it is already viewing that\nfile, otherwise opens ViewFileNode in active buffer", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnm"}, Returns: []string{"TextEditor", "int", "bool"}}, {Name: "CloneActiveView", Doc: "CloneActiveView sets the next text view to view the same file currently being vieweds\nin the active view. returns text view and index", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"TextEditor", "int"}}, {Name: "SaveAll", Doc: "SaveAll saves all of the open filenodes to their current file names\nand saves the project state if it has been saved before (i.e., the .code file exists)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "FocusNextPanel", Doc: "FocusNextPanel moves the keyboard focus to the next panel to the right", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "FocusPrevPanel", Doc: "FocusPrevPanel moves the keyboard focus to the previous panel to the left", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditProjSettings", Doc: "EditProjSettings allows editing of project settings (settings specific to this project)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SplitsSetView", Doc: "SplitsSetView sets split view splitters to given named setting", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"split"}}, {Name: "SplitsSave", Doc: "SplitsSave saves current splitter settings to named splitter settings under\nexisting name, and saves to prefs file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"split"}}, {Name: "SplitsSaveAs", Doc: "SplitsSaveAs saves current splitter settings to new named splitter settings, and\nsaves to prefs file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"name", "desc"}}, {Name: "SplitsEdit", Doc: "SplitsEdit opens the SplitsView editor to customize saved splitter settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Find", Doc: "Find does Find / Replace in files, using given options and filters -- opens up a\nmain tab with the results and further controls.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"find", "repl", "ignoreCase", "regExp", "loc", "langs"}}, {Name: "Spell", Doc: "Spell checks spelling in active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Symbols", Doc: "Symbols displays the Symbols of a file or package", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Debug", Doc: "Debug starts the debugger on the RunExec executable.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "DebugTest", Doc: "DebugTest runs the debugger using testing mode in current active texteditor path", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "DebugAttach", Doc: "DebugAttach runs the debugger by attaching to an already-running process.\npid is the process id to attach to.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"pid"}}, {Name: "VCSUpdateAll", Doc: "VCSUpdateAll does an Update (e.g., Pull) on all VCS repositories within\nthe open tree nodes in FileTree.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "VCSLog", Doc: "VCSLog shows the VCS log of commits for this file, optionally with a\nsince date qualifier: If since is non-empty, it should be\na date-like expression that the VCS will understand, such as\n1/1/2020, yesterday, last year, etc.  SVN only understands a\nnumber as a maximum number of items to return.\nIf allFiles is true, then the log will show revisions for all files, not just\nthis one.\nReturns the Log and also shows it in a VCSLogView which supports further actions.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"since"}, Returns: []string{"Log", "error"}}, {Name: "OpenConsoleTab", Doc: "OpenConsoleTab opens a main tab displaying console output (stdout, stderr)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "ChooseRunExec", Doc: "ChooseRunExec selects the executable to run for the project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"exePath"}}, {Name: "HelpWiki", Doc: "HelpWiki opens wiki page for code on github", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}}, Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "ProjRoot", Doc: "root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"}, {Name: "ProjFilename", Doc: "current project filename for saving / loading specific Code configuration information in a .code file (optional)"}, {Name: "ActiveFilename", Doc: "filename of the currently-active texteditor"}, {Name: "ActiveLang", Doc: "language for current active filename"}, {Name: "ActiveVCS", Doc: "VCS repo for current active filename"}, {Name: "ActiveVCSInfo", Doc: "VCS info for current active filename (typically branch or revision) -- for status"}, {Name: "Changed", Doc: "has the root changed?  we receive update signals from root for changes"}, {Name: "StatusMessage", Doc: "the last status update message"}, {Name: "LastSaveTStamp", Doc: "timestamp for when a file was last saved -- provides dirty state for various updates including rebuilding in debugger"}, {Name: "Files", Doc: "all the files in the project directory and subdirectories"}, {Name: "ActiveTextEditorIndex", Doc: "index of the currently-active texteditor -- new files will be viewed in other views if available"}, {Name: "OpenNodes", Doc: "list of open nodes, most recent first"}, {Name: "CmdBufs", Doc: "the command buffers for commands run in this project"}, {Name: "CmdHistory", Doc: "history of commands executed in this session"}, {Name: "RunningCmds", Doc: "currently running commands in this project"}, {Name: "ArgVals", Doc: "current arg var vals"}, {Name: "Settings", Doc: "settings for this project -- this is what is saved in a .code project file"}, {Name: "CurDbg", Doc: "current debug view"}, {Name: "KeySeq1", Doc: "first key in sequence if needs2 key pressed"}, {Name: "UpdateMu", Doc: "mutex for protecting overall updates to CodeView"}}, Instance: &CodeView{}})

// NewCodeView adds a new [CodeView] with the given name to the given parent:
// CodeView is the core editor and tab viewer framework for the Code system.  The
// default view has a tree browser of files on the left, editor panels in the
// middle, and a tabbed viewer on the right.
func NewCodeView(parent tree.Node, name ...string) *CodeView {
	return parent.NewChild(CodeViewType, name...).(*CodeView)
}

// NodeType returns the [*types.Type] of [CodeView]
func (t *CodeView) NodeType() *types.Type { return CodeViewType }

// New returns a new [*CodeView] value
func (t *CodeView) New() tree.Node { return &CodeView{} }

// SetProjRoot sets the [CodeView.ProjRoot]:
// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename
func (t *CodeView) SetProjRoot(v core.Filename) *CodeView { t.ProjRoot = v; return t }

// SetProjFilename sets the [CodeView.ProjFilename]:
// current project filename for saving / loading specific Code configuration information in a .code file (optional)
func (t *CodeView) SetProjFilename(v core.Filename) *CodeView { t.ProjFilename = v; return t }

// SetActiveLang sets the [CodeView.ActiveLang]:
// language for current active filename
func (t *CodeView) SetActiveLang(v fileinfo.Known) *CodeView { t.ActiveLang = v; return t }

// SetStatusMessage sets the [CodeView.StatusMessage]:
// the last status update message
func (t *CodeView) SetStatusMessage(v string) *CodeView { t.StatusMessage = v; return t }

// SetOpenNodes sets the [CodeView.OpenNodes]:
// list of open nodes, most recent first
func (t *CodeView) SetOpenNodes(v code.OpenNodes) *CodeView { t.OpenNodes = v; return t }

// SetTooltip sets the [CodeView.Tooltip]
func (t *CodeView) SetTooltip(v string) *CodeView { t.Tooltip = v; return t }