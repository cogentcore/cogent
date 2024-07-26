**Cogent Code** is a general purpose code / text editor, written in pure Go, using the [Cogent Core](https://cogentcore.org/core) GUI (for which it serves as a continuous testing platform) and the [parse](https://pkg.go.dev/cogentcore.org/core/parse) interactive parser for syntax highlighting and more advanced code completion.

Some of the main features of *Code* include:

* Designed to function as both a general-purpose text editor and a reasonably powerful Go development tool, including a full GUI interface to the [delve](https://github.com/go-delve/delve) debugger. It comes configured with command support for LaTeX, Markdown, and Makefiles, in addition to Go, and the command system is fully extensible to support any command-line tools.

* Provides a tree-based file browser on the left, with built-in support for version control (git, svn, etc) and standard file management functionality through drag-and-drop, etc.  You can look at git logs, diffs, etc through this interface.  The `VCS Log` gives complete git log and version diff browser, so you don't need to go to github for all that.  The `Open` setting of Find uses the currently open folders in the file browser to control where searching happens, which can be useful when working in a large multi-repository project.

* Command actions show output on a tabbed output display, along with other special "Panels" such as Find / Replace, Symbols, Debugger, etc.  Thus, the overall design is extensible and new panels can be easily added to supply new functionality.  

* Strongly keyboard focused, inspired by Emacs, so existing Emacs users should be immediately productive.  However, other common keyboard bindings are also supported, and key bindings are entirely customizable.

# Install

See [Cogent Core](https://cogentcore.org/core) for general setup info for Cogent Core.

```
go install cogentcore.org/cogent/code/cmd/cogentcode@latest
```

* See Releases on this github page for pre-built OS-specific app packages that install the compiled binaries.

# Future Plans

We plan to incorporate [gopls](https://github.com/golang/tools/tree/master/gopls) to provide more comprehensive Go language IDE-level support, similar to what is found in VS Code.  At present, the completion can be a bit spotty for some kinds of expressions, and the lookup "go to definition" functionality is also not perfect. However, the basic code editing dynamics are pretty solid, so adding gopls would bring it much closer to feature-parity with VS Code.

# Help Guide

## General

The toolbar has lots of buttons to accomplish many tasks.  Depending on how wide your window is, these buttons can migrate to the `overflow menu` (three vertical dots) at the end of the toolbar.  In addition, some less-frequent functions are always found in that overflow menu, so you can look through there as well.

Cogent Code is based on the notion of a "project",  which is just a directory containing files.

* Use `Open` in toolbar to open an existing directory (e.g., a github project repository, etc).

* Or `File/New Project`  in the `:` overflow menu to create a new directory for a new project.

The project has its own specific `Settings` which can be accessed from the toolbar button of that name.  The permanent overflow menu `Settings` option is for the overall Cogent Core global settings, which include some important options for Code as well.  The project-specific Settings are saved in a `.code` file at the top level of your project, and are auto-saved whenever you edit the project settings, and whenever you click `Save All`.

A major design goal for Cogent Code is to replicate the `emacs`-like ability to use relatively-easy-to-remember keyboard sequences to do just about everything, without ever touching the mouse. 

The `Splits` menu can be used to to configure the size and layout of the different panels used in Code, which are as follows:

* __File Browser__ (on the left) -- click on files here to edit / run / view etc.  Use the Context Menu (hint: *always check the context menu!* -- i.e., right-mouse-click or whatever it is on your Mac) to run commands, or delete, rename, etc files (multiple files can be operated on at once).  You can also use drag-and-drop to move files into other subdirectories etc.

    + It knows about your VCS (version control system, e.g., git or svn) and color-codes files according to whether they are under VCS control or not, and have been edited since last commit.  File actions automatically work through the VCS (e.g., rename, move, delete etc).

    + The folders that are currently open in the File Browser determine which files are searched in Find / Replace!  So you can regulate that scope dynamically I that way (you can also tell Find to only look in the current Directory or other options too).

    + You can open files outside of the scope of the current file tree (e.g., using the Open action, or through Lookup or Debugging) -- they will show up at the top of the list under `[external files]`.  These are automatically excluded from any file operations.

* __Text Editor(s)__ (up to 2) -- this is obviously where you do your editing.  Many functions work on the *active* text view, which is brighter than the other one (e.g., the `Open`, `Save`, `Edit` etc buttons.

    + A key design feature (borrowed from emacs) is that either editor window can view any given underlying file (buffer), and that a typical workflow involves jumping frequently between these two editors and loading different files between each (including looking at two different regions of the same file).  This is different from the typical tabs-based design where each editor window has a fixed set of file tabs associated with it.  Once you learn the relevant shortcuts for switching between panels and selecting files to view in a given panel, it becomes a very fluid and powerful way of working.

    + The right-most editor window is "special" in that actions in the Tab views always operate on it.

* __Tabs__ -- contains all special-purpose "panels" (Find, Debug, Symbols..), and where the output of commands goes -- e.g., the output of your `make` or `build` commands, etc.  You can hit the `x` on a tab to kill a running process if you need to, and that's also how you close the debugger.

The idea here is to have a really simple overall plan, with easy customization of what panels are visible for different use-cases, so you know exactly where to look for anything.  We rely on tabs to hold all the many different kinds of output etc, and support multi-row tabs and a menu so you can really load them up.

The overall `Build` and `Run` commands are defined in the project Settings.  You can also select the executable to run (and debug) from the file tree context menu or in the Command menu.  

## Keyboard Usage

You can customize your keyboard shortcuts in the Core `Settings` panel.

* Use `Ctrl+G` to clear highlighting in a buffer (e.g., after a Find) -- it is the `CancelSelect` function and defined as this key by default on all platforms.  Some platforms also define `Shift+Control+A` to do this.

* The `Complete` key function (`Ctrl+.` in most cases) can be used on-demand to drive completion -- automatic continuous completion is only triggered when typing in new text.  It also will pull up a spelling correction menu on misspelled words.

* The `Lookup` key (`Ctrl+,` -- right next to Complete) will pull up the source code where any given entity is defined (e.g., the type of a variable, or method being called).  From there, you can also bring the file into the file tree view (as an external file if not under your current tree).

* To trigger a `Save All` of all edited files, just do `File/Save Project` which has the standard `Command+S` (Ctrl or the mac command key) shortcut.

* Use the `Recenter` command, which is `Ctrl+L` by default for all platforms, to trigger a re-highlight if anything gets out-of whack.

* The interactive "query-replace" function has a `Lexical Items` option, which is key for replacing short variable names like `i` which appear in many words etc -- it will only find cases of `i` that are separate lexical items (variables, etc).

* The full "kill ring" of prior cut / paste items is available with `Shift-Ctrl-Y` in a chooser -- very convenient to grab something you were pasting previously.  As in emacs, it moves any used item up to top so subsequent paste will be of that as well.

### Emacs mode

The KeyMaps "MacEmacs" and "LinuxEmacs" include the following emacs mappings.

Not *everything* in emacs is intended to be supported, but the most important keyboard functions are there, including a few that work slightly differently.  In general for the `Ctrl-X` and `Ctrl-C` execute / command prefixes, the subsequent key sequence can be *either* with a Ctrl or not, just because sometimes it is easier to keep the ctrl key down (except where there is a precedent for each use).

* `Esc`-based commands are *not* supported. Esc is used widely for abort throughout the Gi gui, and it is really awkward to hit.. it is time to fix that and just use Alt or another option!

* `Ctrl-X o` -- go to *next* ("other") panel to the right, looping around at the end back to the first panel.  Within the tabs panels, use `Ctrl+N` to quickly go down into the content of the tab.

* `Ctrl-X p` = go to *previous* panel on the left.

* `Ctrl-X Ctrl-o` = clone buffer in other textview -- very handy for looking up some other stuff in another place in same file.

* `Ctrl-X b` = choose buffer, `Ctrl-X f` = open file, `Ctrl-X s` = save file, `Ctrl-X w`  = save as file,  all work as expected.

* `Ctrl-C c` = execute command on active file

* `Ctrl-X x / g` = register copy / paste supported.  registers are saved with the .gide project and thus persistent across sessions.

* `Shift+Ctrl-Y` = Paste History, which works like `Esc Ctrl+Y` (yank previous), avoiding Esc, and adding a nice chooser.

* `Ctrl-J` = `goto-line` (i.e., jump to line).  never had a good emacs default and people mapped it differently.. `Ctrl-X Ctrl-J` is also avail just in case that was what you used.. 

* `Ctrl-U` = `PageUp`, in complement to `Ctrl-V` -- this is super convenient, and the "universal" arg thing in emacs isn't supported anyway.  Likewise, Esc-x command stuff is not supported.  This is NOT full emacs :)

* `Ctrl+/` = `Undo` -- this is one of several "standard" keys for undo.  Set the `Emacs Undo` option to enable the "undo the undo" native behavior of emacs, where typing any non-undo key causes all the undo actions to be added to the undo stack, so you can then do redo by just doing more undos at that point.

* `Ctrl-X Ctrl-X`, `Ctrl-X Alt-W`, `Ctrl-X Ctrl-Y` = rectangle versions of cut, copy, and paste, respectively -- can be very handy for repeated edits across multiple lines.

* `Ctrl+[`, `Ctrl+]` = `mark-ring` -- the emacs native commands for going to previous marks (set-mark-command) doesn't have a good default key, so we're using the web standard `Ctrl+[` and `Ctrl+]` (also ⌘ versions on the mac) for prev / next history.  cursor positions are saved "heuristically" at important junctures, not just mark sets (i.e., select toggle, `Ctrl+space`).  these can be extremely handy!  they are also stored with the buffer.

* `Ctrl-S` = interactive search, which works just like in emacs.  this is great for within-file searching (including in the Console and other command outputs).  Use "standard" Command+F for cross-file find / replace.

* `Ctrl-R` = query-replace (instead of `M-%`) -- it is mapped to `Ctrl-R` by default as more mnemonic and easier to type -- it is avail directly in TextView, so has a single-length sequence.  doesn't need full Cogent Code.

* `Ctrl-L` = recenter --  works with 3 different modes as you repeat, as in emacs.

* `Ctrl-T` = transpose letters (also Alt version for words).

### Non-Emacs Versions

The other shortcut maps attempt to use OS-specific conventions as much as possible.  How anyone gets anything done without the emacs navigation keys (Ctrl+F/B/N/P) remains a mystery (do you really lift your hand up and go all the way over to the arrow keys every single time you want to move the cursor??), but it is there..

The one place that is somewhat unconventional is in how we handle the multi-key sequences to activate many of the commands.  In emacs, `Ctrl+X` or `Ctrl+C` are general prefixes that then allow you to use more mnemonic 2nd keys in the sequence to execute various functions (see above for details).  This overall seems preferable to using more complex and obscure single-sequence shortcuts, which often involve so many simultaneous modifiers and such obscure letter mappings that they are likely rarely used in practice.

So, we chose `Ctrl-M` as the general `coMMand` key for these other platforms, as it is not generally used for anything else, and it is mnemonic.  Thus, by analogy with the emacs versions above, there are shortcuts such as the following.  The `Control+` version of the 2nd key is also defined as it is easier to keep the control key down sometimes:

* `Ctrl-M o` = move focus to "other" or next panel
* `Ctrl+M p` = move focus to previous panel
* `Ctrl+M c` = execute command
* `Ctrl+M x` = register copy
* `Ctrl+M g` = register get
* `Ctrl+M i` = indent
* `Ctrl+M k` = close (kill) buffer
* `Ctrl+M t` = comment / uncomment (k, c are taken) (also `Ctrl+/` is supported as used in some other IDE's)

Also, the following emacs-isms (which don't conflict with anything else) are supported and strongly recommended:

* `Ctrl+Spacebar` = start selecting text -- then you can just use any navigation keys to select whatever you want
* `Ctrl+G` = cancel selection and also clear any highlights
* `Ctrl+R` = query-replace
* `Alt+S` = interactive search

## Commands

Much of the language-specific functionality (and version control operations) is achieved by calling command-line commands, configured under the `Settings`: click on `Edit cmds` in toolbar, and then `View standard` in the toolbar of the window that comes up from there, to see all the standard built-in commands.

You can add additional custom commands to achieve anything not covered under the standard commands, and also any custom command with same name as a standard command will override the standard version, so that is how you can customize the standard commands.

You can do the Context menu on any of the standard commands and then Paste into the custom commands editor, and modify from there.

Click on the `Cmds` field to edit the actual commands executed -- there can be multiple individual commands executed for any given overall named command.

You can use variables specified below to insert values derived from the current active file or file being operated in the file browser, or variables defined in the project preferences.

### Special Arg Variables

```Go
	/// Current Filename
	"{FilePath}":       ArgVarInfo{"Current file name with full path.", ArgVarFile},
	"{FileName}":       ArgVarInfo{"Current file name only, without path.", ArgVarFile},
	"{FileExt}":        ArgVarInfo{"Extension of current file name.", ArgVarExt},
	"{FileExtLC}":      ArgVarInfo{"Extension of current file name, lowercase.", ArgVarExt},
	"{FileNameNoExt}":  ArgVarInfo{"Current file name without path and extension.", ArgVarFile},
	"{FileDir}":        ArgVarInfo{"Name only of current file's directory.", ArgVarDir},
	"{FileDirPath}":    ArgVarInfo{"Full path to current file's directory.", ArgVarDir},
	"{FileDirProjRel}": ArgVarInfo{"Path to current file's directory relative to project root.", ArgVarDir},

	// Project Root dir
	"{ProjDir}":  ArgVarInfo{"Current project directory name, without full path.", ArgVarDir},
	"{ProjPath}": ArgVarInfo{"Full path to current project directory.", ArgVarDir},

	// BuildDir
	"{BuildDir}":    ArgVarInfo{"Full path to BuildDir specified in project prefs -- the default Build.", ArgVarDir},
	"{BuildDirRel}": ArgVarInfo{"Path to BuildDir relative to project root.", ArgVarDir},

	// BuildTarg
	"{BuildTarg}":           ArgVarInfo{"Build target specified in prefs BuildTarg, just filename by itself, without path.", ArgVarFile},
	"{BuildTargPath}":       ArgVarInfo{"Full path to build target specified in prefs BuildTarg.", ArgVarFile},
	"{BuildTargDirPath}":    ArgVarInfo{"Full path to build target directory, without filename.", ArgVarDir},
	"{BuildTargDirPathRel}": ArgVarInfo{"Project-relative path to build target directory, without filename.", ArgVarDir},

	// RunExec
	"{RunExec}":           ArgVarInfo{"Run-time executable file RunExec specified in project prefs -- just the raw name of the file, without path.", ArgVarFile},
	"{RunExecPath}":       ArgVarInfo{"Full path to the run-time executable file RunExec specified in project prefs.", ArgVarFile},
	"{RunExecDirPath}":    ArgVarInfo{"Full path to the directory of the run-time executable file RunExec specified in project prefs.", ArgVarDir},
	"{RunExecDirPathRel}": ArgVarInfo{"Project-root relative path to the directory of the run-time executable file RunExec specified in project prefs.", ArgVarDir},

	// Cursor, Selection
	"{CurLine}":      ArgVarInfo{"Cursor current line number (starts at 1).", ArgVarPos},
	"{CurCol}":       ArgVarInfo{"Cursor current column number (starts at 0).", ArgVarPos},
	"{SelStartLine}": ArgVarInfo{"Selection starting line (same as CurLine if no selection).", ArgVarPos},
	"{SelStartCol}":  ArgVarInfo{"Selection starting column (same as CurCol if no selection).", ArgVarPos},
	"{SelEndLine}":   ArgVarInfo{"Selection ending line (same as CurLine if no selection).", ArgVarPos},
	"{SelEndCol}":    ArgVarInfo{"Selection ending column (same as CurCol if no selection).", ArgVarPos},

	"{CurSel}":      ArgVarInfo{"Currently selected text.", ArgVarText},
	"{CurLineText}": ArgVarInfo{"Current line text under cursor.", ArgVarText},
	"{CurWord}":     ArgVarInfo{"Current word under cursor.", ArgVarText},

	"{PromptFilePath}":       ArgVarInfo{"Prompt user for a file, and this is the full path to that file.", ArgVarPrompt},
	"{PromptFileName}":       ArgVarInfo{"Prompt user for a file, and this is the filename (only) of that file.", ArgVarPrompt},
	"{PromptFileDir}":        ArgVarInfo{"Prompt user for a file, and this is the directory name (only) of that file.", ArgVarPrompt},
	"{PromptFileDirPath}":    ArgVarInfo{"Prompt user for a file, and this is the full path to that directory.", ArgVarPrompt},
	"{PromptFileDirProjRel}": ArgVarInfo{"Prompt user for a file, and this is the path of that directory relative to the project root.", ArgVarPrompt},
	"{PromptString1}":        ArgVarInfo{"Prompt user for a string -- this is it.", ArgVarPrompt},
	"{PromptString2}":        ArgVarInfo{"Prompt user for another string -- this is it.", ArgVarPrompt},
```

## Debugging

The debugger (currently only supported for Go) runs through a Debug Tab, which provides full access to the running process information.

* `Debug` operates on the current `RunExec` (the same thing that is `Run` with that button).  It starts by building the executable in debug mode, so you typically have to wait a few secs (in Go) until the status says "Ready".

* `Debug Test` operates on the file in the active editor view, to determine the directory to debug tests in.

* `Debug Attach` in the Command menu allows you to attach to a running process.

In general, everything works through double-clicking.  If you want to move to a different frame in the Stack, double click on it.  If you want detail on a specific variable, double-click on it, etc..

### Breakpoints

**Breakpoints can only be set once the debugger has started**

**Breakpoints only 'take' when the debugger is Stopped**

If you set or modify a breakpoint when the process is running, it will show up in the Breaks list, but that will not reflect the actual set breakpoints until after you hit Stop or it stops on another breakpoint.  At which point, you'll likely have to *activate* any new breakpoints by toggling the On flag.  When you do `Cont` or `Step`, the breaks will update with more info indicating that they've been set in the debugger.

You set / delete breakpoints in the text editor, by double-clicking on a given line (or via the context menu).

All breakpoints will be auto-deleted when you close the debugger tab to quit a debugging session.

If you toggle a breakpoint On / Off in the Breaks view, that just tells the debugger whether to actually use a given breakpoint, but doesn't officially "delete" it -- it will still show up in the textview.  If you want to fully delete it, you can toggle it off in the textview.

### Find Frames
 
Particularly in Go, there can be a large number of Tasks (Go routines), most of which are at `runtime.gopark` as they wait for one thing or another, making it impossible to see from the `Task` list where something of interest might be (note: it does already try to avoid showing the runtime frames in the Task list, but that does not guarantee showing the frames of particular interest).

Find Frames, which you access via the context menu on a file you're viewing, will show you exactly those Frames across all the Tasks / Threads that have some stack frame which is in that file.  The frames are sorted in order of how close they are to the line you clicked on when you did Find Frames.  This makes it very easy to find the frames of interest.

### Params

See the `Project Prefs` for `Debug` params that control how much detail is returned about variables, and optional args you can pass to the debugger as needed.


