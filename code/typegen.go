// Code generated by "core generate"; DO NOT EDIT.

package code

import (
	"regexp"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.Code", IDName: "code", Doc: "Code is the core editor and tab viewer widget for the Code system. The\ndefault view has a tree browser of files on the left, editor panels in the\nmiddle, and a tabbed viewer on the right.", Methods: []types.Method{{Name: "UpdateFiles", Doc: "UpdateFiles updates the list of files saved in project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "OpenRecent", Doc: "OpenRecent opens a recently used file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}}, {Name: "OpenFile", Doc: "OpenFile opens file in an open project if it has the same path as the file\nor in a new window.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnm"}}, {Name: "OpenPath", Doc: "OpenPath creates a new project by opening given path, which can either be a\nspecific file or a folder containing multiple files of interest -- opens in\ncurrent Code object if it is empty, or otherwise opens a new window.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"path"}, Returns: []string{"Code"}}, {Name: "OpenProject", Doc: "OpenProject opens .code project file and its settings from given filename,\nin a standard toml-formatted file.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}, Returns: []string{"Code"}}, {Name: "NewProject", Doc: "NewProject creates a new project at given path, making a new folder in that\npath -- all Code projects are essentially defined by a path to a folder\ncontaining files.  If the folder already exists, then use OpenPath.\nCan also specify main language and version control type.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"path", "folder", "mainLang", "versionControl"}, Returns: []string{"Code"}}, {Name: "NewFile", Doc: "NewFile creates a new file in the project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename", "addToVcs"}}, {Name: "SaveProject", Doc: "SaveProject saves project file containing custom project settings, in a\nstandard toml-formatted file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SaveProjectAs", Doc: "SaveProjectAs saves project custom settings to given filename, in a standard\ntoml-formatted file\nsaveAllFiles indicates if user should be prompted for saving all files\nreturns true if the user was prompted, false otherwise", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}, Returns: []string{"bool"}}, {Name: "ExecCmdNameActive", Doc: "ExecCmdNameActive calls given command on current active texteditor", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"cmdName"}}, {Name: "ExecCmd", Doc: "ExecCmd pops up a menu to select a command appropriate for the current\nactive text view, and shows output in Tab with name of command", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "RunBuild", Doc: "RunBuild runs the BuildCmds set for this project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Run", Doc: "Run runs the RunCmds set for this project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Commit", Doc: "Commit commits the current changes using relevant VCS tool.\nChecks for VCS setting and for unsaved files.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CursorToHistPrev", Doc: "CursorToHistPrev moves back to the previous history item.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"bool"}}, {Name: "CursorToHistNext", Doc: "CursorToHistNext moves forward to the next history item.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"bool"}}, {Name: "ReplaceInActive", Doc: "ReplaceInActive does query-replace in active file only", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CutRect", Doc: "CutRect cuts rectangle in active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CopyRect", Doc: "CopyRect copies rectangle in active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "PasteRect", Doc: "PasteRect cuts rectangle in active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "RegisterCopy", Doc: "RegisterCopy saves current selection in active text view\nto register of given name returns true if saved.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"regNm"}}, {Name: "RegisterPaste", Doc: "RegisterPaste prompts user for available registers,\nand pastes selected one into active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"ctx"}}, {Name: "CommentOut", Doc: "CommentOut comments-out selected lines in active text view\nand uncomments if already commented\nIf multiple lines are selected and any line is uncommented all will be commented", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"bool"}}, {Name: "Indent", Doc: "Indent indents selected lines in active view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"bool"}}, {Name: "ReCase", Doc: "ReCase replaces currently selected text in current active view with given case", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"c"}, Returns: []string{"string"}}, {Name: "JoinParaLines", Doc: "JoinParaLines merges sequences of lines with hard returns forming paragraphs,\nseparated by blank lines, into a single line per paragraph,\nfor given selected region (full text if no selection)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "TabsToSpaces", Doc: "TabsToSpaces converts tabs to spaces\nfor given selected region (full text if no selection)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SpacesToTabs", Doc: "SpacesToTabs converts spaces to tabs\nfor given selected region (full text if no selection)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "DiffFiles", Doc: "DiffFiles shows the differences between two given files\nin side-by-side DiffEditor and in the console as a context diff.\nIt opens the files as file nodes and uses existing contents if open already.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnmA", "fnmB"}}, {Name: "DiffFileLines", Doc: "DiffFileLines shows the differences between given file node as the A file,\nand another given file as the B file,\nin side-by-side DiffEditor and in the console as a context diff.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"lna", "lnmB"}}, {Name: "CountWords", Doc: "CountWords counts number of words (and lines) in active file\nreturns a string report thereof.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"string"}}, {Name: "CountWordsRegion", Doc: "CountWordsRegion counts number of words (and lines) in selected region in file\nif no selection, returns numbers for entire file.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"string"}}, {Name: "SaveActiveView", Doc: "SaveActiveView saves the contents of the currently active texteditor.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SaveActiveViewAs", Doc: "SaveActiveViewAs save with specified filename the contents of the\ncurrently active texteditor", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}}, {Name: "RevertActiveView", Doc: "RevertActiveView revert active view to saved version.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "CloseActiveView", Doc: "CloseActiveView closes the buffer associated with active view.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "ViewFile", Doc: "ViewFile views file in an existing TextEditor if it is already viewing that\nfile, otherwise opens ViewLines in active buffer.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnm"}, Returns: []string{"TextEditor", "int", "bool"}}, {Name: "NextViewFile", Doc: "NextViewFile sets the next text view to view given file name.\nWill use a more robust search of file tree if file path is not\ndirectly openable. Returns texteditor and its index, false if not found.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"fnm"}, Returns: []string{"TextEditor", "int", "bool"}}, {Name: "CloneActiveView", Doc: "CloneActiveView sets the next text view to view the same file currently being vieweds\nin the active view. returns text view and index", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"TextEditor", "int"}}, {Name: "SaveAll", Doc: "SaveAll saves all of the open filenodes to their current file names\nand saves the project state if it has been saved before (i.e., the .code file exists)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "FocusNextPanel", Doc: "FocusNextPanel moves the keyboard focus to the next panel to the right", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "FocusPrevPanel", Doc: "FocusPrevPanel moves the keyboard focus to the previous panel to the left", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditProjectSettings", Doc: "EditProjectSettings allows editing of project settings (settings specific to this project)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SplitsSetView", Doc: "SplitsSetView sets split view splitters to given named setting", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"split"}}, {Name: "SplitsSave", Doc: "SplitsSave saves current splitter settings to named splitter settings under\nexisting name, and saves to prefs file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"split"}}, {Name: "SplitsSaveAs", Doc: "SplitsSaveAs saves current splitter settings to new named splitter settings, and\nsaves to prefs file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"name", "desc"}}, {Name: "SplitsEdit", Doc: "SplitsEdit opens the SplitsView editor to customize saved splitter settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Find", Doc: "Find does Find / Replace in files, using given options and filters -- opens up a\nmain tab with the results and further controls.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"find", "repl", "ignoreCase", "regExp", "loc", "langs"}}, {Name: "Spell", Doc: "Spell checks spelling in active text view", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Symbols", Doc: "Symbols displays the Symbols of a file or package", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Debug", Doc: "Debug starts the debugger on the RunExec executable.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "DebugTest", Doc: "DebugTest runs the debugger using testing mode in current active texteditor path.\ntestName specifies which test(s) to run according to the standard go test -run\nspecification.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"testName"}}, {Name: "DebugAttach", Doc: "DebugAttach runs the debugger by attaching to an already-running process.\npid is the process id to attach to.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"pid"}}, {Name: "VCSUpdateAll", Doc: "VCSUpdateAll does an Update (e.g., Pull) on all VCS repositories within\nthe open tree nodes in FileTree.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "VCSLog", Doc: "VCSLog shows the VCS log of commits in this project,\nin an interactive browser from which any revisions can be\ncompared and diffs browsed.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"Log", "error"}}, {Name: "OpenConsoleTab", Doc: "OpenConsoleTab opens a main tab displaying console output (stdout, stderr)", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "ChooseRunExec", Doc: "ChooseRunExec selects the executable to run for the project", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"exePath"}}, {Name: "HelpWiki", Doc: "HelpWiki opens wiki page for code on github", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}}, Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "ProjectRoot", Doc: "root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjectFilename"}, {Name: "ProjectFilename", Doc: "current project filename for saving / loading specific Code configuration information in a .code file (optional)"}, {Name: "ActiveFilename", Doc: "filename of the currently active texteditor"}, {Name: "ActiveLang", Doc: "language for current active filename"}, {Name: "ActiveVCS", Doc: "VCS repo for current active filename"}, {Name: "ActiveVCSInfo", Doc: "VCS info for current active filename (typically branch or revision) -- for status"}, {Name: "Changed", Doc: "has the root changed?  we receive update signals from root for changes"}, {Name: "StatusMessage", Doc: "the last status update message"}, {Name: "LastSaveTStamp", Doc: "timestamp for when a file was last saved -- provides dirty state for various updates including rebuilding in debugger"}, {Name: "Files", Doc: "all the files in the project directory and subdirectories"}, {Name: "ActiveTextEditorIndex", Doc: "index of the currently active texteditor -- new files will be viewed in other views if available"}, {Name: "OpenFiles", Doc: "list of open files, most recent first"}, {Name: "CmdBufs", Doc: "the command buffers for commands run in this project"}, {Name: "CmdHistory", Doc: "history of commands executed in this session"}, {Name: "RunningCmds", Doc: "currently running commands in this project"}, {Name: "ArgVals", Doc: "current arg var vals"}, {Name: "Settings", Doc: "settings for this project -- this is what is saved in a .code project file"}, {Name: "CurDbg", Doc: "current debug view"}, {Name: "KeySeq1", Doc: "first key in sequence if needs2 key pressed"}, {Name: "UpdateMu", Doc: "mutex for protecting overall updates to Code"}}})

// NewCode returns a new [Code] with the given optional parent:
// Code is the core editor and tab viewer widget for the Code system. The
// default view has a tree browser of files on the left, editor panels in the
// middle, and a tabbed viewer on the right.
func NewCode(parent ...tree.Node) *Code { return tree.New[Code](parent...) }

// SetProjectRoot sets the [Code.ProjectRoot]:
// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjectFilename
func (t *Code) SetProjectRoot(v core.Filename) *Code { t.ProjectRoot = v; return t }

// SetProjectFilename sets the [Code.ProjectFilename]:
// current project filename for saving / loading specific Code configuration information in a .code file (optional)
func (t *Code) SetProjectFilename(v core.Filename) *Code { t.ProjectFilename = v; return t }

// SetActiveLang sets the [Code.ActiveLang]:
// language for current active filename
func (t *Code) SetActiveLang(v fileinfo.Known) *Code { t.ActiveLang = v; return t }

// SetStatusMessage sets the [Code.StatusMessage]:
// the last status update message
func (t *Code) SetStatusMessage(v string) *Code { t.StatusMessage = v; return t }

// SetOpenFiles sets the [Code.OpenFiles]:
// list of open files, most recent first
func (t *Code) SetOpenFiles(v OpenFiles) *Code { t.OpenFiles = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.DebugPanel", IDName: "debug-panel", Doc: "DebugPanel is the debugger panel.", Methods: []types.Method{{Name: "StepOver", Doc: "StepOver continues to the next source line, not entering function calls.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "StepInto", Doc: "StepInto continues to the next source line, entering function calls.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "StepOut", Doc: "StepOut continues to the return point of the current function.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SingleStep", Doc: "StepSingle steps a single CPU instruction.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Stop", Doc: "Stop stops a running process.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "ListGlobalVars", Doc: "ListGlobalVars lists global vars matching the given optional filter.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filter"}}}, Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Known", Doc: "known file type to determine debugger"}, {Name: "ExePath", Doc: "path to executable / dir to debug"}, {Name: "DbgTime", Doc: "time when dbg was last restarted"}, {Name: "Dbg", Doc: "the debugger"}, {Name: "State", Doc: "all relevant debug state info"}, {Name: "CurFileLoc", Doc: "current ShowFile location -- cleared before next one or run"}, {Name: "BBreaks", Doc: "backup breakpoints list -- to track deletes"}, {Name: "OutputBuffer", Doc: "output from the debugger"}, {Name: "Code", Doc: "parent code project"}}})

// NewDebugPanel returns a new [DebugPanel] with the given optional parent:
// DebugPanel is the debugger panel.
func NewDebugPanel(parent ...tree.Node) *DebugPanel { return tree.New[DebugPanel](parent...) }

// SetKnown sets the [DebugPanel.Known]:
// known file type to determine debugger
func (t *DebugPanel) SetKnown(v fileinfo.Known) *DebugPanel { t.Known = v; return t }

// SetExePath sets the [DebugPanel.ExePath]:
// path to executable / dir to debug
func (t *DebugPanel) SetExePath(v string) *DebugPanel { t.ExePath = v; return t }

// SetDbgTime sets the [DebugPanel.DbgTime]:
// time when dbg was last restarted
func (t *DebugPanel) SetDbgTime(v time.Time) *DebugPanel { t.DbgTime = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.VarView", IDName: "var-view", Doc: "VarView shows a debug variable in an inspector-like framework,\nwith sub-variables in a tree.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Var", Doc: "variable being edited"}, {Name: "SelectVar"}, {Name: "FrameInfo", Doc: "frame info"}, {Name: "DbgView", Doc: "parent DebugPanel"}}})

// NewVarView returns a new [VarView] with the given optional parent:
// VarView shows a debug variable in an inspector-like framework,
// with sub-variables in a tree.
func NewVarView(parent ...tree.Node) *VarView { return tree.New[VarView](parent...) }

// SetDbgView sets the [VarView.DbgView]:
// parent DebugPanel
func (t *VarView) SetDbgView(v *DebugPanel) *VarView { t.DbgView = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.FileNode", IDName: "file-node", Doc: "FileNode is Code version of FileNode for FileTree", Methods: []types.Method{{Name: "ExecCmdFile", Doc: "ExecCmdFile pops up a menu to select a command appropriate for the given node,\nand shows output in MainTab with name of command", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditFiles", Doc: "EditFiles calls EditFile on selected files", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "SetRunExecs", Doc: "SetRunExecs sets executable as the RunExec executable that will be run with Run / Debug buttons", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}}, Embeds: []types.Field{{Name: "Node"}}})

// NewFileNode returns a new [FileNode] with the given optional parent:
// FileNode is Code version of FileNode for FileTree
func NewFileNode(parent ...tree.Node) *FileNode { return tree.New[FileNode](parent...) }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.FindPanel", IDName: "find-panel", Doc: "FindPanel is a find / replace widget that displays results in a [TextEditor]\nand has a toolbar for controlling find / replace process.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Code", Doc: "parent code project"}, {Name: "Time", Doc: "time of last find"}, {Name: "Re", Doc: "compiled regexp"}}})

// NewFindPanel returns a new [FindPanel] with the given optional parent:
// FindPanel is a find / replace widget that displays results in a [TextEditor]
// and has a toolbar for controlling find / replace process.
func NewFindPanel(parent ...tree.Node) *FindPanel { return tree.New[FindPanel](parent...) }

// SetCode sets the [FindPanel.Code]:
// parent code project
func (t *FindPanel) SetCode(v *Code) *FindPanel { t.Code = v; return t }

// SetTime sets the [FindPanel.Time]:
// time of last find
func (t *FindPanel) SetTime(v time.Time) *FindPanel { t.Time = v; return t }

// SetRe sets the [FindPanel.Re]:
// compiled regexp
func (t *FindPanel) SetRe(v *regexp.Regexp) *FindPanel { t.Re = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.PreviewPanel", IDName: "preview-panel", Doc: "PreviewPanel is a widget that displays an interactive live preview of a\nMD, HTML, or SVG file currently open.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "code", Doc: "code is the parent [Code]."}, {Name: "lastRendered", Doc: "lastRendered is the content that was last rendered in the preview."}}})

// NewPreviewPanel returns a new [PreviewPanel] with the given optional parent:
// PreviewPanel is a widget that displays an interactive live preview of a
// MD, HTML, or SVG file currently open.
func NewPreviewPanel(parent ...tree.Node) *PreviewPanel { return tree.New[PreviewPanel](parent...) }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.SettingsData", IDName: "settings-data", Doc: "SettingsData is the data type for the overall user settings for Code.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Methods: []types.Method{{Name: "Apply", Doc: "Apply settings updates things according with settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditLangOpts", Doc: "EditLangOpts opens the LangsView editor to customize options for each type of\nlanguage / data / file type.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditCmds", Doc: "EditCmds opens the CmdsView editor to customize commands you can run.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditSplits", Doc: "EditSplits opens the SplitsView editor to customize saved splitter settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "EditRegisters", Doc: "EditRegisters opens the RegistersView editor to customize saved registers", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}}, Embeds: []types.Field{{Name: "SettingsBase"}}, Fields: []types.Field{{Name: "Files", Doc: "file picker settings"}, {Name: "SaveLangOpts", Doc: "if set, the current customized set of language options (see Edit Lang Opts) is saved / loaded along with other settings -- if not set, then you always are using the default compiled-in standard set (which will be updated)"}, {Name: "SaveCmds", Doc: "if set, the current customized set of command parameters (see Edit Cmds) is saved / loaded along with other settings -- if not set, then you always are using the default compiled-in standard set (which will be updated)"}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.FileSettings", IDName: "file-settings", Doc: "FileSettings contains file picker settings", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "DirsOnTop", Doc: "if true, then all directories are placed at the top of the tree -- otherwise everything is alpha sorted"}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.ProjectSettings", IDName: "project-settings", Doc: "ProjectSettings are the settings for saving for a project. This IS the project file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Methods: []types.Method{{Name: "Open", Doc: "Open open from file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}, Returns: []string{"error"}}, {Name: "Save", Doc: "Save save to file", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"filename"}, Returns: []string{"error"}}}, Fields: []types.Field{{Name: "Files", Doc: "file picker settings"}, {Name: "Editor", Doc: "editor settings"}, {Name: "SplitName", Doc: "current named-split config in use for configuring the splitters"}, {Name: "MainLang", Doc: "the language associated with the most frequently encountered file\nextension in the file tree -- can be manually set here as well"}, {Name: "VersionControl", Doc: "the type of version control system used in this project (git, svn, etc).\nfilters commands available"}, {Name: "ProjectFilename", Doc: "current project filename for saving / loading specific Code\nconfiguration information in a .code file (optional)"}, {Name: "ProjectRoot", Doc: "root directory for the project. all projects must be organized within\na top-level root directory, with all the files therein constituting\nthe scope of the project. By default it is the path for ProjectFilename"}, {Name: "GoMod", Doc: "if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically"}, {Name: "BuildCmds", Doc: "command(s) to run for main Build button"}, {Name: "BuildDir", Doc: "build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands"}, {Name: "BuildTarg", Doc: "build target for main Build button, if relevant for your  BuildCmds"}, {Name: "RunExec", Doc: "executable to run for this project via main Run button -- called by standard Run Project command"}, {Name: "RunCmds", Doc: "command(s) to run for main Run button (typically Run Project)"}, {Name: "Debug", Doc: "custom debugger parameters for this project"}, {Name: "Find", Doc: "saved find params"}, {Name: "Symbols", Doc: "saved structure params"}, {Name: "Dirs", Doc: "directory properties"}, {Name: "Register", Doc: "last register used"}, {Name: "Splits", Doc: "current splitter splits"}, {Name: "TabsUnder", Doc: "current tabUnder setting for splits"}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.SpellPanel", IDName: "spell-panel", Doc: "SpellPanel is a widget that displays results of a spell check.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Code", Doc: "parent code project"}, {Name: "Text", Doc: "texteditor that we're spell-checking"}, {Name: "Errs", Doc: "current spelling errors"}, {Name: "CurLn", Doc: "current line in text we're on"}, {Name: "CurIndex", Doc: "current index in Errs we're on"}, {Name: "UnkLex", Doc: "current unknown lex token"}, {Name: "UnkWord", Doc: "current unknown word"}, {Name: "Suggest", Doc: "a list of suggestions from spell checker"}, {Name: "LastAction", Doc: "last user action (ignore, change, learn)"}}})

// NewSpellPanel returns a new [SpellPanel] with the given optional parent:
// SpellPanel is a widget that displays results of a spell check.
func NewSpellPanel(parent ...tree.Node) *SpellPanel { return tree.New[SpellPanel](parent...) }

// SetCode sets the [SpellPanel.Code]:
// parent code project
func (t *SpellPanel) SetCode(v *Code) *SpellPanel { t.Code = v; return t }

// SetText sets the [SpellPanel.Text]:
// texteditor that we're spell-checking
func (t *SpellPanel) SetText(v *TextEditor) *SpellPanel { t.Text = v; return t }

// SetErrs sets the [SpellPanel.Errs]:
// current spelling errors
func (t *SpellPanel) SetErrs(v lexer.Line) *SpellPanel { t.Errs = v; return t }

// SetCurLn sets the [SpellPanel.CurLn]:
// current line in text we're on
func (t *SpellPanel) SetCurLn(v int) *SpellPanel { t.CurLn = v; return t }

// SetCurIndex sets the [SpellPanel.CurIndex]:
// current index in Errs we're on
func (t *SpellPanel) SetCurIndex(v int) *SpellPanel { t.CurIndex = v; return t }

// SetUnkLex sets the [SpellPanel.UnkLex]:
// current unknown lex token
func (t *SpellPanel) SetUnkLex(v lexer.Lex) *SpellPanel { t.UnkLex = v; return t }

// SetUnkWord sets the [SpellPanel.UnkWord]:
// current unknown word
func (t *SpellPanel) SetUnkWord(v string) *SpellPanel { t.UnkWord = v; return t }

// SetSuggest sets the [SpellPanel.Suggest]:
// a list of suggestions from spell checker
func (t *SpellPanel) SetSuggest(v ...string) *SpellPanel { t.Suggest = v; return t }

// SetLastAction sets the [SpellPanel.LastAction]:
// last user action (ignore, change, learn)
func (t *SpellPanel) SetLastAction(v *core.Button) *SpellPanel { t.LastAction = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.SymbolsPanel", IDName: "symbols-panel", Doc: "SymbolsPanel is a widget that displays results of a file or package parse of symbols.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Code", Doc: "parent code project"}, {Name: "SymParams", Doc: "params for structure display"}, {Name: "Syms", Doc: "all the symbols for the file or package in a tree"}, {Name: "Match", Doc: "only show symbols that match this string"}}})

// NewSymbolsPanel returns a new [SymbolsPanel] with the given optional parent:
// SymbolsPanel is a widget that displays results of a file or package parse of symbols.
func NewSymbolsPanel(parent ...tree.Node) *SymbolsPanel { return tree.New[SymbolsPanel](parent...) }

// SetCode sets the [SymbolsPanel.Code]:
// parent code project
func (t *SymbolsPanel) SetCode(v *Code) *SymbolsPanel { t.Code = v; return t }

// SetSymParams sets the [SymbolsPanel.SymParams]:
// params for structure display
func (t *SymbolsPanel) SetSymParams(v SymbolsParams) *SymbolsPanel { t.SymParams = v; return t }

// SetSyms sets the [SymbolsPanel.Syms]:
// all the symbols for the file or package in a tree
func (t *SymbolsPanel) SetSyms(v *SymNode) *SymbolsPanel { t.Syms = v; return t }

// SetMatch sets the [SymbolsPanel.Match]:
// only show symbols that match this string
func (t *SymbolsPanel) SetMatch(v string) *SymbolsPanel { t.Match = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.SymNode", IDName: "sym-node", Doc: "SymNode represents a language symbol -- the name of the node is\nthe name of the symbol. Some symbols, e.g. type have children", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Symbol", Doc: "the symbol"}}})

// NewSymNode returns a new [SymNode] with the given optional parent:
// SymNode represents a language symbol -- the name of the node is
// the name of the symbol. Some symbols, e.g. type have children
func NewSymNode(parent ...tree.Node) *SymNode { return tree.New[SymNode](parent...) }

// SetSymbol sets the [SymNode.Symbol]:
// the symbol
func (t *SymNode) SetSymbol(v syms.Symbol) *SymNode { t.Symbol = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.SymTree", IDName: "sym-tree", Doc: "SymTree is a Tree that knows how to operate on FileNode nodes", Embeds: []types.Field{{Name: "Tree"}}})

// NewSymTree returns a new [SymTree] with the given optional parent:
// SymTree is a Tree that knows how to operate on FileNode nodes
func NewSymTree(parent ...tree.Node) *SymTree { return tree.New[SymTree](parent...) }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.TextEditor", IDName: "text-editor", Doc: "TextEditor is the Code-specific version of the TextEditor, with support for\nsetting / clearing breakpoints, etc", Embeds: []types.Field{{Name: "Editor"}}, Fields: []types.Field{{Name: "Code"}}})

// NewTextEditor returns a new [TextEditor] with the given optional parent:
// TextEditor is the Code-specific version of the TextEditor, with support for
// setting / clearing breakpoints, etc
func NewTextEditor(parent ...tree.Node) *TextEditor { return tree.New[TextEditor](parent...) }

// SetCode sets the [TextEditor.Code]
func (t *TextEditor) SetCode(v *Code) *TextEditor { t.Code = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code.CmdButton", IDName: "cmd-button", Doc: "CmdButton represents a [CmdName] value with a button that opens a [CmdView].", Embeds: []types.Field{{Name: "Button"}}})

// NewCmdButton returns a new [CmdButton] with the given optional parent:
// CmdButton represents a [CmdName] value with a button that opens a [CmdView].
func NewCmdButton(parent ...tree.Node) *CmdButton { return tree.New[CmdButton](parent...) }
