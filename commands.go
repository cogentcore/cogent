// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"

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

// Command defines different types of commands that can be run in the project.
// The output of the commands shows up in an associated tab.
type Command struct {
	Name  string       `desc:"name of this type of project (must be unique in list of such types)"`
	Desc  string       `desc:"brief description of this command"`
	Langs LangNames    `desc:"language(s) that this command applies to -- leave empty if it applies to any -- filters the list of commands shown based on file language type"`
	Cmds  []CmdAndArgs `desc:"sequence of commands to run for this overall command."`
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
// which can be displayed -- if !wait, then Buf is updated online as output occurs.
func (cm *Command) Run() {
	if cm.Wait || len(cm.Cmds) > 1 {
		for i := range cm.Cmds {
			cma := &cm.Cmds[i]
			cmd := exec.Command(cma.Cmd, cma.BindArgs()...)
			out, err := cmd.CombinedOutput()
			if err == nil {
				if cm.Buf != nil {
					cm.Buf.AppendText(out)
					fmt.Printf("out:\n%v\n", string(out))
				}
			} else if ee, ok := err.(*exec.ExitError); ok {
				fmt.Printf("Command failed with error: %v\n", ee.Error())
				break
			} else {
				fmt.Printf("Cmd exec error: %v\n", err.Error())
				break
			}
		}
	} else {
	}
}

////////////////////////////////////////////////////////////////////////////////
//  Commands

// Commands is a list of different commands
type Commands []Command

var KiT_Commands = kit.Types.AddType(&Commands{}, CommandsProps)

// CmdName has an associated ValueView for selecting from the list of
// available command names, for use in preferences etc.
type CmdName string

// CmdNames is a slice of command names
type CmdNames []CmdName

// AvailCmds is the current list of available commands for use -- can be
// loaded / saved / edited with preferences.  This is set to StdCommands at
// startup.
var AvailCmds Commands

func init() {
	AvailCmds.CopyFrom(StdCommands)
}

// CmdByName returns a command and index by name -- returns false and emits a
// message to stdout if not found
func (cm *Commands) CmdByName(name CmdName) (*Command, int, bool) {
	for i, _ := range *cm {
		cmd := &((*cm)[i])
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
	{"Go Fmt File", "run go fmt on file", LangNames{"Go"}, []CmdAndArgs{CmdAndArgs{"go", []string{"fmt", "{FilePath}"}}}, true, nil},
	{"Go Imports File", "run goimports on file", LangNames{"Go"}, []CmdAndArgs{CmdAndArgs{"goimports", []string{"{FilePath}"}}}, true, nil},
}
