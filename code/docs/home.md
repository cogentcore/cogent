+++
URL = ""
Title = "Cogent Code"
+++

<img src="icon.svg" alt="Cogent Code icon" width="128">

**Cogent Code** is a code and text editor written in pure Go with [Cogent Core](https://cogentcore.org/core). It works as a general-purpose text editor and as a full development environment. Language servers such as gopls give you diagnostics, completion, hover docs, go to definition, rename, quick fixes, and more. Code also includes a file browser that knows about version control, a tabbed panel for command output, find and replace, symbols, a debugger, terminals, and a live preview. It is strongly keyboard focused, inspired by Emacs, and every key binding can be customized.

You can install it with:

```sh
go install cogentcore.org/cogent/code/cmd/cogentcode@latest
```

## Projects

Code works on *projects*. A project is a directory and all the files in it.

* Use **Open** in the toolbar to open an existing directory (or a single file), or **Open recent** to go back to one you used before.
* Use **File > New project** to make a new directory for a new project, and **File > New file** to add a file to it.

Each project has its own settings, which you edit with the **Settings** button in the toolbar. They are saved in a `.code` file (TOML) at the top level of the project, whenever you edit them and whenever you do **Save all**. The most useful project settings are:

* **Editor**: tab size, spaces or tabs, line numbers, word wrap, and other editor options for this project.
* **MainLang**: the main language of the project, guessed from the most common file type. It sets the defaults for building and running.
* **LangBackend**: `LSP` (the default) uses external language servers; `Parser` uses the built-in parser instead (see [Language support](#language-support)).
* **InlayHints**: shows inlay hints from servers that support them, such as parameter names and variable types, inline in the editor.
* **IgnoreProblems**: problem messages to hide. A problem whose message contains a key set to true is not shown. Common Go warnings are listed already, turned off, so you can easily turn them on.
* **ShowHints**: also shows info and hint problems (such as gopls modernize suggestions), not just errors and warnings.
* **VersionControl**: the version control system (git, svn, and so on), detected automatically.
* **BuildCommand**, **RunCommand**, and **RunExec**: what the **Build** and **Run** buttons do (see [Build, run, and debug](#build-run-and-debug)).
* **Debug**: parameters for the debugger.

The overall settings for the app, in the **Settings** item of the app menu, include the settings for all Cogent Core apps (such as the key map and theme) and the Code settings: options for each language, the commands of the Command menu, the saved split layouts, and the saved registers.

## Layout

The Code window has three main areas, separated by splitters:

* **File tree** (left): click a file to open it. Right-click for the context menu to run commands on files, or to rename, delete, duplicate, or otherwise manage them (several at once if you select several). Drag and drop moves files. Files are colored by their version control status, and file operations go through version control. The folders that are open in the file tree also set where Find searches by default. Files outside the project that you open (for example through go to definition) appear under `[external files]`.
* **Two editors** (middle): either editor can show any open file, including two places in the same file. The active editor is the one with the keyboard focus; the toolbar actions such as Save act on it. A typical way of working is to switch between the two editors and choose which file each shows, rather than using tabs of files.
* **Tabs** (right): all of the panels, and the output of commands. These include Find, Symbols, Spell, Problems, References, Preview, Debug, the console, terminals, and a tab for each command you run. Closing the tab of a running command stops it.

The **Splits** menu in the toolbar sets the sizes and arrangement of these areas: you can choose a saved layout, save the current one, or edit the saved ones.

The **Preview** tab shows a live rendering of the Markdown, HTML, or SVG file in the active editor, updated as you edit.

Use **View > Open terminal** to get a terminal in the project root, **New terminal** for more of them, and **Terminal cd to file** or **Terminal cd to project** to change the directory of the current one. **Terminal here** in the editor context menu opens one in the directory of the file.

Code watches the open folders of the project for changes made by other programs, such as a `git checkout`. Open files that you have not edited are reloaded automatically; if you have edited a file that changed on disk, Code asks what to do.

## Editing and navigation

The editor has the usual editing features, plus several that come from Emacs:

* `Control+Space` starts selecting text, so you can select with any navigation keys; `Control+G` cancels the selection and clears highlights.
* The paste history (`Command+Shift+V` or **Edit > Paste history**) lets you choose anything you cut or copied before.
* Rectangle cut, copy, and paste, and named registers (saved with your settings), are in the **Edit** menu.
* **Edit** also has comment region, indent, change case, join paragraph lines, and tabs to spaces (and back).
* `Control+J` jumps to a line; `Control+L` recenters the view around the cursor.
* `Control+[` and `Control+]` (also the arrow buttons in the toolbar) move back and forward through the positions you have visited, including across files.
* `Control+Shift+UpArrow` expands the selection to the next larger syntactic element (expression, statement, block, function), and `Control+Shift+DownArrow` shrinks it back.

The search field in the app bar finds files in the project and symbols in the current package, in addition to commands. Right-click in an editor for its context menu, which has most of the language features described next.

## Language support

By default, Code uses language servers through the Language Server Protocol (LSP) for the languages that have one: gopls for Go, typescript-language-server for JavaScript and TypeScript, pyright for Python, rust-analyzer for Rust, clangd for C and C++, bash-language-server for shell scripts, and vscode-json-language-server for JSON. Code starts the server for a file automatically when it finds the server on your `PATH`. With a language server you get:

* **Problems**: errors and warnings are underlined as you type, and the problem is shown when you hover over it or put the cursor on it. The **Problems** button opens the Problems tab, which lists the problems in the current file, directory, open folders, or the whole project, and updates as they change. Click a problem to go to it, or use the next problem key.
* **Hover**: hover over a symbol to see its documentation.
* **Completion**: completions appear as you type. `Control+.` shows completions on demand, headed by a **Go to definition** item and a **Docs** item for the symbol at the cursor.
* **Quick fixes**: when the cursor is on a problem that has fixes (organize imports, add a missing return, fix a typo, and so on), `Control+.` (or the bolt button in the toolbar) shows them instead, and the context menu has a **Fix:** item for each. All code actions available at the cursor, including refactorings such as extracting a function or variable, are in the **Code actions** submenu of the context menu.
* **Signature help**: while you type the arguments of a call, its signature appears below the cursor and in the status bar, with the current parameter marked. `Control+Shift+Space` shows all of the signatures in a dialog.
* **Navigation**: the context menu has **Go to definition**, **Go to type**, **Find implementations**, and **Find references** (which lists the references in the References tab). Other uses of the symbol at the cursor are highlighted as you move around.
* **Rename symbol**: renames the symbol at the cursor across the project. The edits are shown in the Find tab first: click one to see it, then **Replace** them one at a time or **All** at once. Files that were not open are opened with the changes; save them to write the changes.
* **Format file**: formats the file with the language server (gopls formats Go files, for example). Indenting a region with a language server also formats it.
* **Symbols**: the Symbols tab lists the symbols of the file or package.

### Installing language servers

The `lsp-server` helper lists, installs, and tests the language servers that Code uses. It uses Homebrew on macOS when you have it, and otherwise `go install`, npm, pipx, or apt, depending on the server and your platform.

```sh
go install cogentcore.org/cogent/code/cmd/lsp-server@latest
lsp-server list            # which servers are installed, and how to install the rest
lsp-server install         # install all of the missing servers (-n to preview)
lsp-server install gopls   # install just one
lsp-server test            # start each server and check that it works
```

### The built-in parser

For Markdown and TeX, Code uses its own built-in parser instead of a language server, for highlighting, completion, lookup, symbols, and indentation. You can change which languages use the parser with the **Parser** option of each language in the language options of the Code settings. To use the parser for every language it supports in a project, set **LangBackend** to `Parser` in the project settings. Languages that have neither a server nor the parser still get syntax highlighting.

## Build, run, and debug

* **Build** runs the **BuildCommand** of the project settings; by default `go build -v` for Go, `latexmk -pdf` for TeX, and otherwise `make`.
* **Run** runs the **RunCommand**; by default `core run` for Go, and otherwise the **RunExec** executable, which you can choose from the context menu of a file in the file tree.

Both run in the directory of the file in the active editor (or the project root), save your files first, and show their output in a tab. Links to files and lines in the output, such as compiler errors, take you there when you click them.

The **Command** button (and menu) has the command-line tools shared with Cogent Files, such as git, go, and core, each with a menu of its subcommands built from its help; choosing one shows a form for its flags and arguments. At the top of each command's menu are its macros, ready-made versions of the most common subcommands, and you can add your own with **New macro**. Commands run in the directory of the file in the active editor, with output in a tab named for the command. If a terminal has the keyboard focus, the command is typed into the terminal instead.

The debugger (for Go, using [delve](https://github.com/go-delve/delve)) runs in a Debug tab:

* **Debug** builds the **RunExec** executable in debug mode and runs it.
* **Debug test** debugs the tests in the directory of the active file; you can give a test name pattern, as with `go test -run`.
* **Command > Debug attach** attaches to a running process.

Set and clear breakpoints by double-clicking a line, or from the editor context menu. Breakpoints can only be set once the debugger has started, and they take effect when the program is stopped. In the Debug tab, double-click a stack frame to go to it or a variable to see its details. **Debug: Find frames** in the context menu shows the frames in all goroutines that are in the current file, closest to the cursor first. Close the Debug tab to end the session.

## Find, replace, and symbols

* **Find** (`Command+F`) searches across files, with options for case, regular expressions, languages, and where to search: the open folders of the file tree, all of the project, the current directory, or the current file. Results appear in the Find tab, where you can click to go to each one and replace them one at a time or all at once.
* `Control+S` does interactive search within the current file, as in Emacs, and works in command output too.
* **Edit > Replace in file** (`Command+R`) does interactive query-replace in the current file. Its **Lexical items** option only matches whole tokens, which is useful for short names like `i`.
* **Symbols** lists the types, functions, and other symbols of the current file or package, with a filter. Click one to go to it.
* **Spell** checks the spelling of the active file in a Spell tab, with suggestions and options to learn or ignore words.

## Version control

Code detects the version control system of the project (git, svn, and others) and shows the current branch in the status bar.

* The file tree colors files by their status and has version control actions in its context menu for the selected files: add, delete, commit, revert, diff, log, and blame.
* **Commit** in the toolbar saves your files and commits all of the changes, asking for the message.
* **VCS log** shows the log of commits in an interactive browser, from which you can compare any revisions and browse their diffs.
* **Command > VCS update all** updates (pulls) all of the repositories in the open folders of the file tree.
* The git commands and macros in the Command menu cover the rest, such as push, pull, and branches.

## Key bindings

You choose the key map, and customize it, in the Cogent Core settings. Below, `Command` means the Command key on macOS and Control on other platforms. Code's own multi-key sequences use `Control+X` and `Control+C` as prefixes on macOS and in the Emacs key maps, and `Control+E` and `Control+B` on Windows, Linux, and ChromeOS. The second key also works with Control held down (for example `Control+X Control+O`).

| Action | macOS and Emacs | Windows and Linux |
| --- | --- | --- |
| Next / previous panel | `Control+Tab` / `Control+Shift+Tab`, `Control+X o` / `Control+X p` | `Control+Tab` / `Control+Shift+Tab`, `Control+E o` / `Control+E p` |
| Open file | `Control+X f` | `Control+O`, `Control+E f` |
| Choose open file for editor | `Control+X b` | `Control+E b` |
| Show file in other editor | `Control+C o` | `Control+B o` |
| Save file / save as | `Control+X s` / `Control+X w` | `Control+S` / `Control+Shift+S` |
| Close file | `Control+X k` | `Control+E k` |
| Command menu | `Control+C c` | `Control+B c` |
| Build / run project | `Control+X m` / `Control+X r` | `Control+E m` / `Control+E r` |
| Problems / next problem | `Control+C p` / `Control+C n` | `Control+B p` / `Control+B n` |
| Comment region | `Control+C k` | `Control+/`, `Control+B k` |
| Indent region | `Control+X i` | `Control+E i` |
| Jump to line | `Control+X j`, `Control+J` | `Control+E j` |
| Register copy / paste | `Control+X x` / `Control+X g` | `Control+E x` / `Control+E g` |
| Rectangle cut / paste | `Control+X Control+X` / `Control+X Control+Y` | `Control+E Control+X` / `Control+E Control+Y` |
| Set split layout | `Control+X v` | `Control+E v` |

These keys are the same in every key map:

* `Control+.`: quick fixes at a problem, otherwise completions and go to definition.
* `Control+Shift+Space`: signature help.
* `Control+Shift+UpArrow` / `Control+Shift+DownArrow`: expand / shrink the selection.
* `Command+U`: update the file tree.
* `Control+M` (and `F10`): search the app bar for files, symbols, and commands.

On macOS, the standard Cogent Core keys include `Command+S` to save, `Command+F` to find, `Command+R` to replace, `Control+S` for interactive search, `Control+G` to cancel a selection, `Control+L` to recenter, `Control+[` and `Control+]` for back and forward, and `Command+Shift+V` for the paste history.

## Environment variables on macOS

When you run Code from an installed macOS app bundle, it does not inherit the environment of your shell, so tools such as `go` or the language servers may not be found. To set environment variables for GUI apps, use:

```sh
launchctl setenv VAR VALUE
```

The `goal` shell does this for you automatically when you set an environment variable. This does not work for `PATH`, which needs:

```sh
sudo launchctl config user path $PATH
```

This takes effect only after you restart your computer. After that, your shell starts with that path already, so commands in your shell startup files that add to the path will create duplicates.
