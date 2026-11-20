+++
URL = ""
Title = "Cogent Files"
+++

<img src="icon.svg" alt="Cogent Files icon" width="128">

Cogent Files is a file explorer for macOS, Windows, and Linux, built with Cogent Core. The window has a file tree, your favorite folders, tabs with the files of folders, and terminals. You can browse, open, copy, move, rename, and delete files. You can open a terminal in any folder in one step, and use the command bar to build and run command lines for tools such as `git` and `go`. It is laid out like Cogent Code and shares its favorites, its commands, and its folder sort orders with the other Cogent Core apps.

## Getting started

When you start Cogent Files, it shows your home directory. To start somewhere else, pass a directory on the command line, for example `files ~/projects`. That directory is the *root* of the file tree. It also opens in the first directory tab and in the default terminal.

## Layout and panels

The window has a toolbar and a command bar at the top, with these panels below them:

- **File tree** (top left): the files and folders in the root directory, which you can expand and collapse.
- **Favorites** (below the file tree): your favorite folders. Click one to make it the root of the file tree.
- **Directory tabs** (middle): each tab shows the files of one folder in a table, with the full path of the folder above the table. The tab is labeled with the folder's name, and its tooltip shows the full path.
- **Command tabs** (right): the **Console** tab, the default **Terminal** tab, and any other terminals you open.

You can drag the dividers between panels to resize them. The sizes are saved and restored the next time you start the app.

The toolbar has these items:

- **Up**: shows the parent folder in the current directory tab.
- **Terminal**: opens a new terminal in the folder of the current directory tab.
- **Layout chooser**: picks a layout of the panels (see Settings below).
- **More menu** (the three dots): the standard Cogent Core items, such as Search, Help, Settings, and Window (New window, Close window, Quit).

### Terminals and the console

Each terminal runs your shell, or the command set in the `Terminal` setting. The tab of a terminal is labeled with the name of its working directory, and the label changes when you `cd`. When several terminals have the same directory name, their labels are numbered. To open a new terminal:

- Click the **+** button of the command tabs to open one in the current directory.
- Click **Terminal** in the toolbar.
- Double-click the tab of a directory, or choose **Terminal here** from its context menu.
- Choose **Terminal here** from the context menu of a folder in the file tree, a directory table, or the favorites.

The **Console** tab shows the app's own output and error messages, with errors in the error color. You can clear it with Clear in its context menu, or with the Kill key (`Command+K`).

## Navigating

- **File tree**: double-click a folder, or choose **View** from its context menu, to show its files in a directory tab. If the folder is already shown in a tab, that tab is selected. With the keyboard, press `RightArrow` on an open folder to show it in a tab and move the focus to its table. Double-click a file, or press `Return`, to open it in its default app.
- **Directory tables**: double-click a folder, or select it and press `Return` or `Space`, to show its files in the same tab. Do the same with a file to open it in its default app. Use `UpArrow`, `DownArrow`, `PageUp`, and `PageDown` to move the selection.
- **Going up**: click **Up** in the toolbar, or press `Control+J` or `Control+LeftArrow` in a table, to show the parent folder.
- **Sorting**: click a column header to sort by that column; folders always stay at the top. The sort order is remembered for each folder and is shared with the other Cogent Core apps.
- **Changes on disk**: a directory table updates automatically when files are added, removed, or changed in its folder.
- **Hidden files**: files whose names start with a dot are hidden unless you turn on `ShowHidden`.
- **macOS apps**: app bundles open as apps when you double-click them or press `Return`. To see the files inside an app bundle, choose **View** from its context menu, or expand it in the file tree with its arrow.

To find text anywhere in the window, use **Search** in the More menu (`Command+F`). This searches the text that is currently shown in the window, such as the file names in the open tables. To find a menu item or app action by name, use **Menu search** (`Control+M` or `F10`).

## Favorites

Favorites are the favorite paths of the Cogent Core system settings, so they are the same as those in the file picker of every Cogent Core app. To add a folder, choose **Add to favorites** from its context menu in the file tree or a directory table. To add, remove, rename, or reorder favorites, right-click the favorites and choose **Edit favorites**. Your changes are saved when you close that window.

## File operations

You can select several files in a directory table with the usual modifier keys (`Shift` and `Command` with a click, or `Shift` with the arrow keys). The context menu of a directory table has these items:

- **Open**, and **Open with** to choose an app for a file.
- **Duplicate**: makes a copy of each selected file.
- **Delete**: moves the selected files to the trash (or recycling bin) after you confirm. You can also press `Delete`, `Backspace`, or `Command+Backspace`.
- **Rename**: renames the selected file (only one at a time).
- **Info**: shows information about the selected file.
- **New folder**: makes a folder in the folder that the table shows.
- For folders: **Terminal here** and **Add to favorites**.

The file tree has its own context menu, with Open, Duplicate, Delete, Rename, New file, New folder, Sort by, copy, cut and paste, and version control items for files in a Git repository.

### Drag and drop

- Drag files from a directory table to another table, the file tree, another Files window, or another app. Dropping files on a directory table copies them into its folder. Hold `Shift` or `Command` while you drop to move them instead. If files with the same names already exist, you are asked before they are overwritten.
- You can also drop files from other apps onto a directory table.
- Drag a directory tab or a terminal tab out of the window to open it in a new Files window with the same root. You can also drag a tab into the tabs of another Files window. A terminal keeps running and keeps its working directory when you move it.
- `Command+N` (Window > New window in the More menu) opens a new Files window with the same root.

## Command bar

The command bar under the toolbar has a button for each command and category of commands, such as `git`, `go`, and `core`, and other tools that are installed. Click a command, or choose a subcommand from its menu, to open a form of its flags and arguments. The form builds the command line for you and starts with the values you used last time.

When you run a command line, it is sent to the last terminal you used, as if you had typed it there. If there is no terminal, a new one opens in the current directory. Choosers in the forms, such as one for git branches, use that terminal's working directory, or else the current directory.

You can drag the buttons to reorder them. To edit the commands and categories, or add your own scripts and macros, click the pencil button at the end of the bar. The commands are shared with Cogent Code and the other Cogent Core apps.

## Keyboard shortcuts

Shortcuts for moving between panels:

| Action | macOS | Linux and Windows |
|---|---|---|
| Focus the next panel | `Control+Tab` or `Control+X o` | `Control+Tab` or `Control+E o` |
| Focus the previous panel | `Control+Shift+Tab` or `Control+X p` | `Control+Shift+Tab` or `Control+E p` |
| Choose the panel layout | `Control+X v` | `Control+E v` |

The second key of each sequence can also be pressed with `Control` (for example, `Control+X Control+O`). The panels are the file tree, the favorites, the directory tabs, and the command tabs; collapsed panels are skipped.

In a terminal, all other keys go to the terminal. `Control+Tab` and `Control+Shift+Tab` always work. The key sequences work only at the shell's command line, not inside a full-screen program such as an editor. If the second key does not complete a Files sequence, both keys go to the shell, so the shell's own sequences still work.

Other useful keys:

- `Return` or `Space`: open the selected file or folder in a directory table. `Command+Return` and `Command+O` also work.
- `Control+J` or `Control+LeftArrow`: go up to the parent folder in a directory table.
- `Delete`, `Backspace`, or `Command+Backspace`: move the selected files to the trash.
- `Command+F`: search the window. `Command+N`: open a new window. `Command+,`: open Settings.

`Command` is the Command key on macOS and the Control key on other platforms. You can see and change all of the key maps in the Cogent Core settings.

## Settings

To open the settings, choose **Settings** from the More menu (`Command+,`). The Files settings are:

- **Splits**: the layout of the panels.
  - `Wide` (the default): the file tree, the directory tabs, and the command tabs side by side.
  - `Compact`: the command tabs under the directory tabs, for smaller screens.
  - `Files`: the file tree and directory tabs only, with the command tabs collapsed.

  Choosing a layout, here, with the toolbar chooser, or with the layout key, resets the panel sizes to that layout's defaults. Sizes you set by dragging the dividers are saved automatically.
- **ShowHidden**: shows hidden files (names starting with a dot) in the file tree and the directory tables.
- **Terminal**: the command, with its arguments, to run in new terminals, such as `goal`. If it is empty, terminals run your default shell.

The settings are saved in the `settings.toml` file in the app's data directory.
