// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"github.com/mattn/go-shellwords"
)

// CmdAndArgs contains the name of an external program to execute and args to
// pass to that program
type CmdAndArgs struct {

	// external program to execute -- must be on path or have full path specified -- use {RunExec} for the project RunExec executable.
	Cmd string `width:"25"`

	// args to pass to the program, one string per arg.
	// Use {Filename} etc to refer to special variables.
	// Use backslash-quoted bracket to insert a literal curly bracket.
	// Use unix-standard path separators (/); they will be replaced with
	// proper os-specific path separator on Windows.
	Args CmdArgs `width:"25"`

	// default value for prompt string, for first use -- thereafter it uses last value provided for given command
	Default string `width:"25"`

	// if true, then do not split any prompted string into separate space-separated fields -- otherwise do so, except for values within quotes
	PromptIsString bool
}

// Label satisfies the Labeler interface
func (cm CmdAndArgs) Label() string {
	return cm.Cmd
}

// CmdArgs is a slice of arguments for a command
type CmdArgs []string

// HasPrompts returns true if any prompts are required before running command,
// and the set of such args
func (cm *CmdAndArgs) HasPrompts() (map[string]struct{}, bool) {
	var ps map[string]struct{}
	if aps, has := ArgVarPrompts(cm.Cmd); has {
		ps = aps
	}
	for _, av := range cm.Args {
		if aps, has := ArgVarPrompts(av); has {
			if ps == nil {
				ps = aps
			} else {
				for key := range aps {
					ps[key] = struct{}{}
				}
			}
		}
	}
	if len(ps) > 0 {
		return ps, true
	}
	return nil, false
}

// BindArgs replaces any variables in the args with their values, and returns resulting args
func (cm *CmdAndArgs) BindArgs(avp *ArgVarVals) []string {
	sz := len(cm.Args)
	if sz == 0 {
		return nil
	}
	args := []string{}
	for i := range cm.Args {
		argNm := cm.Args[i]
		av := avp.Bind(argNm)
		if len(av) == 0 {
			continue
		}
		switch {
		case !cm.PromptIsString && argNm == "{PromptString1}":
			fallthrough
		case !cm.PromptIsString && argNm == "{PromptString2}":
			flds, err := shellwords.Parse(av)
			if err != nil {
				fmt.Println(err)
			}
			args = append(args, flds...)
			continue
		case av[0] == '*': // only allow at *start* of args -- for *.ext exprs
			glob, err := filepath.Glob(av)
			if err == nil && len(glob) > 0 {
				args = append(args, glob...)
			}
			continue
		}
		args = append(args, av)
	}
	return args
}

// PrepCmd prepares to run command, returning *exec.Cmd and a string of the full command
func (cm *CmdAndArgs) PrepCmd(avp *ArgVarVals) (*exec.Cmd, string) {
	cstr := avp.Bind(cm.Cmd)
	switch cm.Cmd {
	case "{PromptString1}": // special case -- expand args
		cmdstr := cstr
		args, err := shellwords.Parse(cmdstr)
		if err != nil {
			fmt.Println(err)
		}
		if len(args) > 1 {
			cstr = args[0]
			args = args[1:]
		} else {
			cstr = args[0]
			args = nil
		}
		cmd := exec.Command(cstr, args...)
		return cmd, cmdstr
	case "open":
		cstr = filetree.OSOpenCommand()
		cmdstr := cstr
		args := cm.BindArgs(avp)
		if args != nil {
			astr := strings.Join(args, " ")
			cmdstr += " " + astr
		}
		cmd := exec.Command(cstr, args...)
		return cmd, cmdstr
	default:
		cmdstr := cstr
		args := cm.BindArgs(avp)
		if args != nil {
			astr := strings.Join(args, " ")
			cmdstr += " " + astr
		}
		cmd := exec.Command(cstr, args...)
		return cmd, cmdstr
	}
}

///////////////////////////////////////////////////////////////////////////
//  CmdRun, RunningCmds

// CmdRun tracks running commands
type CmdRun struct {

	// Name of command being run -- same as Command.Name
	Name string

	// command string
	CmdStr string

	// Details of the command and args
	CmdArgs *CmdAndArgs

	// exec.Cmd for the command
	Exec *exec.Cmd
}

// Kill kills the process
func (cm *CmdRun) Kill() {
	if cm.Exec.Process != nil {
		cm.Exec.Process.Kill()
	}
}

// CmdRuns is a slice list of running commands
type CmdRuns []*CmdRun

// Add adds a new running command
func (rc *CmdRuns) Add(cm *CmdRun) {
	if *rc == nil {
		*rc = make(CmdRuns, 0, 100)
	}
	*rc = append(*rc, cm)
}

// AddCmd adds a new running command, creating CmdRun via args
func (rc *CmdRuns) AddCmd(name, cmdstr string, cmdargs *CmdAndArgs, ex *exec.Cmd) {
	cm := &CmdRun{name, cmdstr, cmdargs, ex}
	rc.Add(cm)
}

// DeleteIndex delete command at given index
func (rc *CmdRuns) DeleteIndex(idx int) {
	*rc = append((*rc)[:idx], (*rc)[idx+1:]...)
}

// ByName returns command with given name
func (rc *CmdRuns) ByName(name string) (*CmdRun, int) {
	for i, cm := range *rc {
		if cm.Name == name {
			return cm, i
		}
	}
	return nil, -1
}

// DeleteByName deletes command by name
func (rc *CmdRuns) DeleteByName(name string) bool {
	_, idx := rc.ByName(name)
	if idx >= 0 {
		rc.DeleteIndex(idx)
		return true
	}
	return false
}

// KillByName kills a running process by name, and removes it from the list of
// running commands
func (rc *CmdRuns) KillByName(name string) bool {
	cm, idx := rc.ByName(name)
	if idx >= 0 {
		cm.Kill()
		rc.DeleteIndex(idx)
		return true
	}
	return false
}

///////////////////////////////////////////////////////////////////////////
//  Command

// Command defines different types of commands that can be run in the project.
// The output of the commands shows up in an associated tab.
type Command struct {

	// category for the command -- commands are organized in to hierarchical menus according to category
	Cat string

	// name of this command (must be unique in list of commands)
	Name string `width:"20"`

	// brief description of this command
	Desc string `width:"40"`

	// supported language / file type that this command applies to -- choose Any or e.g., AnyCode for subtypes -- filters the list of commands shown based on file language type
	Lang fileinfo.Known

	// sequence of commands to run for this overall command.
	Cmds []CmdAndArgs `table-select:"-"`

	// if specified, will change to this directory before executing the command;
	// e.g., use {FileDirPath} for current file's directory. Only use directory
	// values here; if not specified, directory will be project root directory.
	Dir string `width:"20"`

	// if true, we wait for the command to run before displaying output -- mainly for post-save commands and those with subsequent steps: if multiple commands are present, then it uses Wait mode regardless.
	Wait bool

	// if true, keyboard focus is directed to the command output tab panel after the command runs.
	Focus bool

	// if true, command requires Ok / Cancel confirmation dialog -- only needed for non-prompt commands
	Confirm bool

	//	what type of file to use for syntax highlighting.  Bash is the default.
	Hilight fileinfo.Known
}

// CommandName returns a qualified command name as cat: cmd
func CommandName(cat, cmd string) string {
	return cat + ": " + cmd
}

// Label satisfies the Labeler interface
func (cm Command) Label() string {
	return CommandName(cm.Cat, cm.Name)
}

// HasPrompts returns true if any prompts are required before running command,
// and the set of such args
func (cm *Command) HasPrompts() (map[string]struct{}, bool) {
	var ps map[string]struct{}
	for i := range cm.Cmds {
		cma := &cm.Cmds[i]
		if aps, has := cma.HasPrompts(); has {
			if ps == nil {
				ps = aps
			} else {
				for key := range aps {
					ps[key] = struct{}{}
				}
			}
		}
	}
	if len(ps) > 0 {
		return ps, true
	}
	return nil, false
}

// CmdNoUserPrompt can be set to true to prevent user from being prompted for strings
// this is useful when a custom outer-loop has already set the string values.
// this will be reset automatically after command is run.
var CmdNoUserPrompt bool

// CmdWaitOverride will cause the next commands that are run to be in wait mode
// (sequentially, waiting for completion after each), instead of running each in
// a separate process as is typical.  Don't forget to reset it after commands.
// This is important when running multiple of the same command, to prevent collisions
// in the output buffer.
var CmdWaitOverride bool

// CmdPrompt1Vals holds last values  for PromptString1 per command, so that
// each such command has its own appropriate history
var CmdPrompt1Vals = map[string]string{}

// CmdPrompt2Vals holds last values  for PromptString2 per command, so that
// each such command has its own appropriate history
var CmdPrompt2Vals = map[string]string{}

// RepoCurBranches returns the current branch and a list of all branches
// ensuring that the current also appears on the list of all.
// In git, a new branch may not so appear.
func RepoCurBranches(repo vcs.Repo) (string, []string, error) {
	cur, err := repo.Current()
	if err != nil {
		return "", nil, err
	}
	br, err := repo.Branches()
	if err != nil {
		return cur, nil, err
	}
	hasCur := false
	for _, b := range br {
		if b == cur {
			hasCur = true
			break
		}
	}
	if !hasCur {
		br = append(br, cur)
	}
	return cur, br, nil
}

// PromptUser prompts for values that need prompting for, and then runs
// RunAfterPrompts if not otherwise cancelled by user
func (cm *Command) PromptUser(cv *Code, buf *texteditor.Buffer, pvals map[string]struct{}) {
	sz := len(pvals)
	cnt := 0
	tv := cv.ActiveTextEditor()
	var cmvals map[string]string
	for pv := range pvals {
		switch pv {
		case "{PromptString1}":
			cmvals = CmdPrompt1Vals
			fallthrough
		case "{PromptString2}":
			if cmvals == nil {
				cmvals = CmdPrompt2Vals
			}
			curval, _ := cmvals[cm.Label()] // (*avp)[pv]
			if curval == "" && cm.Cmds[0].Default != "" {
				curval = cm.Cmds[0].Default
			}
			d := core.NewBody().AddTitle("Code Command Prompt").
				AddText(fmt.Sprintf("Command: %v: %v", cm.Name, cm.Desc))
			tf := core.NewTextField(d).SetText(curval)
			tf.Styler(func(s *styles.Style) {
				s.Min.X.Ch(100)
			})
			d.AddBottomBar(func(parent core.Widget) {
				d.AddCancel(parent)
				d.AddOK(parent).OnClick(func(e events.Event) {
					val := tf.Text()
					cmvals[cm.Label()] = val
					cv.ArgVals[pv] = val
					cnt++
					if cnt == sz {
						cm.RunAfterPrompts(cv, buf)
					}
				})
			})
			d.RunDialog(tv) // SetModal(false).

		// todo: looks like all the file prompts are not supported?
		case "{PromptBranch}":
			fn := cv.ActiveFileNode()
			if fn != nil {
				repo, _ := fn.Repo()
				if repo != nil {
					cur, br, err := RepoCurBranches(repo)
					if err == nil {
						m := core.NewMenuFromStrings(br, cur, func(idx int) {
							cv.ArgVals[pv] = br[idx]
							cnt++
							if cnt == sz {
								cm.RunAfterPrompts(cv, buf)
							}
						})
						m.Name = "prompt-branch"
						core.NewMenuStage(m, tv, tv.ContextMenuPos(nil)).Run()
					} else {
						fmt.Println(err)
					}
				}
			}
		}
	}
}

// Run runs the command and saves the output in the Buf if it is non-nil,
// which can be displayed -- if !wait, then Buf is updated online as output
// occurs.  Status is updated with status of command exec.  User is prompted
// for any values that might be needed for command.
func (cm *Command) Run(cv *Code, buf *texteditor.Buffer) {
	// if cm.Hilight != fileinfo.Unknown {
	// 	buf.Info.Known = cm.Hilight
	// 	buf.Info.Mime = fileinfo.MimeString(fileinfo.Bash)
	// 	buf.Hi.Lang = cm.Hilight.String()
	// }
	// todo: trying to use native highlighting
	// buf.Hi.Init(&buf.Info, nil)
	if cm.Confirm {
		d := core.NewBody().AddTitle("Confirm command").
			AddText(fmt.Sprintf("Command: %v: %v", cm.Label(), cm.Desc))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).SetText("Run").OnClick(func(e events.Event) {
				cm.RunAfterPrompts(cv, buf)
			})
		})
		d.RunDialog(cv.AsWidget().Scene)
		return
	}
	pvals, hasp := cm.HasPrompts()
	if !hasp || CmdNoUserPrompt {
		cm.RunAfterPrompts(cv, buf)
		return
	}
	cm.PromptUser(cv, buf, pvals)
}

// RunAfterPrompts runs after any prompts have been set, if needed
func (cm *Command) RunAfterPrompts(cv *Code, buf *texteditor.Buffer) {
	// ge.RunningCmds.KillByName(cm.Label()) // make sure nothing still running for us..
	CmdNoUserPrompt = false
	cdir := "{ProjectPath}"
	if cm.Dir != "" {
		cdir = cm.Dir
	}
	cds := cv.ArgVals.Bind(cdir)
	err := os.Chdir(cds)
	cm.AppendCmdOut(cv, buf, []byte(fmt.Sprintf("cd %v (from: %v)\n", cds, cdir)))
	if err != nil {
		cm.AppendCmdOut(cv, buf, []byte(fmt.Sprintf("Could not change to directory %v -- error: %v\n", cds, err)))
	}

	if CmdWaitOverride || cm.Wait || len(cm.Cmds) > 1 {
		for i := range cm.Cmds {
			cma := &cm.Cmds[i]
			if buf == nil {
				if !cm.RunNoBuf(cv, cma) {
					break
				}
			} else {
				if !cm.RunBufWait(cv, buf, cma) {
					break
				}
			}
		}
	} else if len(cm.Cmds) > 0 {
		cma := &cm.Cmds[0]
		if buf == nil {
			go cm.RunNoBuf(cv, cma)
		} else {
			go cm.RunBuf(cv, buf, cma)
		}
	}
}

// RunBufWait runs a command with output to the buffer, using CombinedOutput
// so it waits for completion -- returns overall command success, and logs one
// line of the command output to code statusbar
func (cm *Command) RunBufWait(cv *Code, buf *texteditor.Buffer, cma *CmdAndArgs) bool {
	cmd, cmdstr := cma.PrepCmd(&cv.ArgVals)
	cv.RunningCmds.AddCmd(cm.Label(), cmdstr, cma, cmd)
	out, err := cmd.CombinedOutput()
	cm.AppendCmdOut(cv, buf, out)
	return cm.RunStatus(cv, buf, cmdstr, err, out)
}

// RunBuf runs a command with output to the buffer, incrementally updating the
// buffer with new results line-by-line as they come in
func (cm *Command) RunBuf(cv *Code, buf *texteditor.Buffer, cma *CmdAndArgs) bool {
	cmd, cmdstr := cma.PrepCmd(&cv.ArgVals)
	cv.RunningCmds.AddCmd(cm.Label(), cmdstr, cma, cmd)
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		cmd.Stderr = cmd.Stdout
		err = cmd.Start()
		if err == nil {
			obuf := texteditor.OutputBuffer{}
			obuf.Init(stdout, buf, 0, cm.MarkupCmdOutput)
			obuf.MonitorOutput()
		}
		err = cmd.Wait()
	}
	return cm.RunStatus(cv, buf, cmdstr, err, nil)
}

// RunNoBuf runs a command without any output to the buffer -- can call using
// go as a goroutine for no-wait case -- returns overall command success, and
// logs one line of the command output to code statusbar
func (cm *Command) RunNoBuf(cv *Code, cma *CmdAndArgs) bool {
	cmd, cmdstr := cma.PrepCmd(&cv.ArgVals)
	cv.RunningCmds.AddCmd(cm.Label(), cmdstr, cma, cmd)
	out, err := cmd.CombinedOutput()
	return cm.RunStatus(cv, nil, cmdstr, err, out)
}

// AppendCmdOut appends command output to buffer, applying markup for links
func (cm *Command) AppendCmdOut(cv *Code, buf *texteditor.Buffer, out []byte) {
	if buf == nil {
		return
	}

	buf.SetReadOnly(true)

	lns := bytes.Split(out, []byte("\n"))
	sz := len(lns)
	outmus := make([][]byte, sz)
	for i, txt := range lns {
		outmus[i] = cm.MarkupCmdOutput(txt)
	}
	lfb := []byte("\n")
	mlns := bytes.Join(outmus, lfb)
	mlns = append(mlns, lfb...)

	buf.AppendTextMarkup(out, mlns, texteditor.EditSignal)
	buf.AutoScrollViews()
}

// CmdOutStatusLen is amount of command output to include in the status update
var CmdOutStatusLen = 80

// RunStatus reports the status of the command run (given in cmdstr) to
// ge.StatusBar -- returns true if there are no errors, and false if there
// were errors
func (cm *Command) RunStatus(cv *Code, buf *texteditor.Buffer, cmdstr string, err error, out []byte) bool {
	cv.RunningCmds.DeleteByName(cm.Label())
	var rval bool
	outstr := ""
	if out != nil {
		outstr = string(out[:CmdOutStatusLen])
	}
	finstat := ""
	tstr := time.Now().Format("Mon Jan  2 15:04:05 MST 2006")
	if err == nil {
		finstat = fmt.Sprintf("%v <b>successful</b> at: %v", cmdstr, tstr)
		rval = true
	} else if ee, ok := err.(*exec.ExitError); ok {
		finstat = fmt.Sprintf("%v <b>failed</b> at: %v with error: %v", cmdstr, tstr, ee.Error())
		rval = false
	} else {
		finstat = fmt.Sprintf("%v <b>exec error</b> at: %v error: %v", cmdstr, tstr, err.Error())
		rval = false
	}
	if buf != nil {
		buf.SetReadOnly(true)
		if err != nil {
			cv.SelectTabByName(cm.Label()) // sometimes it isn't
		}
		fsb := []byte(finstat)
		buf.AppendTextLineMarkup([]byte(""), []byte(""), texteditor.EditSignal)
		buf.AppendTextLineMarkup(fsb, cm.MarkupCmdOutput(fsb), texteditor.EditSignal)
		// todo: attempt to support syntax highlighting using builtin texteditor formatting
		// buf.AppendTextLine([]byte(""), texteditor.EditSignal)
		// buf.AppendTextLine(cm.MarkupCmdOutput(fsb), texteditor.EditSignal)
		buf.AutoScrollViews()
		if cm.Focus {
			cv.FocusOnTabs()
		}
	}
	cv.SetStatus(cmdstr + " " + outstr)
	return rval
}

// LangMatch returns true if the given language matches the command Lang constraints
func (cm *Command) LangMatch(lang fileinfo.Known) bool {
	return fileinfo.IsMatch(cm.Lang, lang)
}

// MarkupCmdOutput applies links to the first element in command output line
// if it looks like a file name / position
func (cm *Command) MarkupCmdOutput(out []byte) []byte {
	flds := strings.Fields(string(out))
	if len(flds) == 0 {
		return out
	}
	orig, link := lexer.MarkupPathsAsLinks(flds, 2) // only first 2 fields
	nt := out
	if len(link) > 0 {
		nt = bytes.Replace(out, orig, link, -1)
	}
	return nt
}

// MarkupCmdOutput applies links to the first element in command output line
// if it looks like a file name / position
func MarkupCmdOutput(out []byte) []byte {
	flds := strings.Fields(string(out))
	if len(flds) == 0 {
		return out
	}
	orig, link := lexer.MarkupPathsAsLinks(flds, 2) // only first 2 fields
	nt := out
	if len(link) > 0 {
		nt = bytes.Replace(out, orig, link, -1)
	}
	return nt
}

////////////////////////////////////////////////////////////////////////////////
//  Commands

// Commands is a list of different commands
type Commands []*Command //types:add

// CmdName has an associated ValueView for selecting from the list of
// available command names, for use in settings etc.
// Formatted as Cat: Name as in Command.Label()
type CmdName string

// IsValid checks if command name exists on AvailCmds list
func (cn CmdName) IsValid() bool {
	_, _, ok := AvailableCommands.CmdByName(cn, false)
	return ok
}

// Command returns command associated with command name in AvailCmds, and
// false if it doesn't exist
func (cn CmdName) Command() (*Command, bool) {
	cmd, _, ok := AvailableCommands.CmdByName(cn, true)
	return cmd, ok
}

// CmdNames is a slice of command names
type CmdNames []CmdName

// Add adds a name to the list
func (cn *CmdNames) Add(cmd CmdName) {
	*cn = append(*cn, cmd)
}

// AvailableCommands is the current list of ALL available commands for use -- it
// combines StdCmds and CustomCmds.  Custom overrides Std items with
// the same names.
var AvailableCommands Commands

// CustomCommands is user-specific list of commands saved in settings available
// for all Code projects.  These will override StdCmds with the same names.
var CustomCommands = Commands{}

// FilterCmdNames returns a slice of commands organized by category
// that are compatible with given language and version control system.
func (cm *Commands) FilterCmdNames(lang fileinfo.Known, vcnm filetree.VersionControlName) [][]string {
	vnm := strings.ToLower(string(vcnm))
	var cmds [][]string
	cat := ""
	var csub []string
	for _, cmd := range *cm {
		if cmd.LangMatch(lang) {
			if cmd.Cat != cat {
				lcat := strings.ToLower(cmd.Cat)
				if filetree.IsVersionControlSystem(lcat) && lcat != vnm {
					continue
				}
				cat = cmd.Cat
				csub = []string{cat}
				cmds = append(cmds, csub)
			}
			csub = append(csub, cmd.Name)
			cmds[len(cmds)-1] = csub // in case updated
		}
	}
	return cmds
}

func init() {
	AvailableCommands.CopyFrom(StandardCommands)
}

// CmdByName returns a command and index by name -- returns false and emits a
// message to log if not found if msg is true
func (cm *Commands) CmdByName(name CmdName, msg bool) (*Command, int, bool) {
	for i, cmd := range *cm {
		if cmd.Label() == string(name) {
			return cmd, i, true
		}
	}
	if msg {
		log.Printf("core.Commands.CmdByName: command named: %v not found\n", name)
	}
	return nil, -1, false
}

// CommandSettingsFilename is the name of the settings file in the app settings
// directory for saving / loading your CustomCommands commands list
var CommandSettingsFilename = "command-settings.toml"

// Open opens commands from a toml-formatted file.
func (cm *Commands) Open(filename core.Filename) error { //types:add
	*cm = make(Commands, 0, 10) // reset
	return errors.Log(jsonx.Open(cm, string(filename)))
}

// Save saves commands to a toml-formatted file.
func (cm *Commands) Save(filename core.Filename) error { //types:add
	return errors.Log(jsonx.Save(cm, string(filename)))
}

// OpenSettings opens custom Commands from the app settings directory, using
// CommandSettingsFilename.
func (cm *Commands) OpenSettings() error { //types:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, CommandSettingsFilename)
	CustomCommandsChanged = false
	err := cm.Open(core.Filename(pnm))
	if err == nil {
		MergeAvailableCmds()
	} else {
		cm = &Commands{} // restore a blank
	}
	return err
}

// SaveSettings saves custom Commands to the app settings directory, using
// CommandSettingsFilename.
func (cm *Commands) SaveSettings() error { //types:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, CommandSettingsFilename)
	CustomCommandsChanged = false
	err := cm.Save(core.Filename(pnm))
	if err == nil {
		MergeAvailableCmds()
	}
	return err
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

// MergeAvailableCmds updates the AvailCmds list from CustomCmds and StdCmds
func MergeAvailableCmds() {
	AvailableCommands.CopyFrom(StandardCommands)
	for _, cmd := range CustomCommands {
		_, idx, has := AvailableCommands.CmdByName(CmdName(cmd.Label()), false)
		if has {
			AvailableCommands[idx] = cmd // replace
		} else {
			AvailableCommands = append(AvailableCommands, cmd)
		}
	}
}

// ViewStandard shows the standard types that are compiled into the program and have
// all the lastest standards.  Useful for comparing against custom lists.
func (cm *Commands) ViewStandard() { //types:add
	CmdsView(&StandardCommands)
}

// CustomCommandsChanged is used to update core.CmdsView toolbars via following
// menu, toolbar properties update methods.
var CustomCommandsChanged = false

// CommandMenu returns a menu function for commands for given language and vcs name
func CommandMenu(fn *filetree.Node) func(mm *core.Scene) {
	cv, ok := ParentCode(fn.This)
	if !ok {
		return nil
	}
	lang := fn.Info.Known
	vcnm := cv.VersionControl()
	if repo, _ := fn.Repo(); repo != nil {
		vcnm = filetree.VersionControlName(repo.Vcs())
	}
	cmds := AvailableCommands.FilterCmdNames(lang, vcnm)
	lastCmd := ""
	hsz := len(cv.CmdHistory)
	if hsz > 0 {
		lastCmd = string((cv.CmdHistory)[hsz-1])
	}
	return func(mm *core.Scene) {
		for _, cc := range cmds {
			cc := cc
			n := len(cc)
			if n < 2 {
				continue
			}
			cmdCat := cc[0]
			ic := icons.Icon(strings.ToLower(cmdCat))
			if !ic.IsValid() {
				fmt.Println("icon not found", cmdCat)
			}
			cb := core.NewButton(mm).SetText(cmdCat).SetType(core.ButtonMenu).SetIcon(ic)
			cb.SetMenu(func(m *core.Scene) {
				for ii := 1; ii < n; ii++ {
					ii := ii
					it := cc[ii]
					cmdNm := CommandName(cmdCat, it)
					bt := core.NewButton(m).SetText(it).SetIcon(ic)
					bt.OnClick(func(e events.Event) {
						// e.SetHandled() // note: this allows menu to stay open :)
						cmd := CmdName(cmdNm)
						cv.CmdHistory.Add(cmd)         // only save commands executed via chooser
						cv.SaveAllCheck(true, func() { // true = cancel option
							cv.ExecCmdNameFileNode(fn, cmd, true, true) // sel, clear
						})
					})
					if cmdNm == lastCmd {
						bt.SetSelected(true)
					}
				}
			})
		}
	}
}
