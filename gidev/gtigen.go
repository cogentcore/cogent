// Code generated by "goki generate"; DO NOT EDIT.

package gidev

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gide/v2/gide"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/ki/v2"
	"goki.dev/ordmap"
	"goki.dev/pi/v2/filecat"
)

// GideViewType is the [gti.Type] for [GideView]
var GideViewType = gti.AddType(&gti.Type{
	Name:       "goki.dev/gide/v2/gidev.GideView",
	ShortName:  "gidev.GideView",
	IDName:     "gide-view",
	Doc:        "GideView is the core editor and tab viewer framework for the Gide system.  The\ndefault view has a tree browser of files on the left, editor panels in the\nmiddle, and a tabbed viewer on the right.",
	Directives: gti.Directives{},
	Fields: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
		{"ProjRoot", &gti.Field{Name: "ProjRoot", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename", Directives: gti.Directives{}, Tag: ""}},
		{"ProjFilename", &gti.Field{Name: "ProjFilename", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "current project filename for saving / loading specific Gide configuration information in a .gide file (optional)", Directives: gti.Directives{}, Tag: "ext:\".gide\""}},
		{"ActiveFilename", &gti.Field{Name: "ActiveFilename", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "filename of the currently-active textview", Directives: gti.Directives{}, Tag: "set:\"-\""}},
		{"ActiveLang", &gti.Field{Name: "ActiveLang", Type: "goki.dev/pi/v2/filecat.Supported", LocalType: "filecat.Supported", Doc: "language for current active filename", Directives: gti.Directives{}, Tag: ""}},
		{"ActiveVCS", &gti.Field{Name: "ActiveVCS", Type: "goki.dev/vci/v2.Repo", LocalType: "vci.Repo", Doc: "VCS repo for current active filename", Directives: gti.Directives{}, Tag: "set:\"-\""}},
		{"ActiveVCSInfo", &gti.Field{Name: "ActiveVCSInfo", Type: "string", LocalType: "string", Doc: "VCS info for current active filename (typically branch or revision) -- for status", Directives: gti.Directives{}, Tag: "set:\"-\""}},
		{"Changed", &gti.Field{Name: "Changed", Type: "bool", LocalType: "bool", Doc: "has the root changed?  we receive update signals from root for changes", Directives: gti.Directives{}, Tag: "set:\"-\" json:\"-\""}},
		{"StatusMessage", &gti.Field{Name: "StatusMessage", Type: "string", LocalType: "string", Doc: "the last status update message", Directives: gti.Directives{}, Tag: ""}},
		{"LastSaveTStamp", &gti.Field{Name: "LastSaveTStamp", Type: "time.Time", LocalType: "time.Time", Doc: "timestamp for when a file was last saved -- provides dirty state for various updates including rebuilding in debugger", Directives: gti.Directives{}, Tag: "set:\"-\" json:\"-\""}},
		{"Files", &gti.Field{Name: "Files", Type: "*goki.dev/gi/v2/filetree.Tree", LocalType: "*filetree.Tree", Doc: "all the files in the project directory and subdirectories", Directives: gti.Directives{}, Tag: "set:\"-\" json:\"-\""}},
		{"ActiveTextEditorIdx", &gti.Field{Name: "ActiveTextEditorIdx", Type: "int", LocalType: "int", Doc: "index of the currently-active textview -- new files will be viewed in other views if available", Directives: gti.Directives{}, Tag: "set:\"-\" json:\"-\""}},
		{"OpenNodes", &gti.Field{Name: "OpenNodes", Type: "goki.dev/gide/v2/gide.OpenNodes", LocalType: "gide.OpenNodes", Doc: "list of open nodes, most recent first", Directives: gti.Directives{}, Tag: "json:\"-\""}},
		{"CmdBufs", &gti.Field{Name: "CmdBufs", Type: "map[string]*goki.dev/gi/v2/texteditor.Buf", LocalType: "map[string]*texteditor.Buf", Doc: "the command buffers for commands run in this project", Directives: gti.Directives{}, Tag: "set:\"-\" json:\"-\""}},
		{"CmdHistory", &gti.Field{Name: "CmdHistory", Type: "goki.dev/gide/v2/gide.CmdNames", LocalType: "gide.CmdNames", Doc: "history of commands executed in this session", Directives: gti.Directives{}, Tag: "set:\"-\" json:\"-\""}},
		{"RunningCmds", &gti.Field{Name: "RunningCmds", Type: "goki.dev/gide/v2/gide.CmdRuns", LocalType: "gide.CmdRuns", Doc: "currently running commands in this project", Directives: gti.Directives{}, Tag: "set:\"-\" json:\"-\" xml:\"-\""}},
		{"ArgVals", &gti.Field{Name: "ArgVals", Type: "goki.dev/gide/v2/gide.ArgVarVals", LocalType: "gide.ArgVarVals", Doc: "current arg var vals", Directives: gti.Directives{}, Tag: "set:\"-\" json:\"-\" xml:\"-\""}},
		{"Prefs", &gti.Field{Name: "Prefs", Type: "goki.dev/gide/v2/gide.ProjPrefs", LocalType: "gide.ProjPrefs", Doc: "preferences for this project -- this is what is saved in a .gide project file", Directives: gti.Directives{}, Tag: "set:\"-\""}},
		{"CurDbg", &gti.Field{Name: "CurDbg", Type: "*goki.dev/gide/v2/gide.DebugView", LocalType: "*gide.DebugView", Doc: "current debug view", Directives: gti.Directives{}, Tag: "set:\"-\""}},
		{"KeySeq1", &gti.Field{Name: "KeySeq1", Type: "goki.dev/goosi/events/key.Chord", LocalType: "key.Chord", Doc: "first key in sequence if needs2 key pressed", Directives: gti.Directives{}, Tag: "set:\"-\""}},
		{"UpdtMu", &gti.Field{Name: "UpdtMu", Type: "sync.Mutex", LocalType: "sync.Mutex", Doc: "mutex for protecting overall updates to GideView", Directives: gti.Directives{}, Tag: "set:\"-\""}},
	}),
	Embeds: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
		{"Frame", &gti.Field{Name: "Frame", Type: "goki.dev/gi/v2/gi.Frame", LocalType: "gi.Frame", Doc: "", Directives: gti.Directives{}, Tag: ""}},
	}),
	Methods: ordmap.Make([]ordmap.KeyVal[string, *gti.Method]{
		{"ExecCmdNameActive", &gti.Method{Name: "ExecCmdNameActive", Doc: "ExecCmdNameActive calls given command on current active textview", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"cmdNm", &gti.Field{Name: "cmdNm", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"ExecCmd", &gti.Method{Name: "ExecCmd", Doc: "ExecCmd pops up a menu to select a command appropriate for the current\nactive text view, and shows output in Tab with name of command", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"Build", &gti.Method{Name: "Build", Doc: "Build runs the BuildCmds set for this project", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"Run", &gti.Method{Name: "Run", Doc: "Run runs the RunCmds set for this project", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"Commit", &gti.Method{Name: "Commit", Doc: "Commit commits the current changes using relevant VCS tool.\nChecks for VCS setting and for unsaved files.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"CursorToHistPrev", &gti.Method{Name: "CursorToHistPrev", Doc: "CursorToHistPrev moves cursor to previous position on history list --\nreturns true if moved", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"CursorToHistNext", &gti.Method{Name: "CursorToHistNext", Doc: "CursorToHistNext moves cursor to next position on history list --\nreturns true if moved", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"ReplaceInActive", &gti.Method{Name: "ReplaceInActive", Doc: "ReplaceInActive does query-replace in active file only", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"CutRect", &gti.Method{Name: "CutRect", Doc: "CutRect cuts rectangle in active text view", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"CopyRect", &gti.Method{Name: "CopyRect", Doc: "CopyRect copies rectangle in active text view", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"PasteRect", &gti.Method{Name: "PasteRect", Doc: "PasteRect cuts rectangle in active text view", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"RegisterCopy", &gti.Method{Name: "RegisterCopy", Doc: "RegisterCopy saves current selection in active text view to register of given name\nreturns true if saved", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"name", &gti.Field{Name: "name", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"RegisterPaste", &gti.Method{Name: "RegisterPaste", Doc: "RegisterPaste pastes register of given name into active text view\nreturns true if pasted", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"name", &gti.Field{Name: "name", Type: "goki.dev/gide/v2/gide.RegisterName", LocalType: "gide.RegisterName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"CommentOut", &gti.Method{Name: "CommentOut", Doc: "CommentOut comments-out selected lines in active text view\nand uncomments if already commented\nIf multiple lines are selected and any line is uncommented all will be commented", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"Indent", &gti.Method{Name: "Indent", Doc: "Indent indents selected lines in active view", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"ReCase", &gti.Method{Name: "ReCase", Doc: "ReCase replaces currently selected text in current active view with given case", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"c", &gti.Field{Name: "c", Type: "goki.dev/gi/v2/texteditor/textbuf.Cases", LocalType: "textbuf.Cases", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"string", &gti.Field{Name: "string", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"JoinParaLines", &gti.Method{Name: "JoinParaLines", Doc: "JoinParaLines merges sequences of lines with hard returns forming paragraphs,\nseparated by blank lines, into a single line per paragraph,\nfor given selected region (full text if no selection)", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"TabsToSpaces", &gti.Method{Name: "TabsToSpaces", Doc: "TabsToSpaces converts tabs to spaces\nfor given selected region (full text if no selection)", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"SpacesToTabs", &gti.Method{Name: "SpacesToTabs", Doc: "SpacesToTabs converts spaces to tabs\nfor given selected region (full text if no selection)", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"DiffFiles", &gti.Method{Name: "DiffFiles", Doc: "DiffFiles shows the differences between two given files\nin side-by-side DiffView and in the console as a context diff.\nIt opens the files as file nodes and uses existing contents if open already.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"fnmA", &gti.Field{Name: "fnmA", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"DiffFileNode", &gti.Method{Name: "DiffFileNode", Doc: "DiffFileNode shows the differences between given file node as the A file,\nand another given file as the B file,\nin side-by-side DiffView and in the console as a context diff.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"fna", &gti.Field{Name: "fna", Type: "*goki.dev/gi/v2/filetree.Node", LocalType: "*filetree.Node", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"fnmB", &gti.Field{Name: "fnmB", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"CountWords", &gti.Method{Name: "CountWords", Doc: "CountWords counts number of words (and lines) in active file\nreturns a string report thereof.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"string", &gti.Field{Name: "string", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"CountWordsRegion", &gti.Method{Name: "CountWordsRegion", Doc: "CountWordsRegion counts number of words (and lines) in selected region in file\nif no selection, returns numbers for entire file.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"string", &gti.Field{Name: "string", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"SaveActiveView", &gti.Method{Name: "SaveActiveView", Doc: "SaveActiveView saves the contents of the currently-active textview", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"SaveActiveViewAs", &gti.Method{Name: "SaveActiveViewAs", Doc: "SaveActiveViewAs save with specified filename the contents of the\ncurrently-active textview", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"filename", &gti.Field{Name: "filename", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"RevertActiveView", &gti.Method{Name: "RevertActiveView", Doc: "RevertActiveView revert active view to saved version", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"CloseActiveView", &gti.Method{Name: "CloseActiveView", Doc: "CloseActiveView closes the buffer associated with active view", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"NextViewFile", &gti.Method{Name: "NextViewFile", Doc: "NextViewFile sets the next text view to view given file name -- include as\nmuch of name as possible to disambiguate -- will use the first matching --\nif already being viewed, that is activated -- returns textview and its\nindex, false if not found", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"fnm", &gti.Field{Name: "fnm", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"TextEditor", &gti.Field{Name: "TextEditor", Type: "*goki.dev/gide/v2/gide.TextEditor", LocalType: "*gide.TextEditor", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"int", &gti.Field{Name: "int", Type: "int", LocalType: "int", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"ViewFile", &gti.Method{Name: "ViewFile", Doc: "ViewFile views file in an existing TextEditor if it is already viewing that\nfile, otherwise opens ViewFileNode in active buffer", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"fnm", &gti.Field{Name: "fnm", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"TextEditor", &gti.Field{Name: "TextEditor", Type: "*goki.dev/gide/v2/gide.TextEditor", LocalType: "*gide.TextEditor", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"int", &gti.Field{Name: "int", Type: "int", LocalType: "int", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"SaveAll", &gti.Method{Name: "SaveAll", Doc: "SaveAll saves all of the open filenodes to their current file names\nand saves the project state if it has been saved before (i.e., the .gide file exists)", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"UpdateFiles", &gti.Method{Name: "UpdateFiles", Doc: "UpdateFiles updates the list of files saved in project", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"OpenRecent", &gti.Method{Name: "OpenRecent", Doc: "OpenRecent opens a recently-used file", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"filename", &gti.Field{Name: "filename", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"OpenFile", &gti.Method{Name: "OpenFile", Doc: "OpenFile opens file in an open project if it has the same path as the file\nor in a new window.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"fnm", &gti.Field{Name: "fnm", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"OpenPath", &gti.Method{Name: "OpenPath", Doc: "OpenPath creates a new project by opening given path, which can either be a\nspecific file or a folder containing multiple files of interest -- opens in\ncurrent GideView object if it is empty, or otherwise opens a new window.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"path", &gti.Field{Name: "path", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"GideView", &gti.Field{Name: "GideView", Type: "*goki.dev/gide/v2/gidev.GideView", LocalType: "*GideView", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"OpenProj", &gti.Method{Name: "OpenProj", Doc: "OpenProj opens .gide project file and its settings from given filename, in a standard\nJSON-formatted file", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"filename", &gti.Field{Name: "filename", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"GideView", &gti.Field{Name: "GideView", Type: "*goki.dev/gide/v2/gidev.GideView", LocalType: "*GideView", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"NewProj", &gti.Method{Name: "NewProj", Doc: "NewProj creates a new project at given path, making a new folder in that\npath -- all GideView projects are essentially defined by a path to a folder\ncontaining files.  If the folder already exists, then use OpenPath.\nCan also specify main language and version control type", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"path", &gti.Field{Name: "path", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"folder", &gti.Field{Name: "folder", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"mainLang", &gti.Field{Name: "mainLang", Type: "goki.dev/pi/v2/filecat.Supported", LocalType: "filecat.Supported", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"versCtrl", &gti.Field{Name: "versCtrl", Type: "goki.dev/gi/v2/filetree.VersCtrlName", LocalType: "filetree.VersCtrlName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"GideView", &gti.Field{Name: "GideView", Type: "*goki.dev/gide/v2/gidev.GideView", LocalType: "*GideView", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"NewFile", &gti.Method{Name: "NewFile", Doc: "NewFile creates a new file in the project", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"filename", &gti.Field{Name: "filename", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"addToVcs", &gti.Field{Name: "addToVcs", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"SaveProj", &gti.Method{Name: "SaveProj", Doc: "SaveProj saves project file containing custom project settings, in a\nstandard JSON-formatted file", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"SaveProjAs", &gti.Method{Name: "SaveProjAs", Doc: "SaveProjAs saves project custom settings to given filename, in a standard\nJSON-formatted file\nsaveAllFiles indicates if user should be prompted for saving all files\nreturns true if the user was prompted, false otherwise", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"filename", &gti.Field{Name: "filename", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"bool", &gti.Field{Name: "bool", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		})}},
		{"EditProjPrefs", &gti.Method{Name: "EditProjPrefs", Doc: "EditProjPrefs allows editing of project preferences (settings specific to this project)", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"SplitsSetView", &gti.Method{Name: "SplitsSetView", Doc: "SplitsSetView sets split view splitters to given named setting", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"split", &gti.Field{Name: "split", Type: "goki.dev/gide/v2/gide.SplitName", LocalType: "gide.SplitName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"SplitsSave", &gti.Method{Name: "SplitsSave", Doc: "SplitsSave saves current splitter settings to named splitter settings under\nexisting name, and saves to prefs file", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"split", &gti.Field{Name: "split", Type: "goki.dev/gide/v2/gide.SplitName", LocalType: "gide.SplitName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"SplitsSaveAs", &gti.Method{Name: "SplitsSaveAs", Doc: "SplitsSaveAs saves current splitter settings to new named splitter settings, and\nsaves to prefs file", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"name", &gti.Field{Name: "name", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"SplitsEdit", &gti.Method{Name: "SplitsEdit", Doc: "SplitsEdit opens the SplitsView editor to customize saved splitter settings", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"TopAppBar", &gti.Method{Name: "TopAppBar", Doc: "", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"tb", &gti.Field{Name: "tb", Type: "*goki.dev/gi/v2/gi.TopAppBar", LocalType: "*gi.TopAppBar", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"Find", &gti.Method{Name: "Find", Doc: "Find does Find / Replace in files, using given options and filters -- opens up a\nmain tab with the results and further controls.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"find", &gti.Field{Name: "find", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"repl", &gti.Field{Name: "repl", Type: "string", LocalType: "string", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"ignoreCase", &gti.Field{Name: "ignoreCase", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"regExp", &gti.Field{Name: "regExp", Type: "bool", LocalType: "bool", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"loc", &gti.Field{Name: "loc", Type: "goki.dev/gide/v2/gide.FindLoc", LocalType: "gide.FindLoc", Doc: "", Directives: gti.Directives{}, Tag: ""}},
			{"langs", &gti.Field{Name: "langs", Type: "[]goki.dev/pi/v2/filecat.Supported", LocalType: "[]filecat.Supported", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"Spell", &gti.Method{Name: "Spell", Doc: "Spell checks spelling in active text view", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"Symbols", &gti.Method{Name: "Symbols", Doc: "Symbols displays the Symbols of a file or package", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"Debug", &gti.Method{Name: "Debug", Doc: "Debug starts the debugger on the RunExec executable.", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"DebugTest", &gti.Method{Name: "DebugTest", Doc: "DebugTest runs the debugger using testing mode in current active textview path", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
		{"ChooseRunExec", &gti.Method{Name: "ChooseRunExec", Doc: "ChooseRunExec selects the executable to run for the project", Directives: gti.Directives{
			&gti.Directive{Tool: "gti", Directive: "add", Args: []string{}},
		}, Args: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{
			{"exePath", &gti.Field{Name: "exePath", Type: "goki.dev/gi/v2/gi.FileName", LocalType: "gi.FileName", Doc: "", Directives: gti.Directives{}, Tag: ""}},
		}), Returns: ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{})}},
	}),
	Instance: &GideView{},
})

// NewGideView adds a new [GideView] with the given name
// to the given parent. If the name is unspecified, it defaults
// to the ID (kebab-case) name of the type, plus the
// [ki.Ki.NumLifetimeChildren] of the given parent.
func NewGideView(par ki.Ki, name ...string) *GideView {
	return par.NewChild(GideViewType, name...).(*GideView)
}

// KiType returns the [*gti.Type] of [GideView]
func (t *GideView) KiType() *gti.Type {
	return GideViewType
}

// New returns a new [*GideView] value
func (t *GideView) New() ki.Ki {
	return &GideView{}
}

// SetProjRoot sets the [GideView.ProjRoot]:
// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename
func (t *GideView) SetProjRoot(v gi.FileName) *GideView {
	t.ProjRoot = v
	return t
}

// SetProjFilename sets the [GideView.ProjFilename]:
// current project filename for saving / loading specific Gide configuration information in a .gide file (optional)
func (t *GideView) SetProjFilename(v gi.FileName) *GideView {
	t.ProjFilename = v
	return t
}

// SetActiveLang sets the [GideView.ActiveLang]:
// language for current active filename
func (t *GideView) SetActiveLang(v filecat.Supported) *GideView {
	t.ActiveLang = v
	return t
}

// SetStatusMessage sets the [GideView.StatusMessage]:
// the last status update message
func (t *GideView) SetStatusMessage(v string) *GideView {
	t.StatusMessage = v
	return t
}

// SetOpenNodes sets the [GideView.OpenNodes]:
// list of open nodes, most recent first
func (t *GideView) SetOpenNodes(v gide.OpenNodes) *GideView {
	t.OpenNodes = v
	return t
}

// SetTooltip sets the [GideView.Tooltip]
func (t *GideView) SetTooltip(v string) *GideView {
	t.Tooltip = v
	return t
}

// SetClass sets the [GideView.Class]
func (t *GideView) SetClass(v string) *GideView {
	t.Class = v
	return t
}

// SetPriorityEvents sets the [GideView.PriorityEvents]
func (t *GideView) SetPriorityEvents(v []events.Types) *GideView {
	t.PriorityEvents = v
	return t
}

// SetCustomContextMenu sets the [GideView.CustomContextMenu]
func (t *GideView) SetCustomContextMenu(v func(m *gi.Scene)) *GideView {
	t.CustomContextMenu = v
	return t
}

// SetStackTop sets the [GideView.StackTop]
func (t *GideView) SetStackTop(v int) *GideView {
	t.StackTop = v
	return t
}

// SetStripes sets the [GideView.Stripes]
func (t *GideView) SetStripes(v gi.Stripes) *GideView {
	t.Stripes = v
	return t
}
