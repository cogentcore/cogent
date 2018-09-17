// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// CmdAndArgs contains the name of an external program to execute and args to
// pass to that program
type CmdAndArgs struct {
	Cmd  string   `desc:"external program to execute -- must be on path"`
	Args []string `desc:"args to pass to the program, one string per arg -- use {FileName} etc to refer to special variables -- just start typing { and you'll get a completion menu of options, and use \{ to insert a literal curly bracket.  A '/' path separator directly between path variables will be replaced with \ on Windows."`
}

// BindArgs replaces any variables in the args with their values, and returns resulting args
func (cm *CmdAndArgs) BindArgs() []string {
	sz := len(cm.Args)
	if sz == 0 {
		return nil
	}
	args := make([]string, sz)
	for i := range cm.Args {
		av := BindArgVars(cm.Args[i])
		args[i] = av
	}
	return args
}

// PrepCmd prepares to run command, returning *exec.Cmd and a string of the full command
func (cm *CmdAndArgs) PrepCmd() (*exec.Cmd, string) {
	args := cm.BindArgs()
	astr := strings.Join(args, " ")
	cmdstr := cm.Cmd + " " + astr
	cmd := exec.Command(cm.Cmd, args...)
	return cmd, cmdstr
}

// Command defines different types of commands that can be run in the project.
// The output of the commands shows up in an associated tab.
type Command struct {
	Name  string       `desc:"name of this type of project (must be unique in list of such types)"`
	Desc  string       `desc:"brief description of this command"`
	Langs LangNames    `desc:"language(s) that this command applies to -- leave empty if it applies to any -- filters the list of commands shown based on file language type"`
	Cmds  []CmdAndArgs `tableview-select:"-" desc:"sequence of commands to run for this overall command."`
	Wait  bool         `desc:"if true, we wait for the command to run before displaying output -- for quick commands and those where subsequent steps. If multiple commands are present, then subsequent steps always wait for prior steps in the sequence"`
	Buf   *giv.TextBuf `tableview:"-" view:"-" desc:"text buffer for displaying output of command"`
}

// MakeBuf creates the buffer object to save output from the command -- if
// this is not called in advance of Run, then output is ignored.  returns true
// if a new buffer was created, false if one already existed -- if clear is
// true, then any existing buffer is cleared.
func (cm *Command) MakeBuf(clear bool) bool {
	if cm.Buf != nil {
		if clear {
			cm.Buf.New(0)
		}
		return false
	}
	cm.Buf = &giv.TextBuf{}
	cm.Buf.InitName(cm.Buf, cm.Name+"-buf")
	return true
}

// Run runs the command and saves the output in the Buf if it is non-nil,
// which can be displayed -- if !wait, then Buf is updated online as output
// occurs.  Status is updated with status of command exec.
func (cm *Command) Run(ge *Gide) {
	if cm.Wait || len(cm.Cmds) > 1 {
		for i := range cm.Cmds {
			cma := &cm.Cmds[i]
			if cm.Buf == nil {
				if !cm.RunNoBuf(ge, cma) {
					break
				}
			} else {
				if !cm.RunBufWait(ge, cma) {
					break
				}
			}
		}
	} else {
		cma := &cm.Cmds[0]
		if cm.Buf == nil {
			go cm.RunNoBuf(ge, cma)
		} else {
			go cm.RunBuf(ge, cma)
		}
	}
}

// RunBufWait runs a command with output to the buffer, using CombinedOutput
// so it waits for completion -- returns overall command success, and logs one
// line of the command output to gide statusbar
func (cm *Command) RunBufWait(ge *Gide, cma *CmdAndArgs) bool {
	cmd, cmdstr := cma.PrepCmd()
	out, err := cmd.CombinedOutput()
	cm.Buf.AppendText(out)
	return cm.RunStatus(ge, cmdstr, err, out)
}

// RunBuf runs a command with output to the buffer, incrementally updating the
// buffer with new results line-by-line as they come in
func (cm *Command) RunBuf(ge *Gide, cma *CmdAndArgs) bool {
	cmd, cmdstr := cma.PrepCmd()
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		cmd.Stderr = cmd.Stdout
		err = cmd.Start()
		if err == nil {
			outscan := bufio.NewScanner(stdout) // line at a time
			for outscan.Scan() {
				cm.Buf.AppendTextLine(MarkupCmdOutput(outscan.Bytes()))
			}
		}
		err = cmd.Wait()
	}
	return cm.RunStatus(ge, cmdstr, err, nil)
}

// RunNoBuf runs a command without any output to the buffer -- can call using
// go as a goroutine for no-wait case -- returns overall command success, and
// logs one line of the command output to gide statusbar
func (cm *Command) RunNoBuf(ge *Gide, cma *CmdAndArgs) bool {
	cmd, cmdstr := cma.PrepCmd()
	out, err := cmd.CombinedOutput()
	return cm.RunStatus(ge, cmdstr, err, out)
}

// CmdOutStatusLen is amount of command output to include in the status update
var CmdOutStatusLen = 80

// RunStatus reports the status of the command run (given in cmdstr) to
// ge.StatusBar -- returns true if there are no errors, and false if there
// were errors
func (cm *Command) RunStatus(ge *Gide, cmdstr string, err error, out []byte) bool {
	rval := true
	outstr := ""
	if out != nil {
		outstr = string(out[:CmdOutStatusLen])
	}
	finstat := ""
	tstr := time.Now().Format("Mon Jan  2 15:04:05 MST 2006")
	if err == nil {
		finstat = fmt.Sprintf("%v <b>succesful</b> at: %v", cmdstr, tstr)
		rval = true
	} else if ee, ok := err.(*exec.ExitError); ok {
		finstat = fmt.Sprintf("%v <b>failed</b> at: %v with error: %v", cmdstr, tstr, ee.Error())
		rval = false
	} else {
		finstat = fmt.Sprintf("%v <b>exec error</b> at: %v error: %v", cmdstr, tstr, err.Error())
		rval = false
	}
	cm.Buf.AppendTextLine([]byte("\n"))
	cm.Buf.AppendTextLine(MarkupCmdOutput([]byte(finstat)))
	cm.Buf.Refresh()
	ge.SetStatus(cmdstr + " " + outstr)
	return rval
}

// LangMatch returns true if the given languages match those of the command,
// or command has no language restrictions
func (cm *Command) LangMatch(langs LangNames) bool {
	if len(cm.Langs) == 0 {
		return true
	}
	if len(langs) == 0 {
		return false
	}
	for _, cln := range cm.Langs {
		for _, lnm := range langs {
			if cln == lnm {
				return true
			}
		}
	}
	return false
}

// MarkupCmdOutput applies links to the first element in command output line
// if it looks like a file name / position
func MarkupCmdOutput(out []byte) []byte {
	flds := bytes.Fields(out)
	if len(flds) == 0 {
		return out
	}
	ff := flds[0]
	if bytes.Contains(ff, []byte(":")) {
		fnflds := bytes.Split(ff, []byte(":"))
		fn := string(fnflds[0])
		pos := string(fnflds[1])
		col := ""
		if len(fnflds) > 2 {
			col = string(fnflds[2])
		}
		cpath := ArgVarVals["{FileDirPath}"]
		if !strings.HasPrefix(fn, cpath) {
			fn = filepath.Join(cpath, strings.TrimPrefix(fn, "./"))
		}
		link := ""
		if col != "" {
			link = fmt.Sprintf(`<a href="file:///%v#L%vC%v">%v</a>`, fn, pos, col, string(ff))
		} else {
			link = fmt.Sprintf(`<a href="file:///%v#L%v">%v</a>`, fn, pos, string(ff))
		}
		flds[0] = []byte(link)
	} // todo: other cases, e.g., look for extension
	jf := bytes.Join(flds, []byte(" "))
	return jf
}

////////////////////////////////////////////////////////////////////////////////
//  Commands

// Commands is a list of different commands
type Commands []*Command

var KiT_Commands = kit.Types.AddType(&Commands{}, CommandsProps)

// CmdName has an associated ValueView for selecting from the list of
// available command names, for use in preferences etc.
type CmdName string

// IsValid checks if command name exists on AvailCmds list
func (cn CmdName) IsValid() bool {
	_, _, ok := AvailCmds.CmdByName(cn)
	return ok
}

// Command returns command associated with command name in AvailCmds, and
// false if it doesn't exist
func (cn CmdName) Command() (*Command, bool) {
	cmd, _, ok := AvailCmds.CmdByName(cn)
	return cmd, ok
}

// CmdNames is a slice of command names
type CmdNames []CmdName

// Add adds a name to the list
func (cn *CmdNames) Add(cmd CmdName) {
	*cn = append(*cn, cmd)
}

// AvailCmds is the current list of available commands for use -- can be
// loaded / saved / edited with preferences.  This is set to StdCommands at
// startup.
var AvailCmds Commands

// LangCmdNames returns a slice of commands that are compatible with given
// language(s).
func (cm *Commands) LangCmdNames(langs LangNames) []string {
	cmds := make([]string, 0, 100)
	for _, cmd := range *cm {
		if cmd.LangMatch(langs) {
			cmds = append(cmds, cmd.Name)
		}
	}
	return cmds
}

func init() {
	AvailCmds.CopyFrom(StdCommands)
}

// CmdByName returns a command and index by name -- returns false and emits a
// message to stdout if not found
func (cm *Commands) CmdByName(name CmdName) (*Command, int, bool) {
	for i, cmd := range *cm {
		if cmd.Name == string(name) {
			return cmd, i, true
		}
	}
	fmt.Printf("gi.Commands.CmdByName: command named: %v not found\n", name)
	return nil, -1, false
}

// PrefsCommandsFileName is the name of the preferences file in App prefs
// directory for saving / loading the default AvailCmds commands list
var PrefsCommandsFileName = "command_prefs.json"

// OpenJSON opens commands from a JSON-formatted file.
func (cm *Commands) OpenJSON(filename gi.FileName) error {
	*cm = make(Commands, 0, 10) // reset
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		// gi.PromptDialog(nil, gi.DlgOpts{Title: "File Not Found", Prompt: err.Error()}, true, false, nil, nil)
		// log.Println(err)
		return err
	}
	return json.Unmarshal(b, cm)
}

// SaveJSON saves commands to a JSON-formatted file.
func (cm *Commands) SaveJSON(filename gi.FileName) error {
	b, err := json.MarshalIndent(cm, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
	}
	return err
}

// OpenPrefs opens Commands from App standard prefs directory, using PrefsCommandsFileName
func (cm *Commands) OpenPrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsCommandsFileName)
	AvailCmdsChanged = false
	return cm.OpenJSON(gi.FileName(pnm))
}

// SavePrefs saves Commands to App standard prefs directory, using PrefsCommandsFileName
func (cm *Commands) SavePrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsCommandsFileName)
	AvailCmdsChanged = false
	return cm.SaveJSON(gi.FileName(pnm))
}

// CopyFrom copies commands from given other map
func (cm *Commands) CopyFrom(cp Commands) {
	*cm = make(Commands, 0, len(cp)) // reset
	b, err := json.Marshal(cp)
	if err != nil {
		fmt.Printf("json err: %v\n", err.Error())
	}
	json.Unmarshal(b, cm)
}

// RevertToStd reverts this map to using the StdCommands that are compiled into
// the program and have all the lastest standards.
func (cm *Commands) RevertToStd() {
	cm.CopyFrom(StdCommands)
	AvailCmdsChanged = true
}

// ViewStd shows the standard types that are compiled into the program and have
// all the lastest standards.  Useful for comparing against custom lists.
func (cm *Commands) ViewStd() {
	CmdsView(&StdCommands)
}

// AvailCmdsChanged is used to update giv.CmdsView toolbars via
// following menu, toolbar props update methods -- not accurate if editing any
// other map but works for now..
var AvailCmdsChanged = false

// CommandsProps define the ToolBar and MenuBar for TableView of Commands, e.g., CmdsView
var CommandsProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenPrefs", ki.Props{}},
			{"SavePrefs", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": func(cmi interface{}, act *gi.Action) {
					act.SetActiveState(AvailCmdsChanged)
				},
			}},
			{"sep-file", ki.BlankProp{}},
			{"OpenJSON", ki.Props{
				"label":    "Open from file",
				"desc":     "You can save and open commands to / from files to share, experiment, transfer, etc",
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"SaveJSON", ki.Props{
				"label": "Save to file",
				"desc":  "You can save and open commands to / from files to share, experiment, transfer, etc",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"RevertToStd", ki.Props{
				"desc":    "This reverts the commands to using the StdCommands that are compiled into the program and have all the lastest standards. <b>Your current edits will be lost if you proceed!</b>  Continue?",
				"confirm": true,
			}},
		}},
		{"Edit", "Copy Cut Paste Dupe"},
		{"Window", "Windows"},
	},
	"ToolBar": ki.PropSlice{
		{"SavePrefs", ki.Props{
			"desc": "saves Commands to App standard prefs directory, in file proj_types_prefs.json, which will be loaded automatically at startup if prefs SaveCommands is checked (should be if you're using custom commands)",
			"icon": "file-save",
			"updtfunc": func(cmi interface{}, act *gi.Action) {
				act.SetActiveStateUpdt(AvailCmdsChanged)
			},
		}},
		{"sep-file", ki.BlankProp{}},
		{"OpenJSON", ki.Props{
			"label": "Open from file",
			"icon":  "file-open",
			"desc":  "You can save and open commands to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", ki.Props{
			"label": "Save to file",
			"icon":  "file-save",
			"desc":  "You can save and open commands to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"sep-std", ki.BlankProp{}},
		{"ViewStd", ki.Props{
			"desc":    "Shows the standard types that are compiled into the program and have all the latest changes.  Useful for comparing against custom types.",
			"confirm": true,
		}},
		{"RevertToStd", ki.Props{
			"icon":    "update",
			"desc":    "This reverts the commands to using the StdCommands that are compiled into the program and have all the lastest standards.  <b>Your current edits will be lost if you proceed!</b>  Continue?",
			"confirm": true,
		}},
	},
}

// StdCommands is the original compiled-in set of standard commands.
var StdCommands = Commands{
	{"Imports Go File", "run goimports on file", LangNames{"Go"},
		[]CmdAndArgs{CmdAndArgs{"goimports", []string{"-w", "{FilePath}"}}}, true, nil},
	{"Fmt Go File", "run go fmt on file", LangNames{"Go"},
		[]CmdAndArgs{CmdAndArgs{"gofmt", []string{"-w", "{FilePath}"}}}, true, nil},
	{"Build Go", "run go build to build in current dir", LangNames{"Go"},
		[]CmdAndArgs{CmdAndArgs{"go", []string{"build", "-v", "{FileDirPath}"}}}, false, nil},
	{"Vet Go", "run go vet in current dir", LangNames{"Go"},
		[]CmdAndArgs{CmdAndArgs{"go", []string{"vet", "{FileDirPath}"}}}, false, nil},
	{"List Dir", "list current dir -- just for testing", nil,
		[]CmdAndArgs{CmdAndArgs{"ls", []string{"-la"}}}, false, nil},
	{"Git Status", "git status", nil,
		[]CmdAndArgs{CmdAndArgs{"git", []string{"status", "{FileDirPath}"}}}, true, nil},
	{"Git Push", "git push", nil,
		[]CmdAndArgs{CmdAndArgs{"git", []string{"push"}}}, true, nil},
	{"PDFLaTeX File", "run PDFLaTeX on file", LangNames{"LaTeX"},
		[]CmdAndArgs{CmdAndArgs{"pdflatex", []string{"-file-line-error", "-interaction=nonstopmode", "{FilePath}"}}}, false, nil},
}
