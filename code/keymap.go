// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import "cogentcore.org/core/keymap"

func init() {
	keymap.AvailableMaps.MergeFrom(StandardKeyMaps)
}

// KeyFunctions are special functions for the overall control of the
// system: moving between windows, running commands, etc. Multi-key sequences can be used.
// type KeyFunctions keymap.Functions //enums:enum -trim-prefix Key

const (
	// move to next panel to the right
	KeyNextPanel keymap.Functions = keymap.FunctionsN + iota
	// move to prev panel to the left
	KeyPrevPanel
	// open a new file in active texteditor
	KeyFileOpen
	// select an open buffer to edit in active texteditor
	KeyBufSelect
	// open active file in other view
	KeyBufClone
	// save active texteditor buffer to its file
	KeyBufSave
	// save as active texteditor buffer to its file
	KeyBufSaveAs
	// close active texteditor buffer
	KeyBufClose
	// execute a command on active texteditor buffer
	KeyExecCmd
	// copy rectangle
	KeyRectCopy
	// cut rectangle
	KeyRectCut
	// paste rectangle
	KeyRectPaste
	// copy selection to named register
	KeyRegCopy
	// paste selection from named register
	KeyRegPaste
	// comment out region
	KeyCommentOut
	// indent region
	KeyIndent
	// jump to line (same as keyfun.Jump)
	KeyJump
	// set named splitter config
	KeySetSplit
	// build overall project
	KeyBuildProject
	// run overall project
	KeyRunProject
)

// StandardKeyMaps are the standard extended maps for Code
var StandardKeyMaps = keymap.Maps{
	{"MacStandard", "Standard Mac KeyMap", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+X o":         KeyNextPanel,
		"Control+X Control+O": KeyNextPanel,
		"Control+X p":         KeyPrevPanel,
		"Control+X Control+P": KeyPrevPanel,
		"Control+X f":         KeyFileOpen,
		"Control+X Control+F": KeyFileOpen,
		"Control+X b":         KeyBufSelect,
		"Control+X Control+B": KeyBufSelect,
		"Control+X s":         KeyBufSave,
		"Control+X Control+S": KeyBufSave,
		"Control+X w":         KeyBufSaveAs,
		"Control+X Control+W": KeyBufSaveAs,
		"Control+X k":         KeyBufClose,
		"Control+X Control+K": KeyBufClose,
		"Control+X c":         KeyExecCmd,
		"Control+X Control+C": KeyExecCmd,
		"Control+C c":         KeyExecCmd,
		"Control+C Control+C": KeyExecCmd,
		"Control+C o":         KeyBufClone,
		"Control+C Control+O": KeyBufClone,
		"Control+X x":         KeyRegCopy,
		"Control+X g":         KeyRegPaste,
		"Control+X Control+X": KeyRectCut,
		"Control+X Control+Y": KeyRectPaste,
		"Control+X Alt+∑":     KeyRectCopy,
		"Control+C k":         KeyCommentOut,
		"Control+C Control+K": KeyCommentOut,
		"Control+X i":         KeyIndent,
		"Control+X Control+I": KeyIndent,
		"Control+X j":         KeyJump,
		"Control+X Control+J": KeyJump,
		"Control+X v":         KeySetSplit,
		"Control+X Control+V": KeySetSplit,
		"Control+X m":         KeyBuildProject,
		"Control+X Control+M": KeyBuildProject,
		"Control+X r":         KeyRunProject,
		"Control+X Control+R": KeyRunProject,
	}},
	{"MacEmacs", "Mac with emacs-style navigation -- emacs wins in conflicts", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+X o":         KeyNextPanel,
		"Control+X Control+O": KeyNextPanel,
		"Control+X p":         KeyPrevPanel,
		"Control+X Control+P": KeyPrevPanel,
		"Control+X f":         KeyFileOpen,
		"Control+X Control+F": KeyFileOpen,
		"Control+X b":         KeyBufSelect,
		"Control+X Control+B": KeyBufSelect,
		"Control+X s":         KeyBufSave,
		"Control+X Control+S": KeyBufSave,
		"Control+X w":         KeyBufSaveAs,
		"Control+X Control+W": KeyBufSaveAs,
		"Control+X k":         KeyBufClose,
		"Control+X Control+K": KeyBufClose,
		"Control+X c":         KeyExecCmd,
		"Control+X Control+C": KeyExecCmd,
		"Control+C c":         KeyExecCmd,
		"Control+C Control+C": KeyExecCmd,
		"Control+C o":         KeyBufClone,
		"Control+C Control+O": KeyBufClone,
		"Control+X x":         KeyRegCopy,
		"Control+X g":         KeyRegPaste,
		"Control+X Control+X": KeyRectCut,
		"Control+X Control+Y": KeyRectPaste,
		"Control+X Alt+∑":     KeyRectCopy,
		"Control+C k":         KeyCommentOut,
		"Control+C Control+K": KeyCommentOut,
		"Control+X i":         KeyIndent,
		"Control+X Control+I": KeyIndent,
		"Control+X j":         KeyJump,
		"Control+X Control+J": KeyJump,
		"Control+X v":         KeySetSplit,
		"Control+X Control+V": KeySetSplit,
		"Control+X m":         KeyBuildProject,
		"Control+X Control+M": KeyBuildProject,
		"Control+X r":         KeyRunProject,
		"Control+X Control+R": KeyRunProject,
	}},
	{"LinuxEmacs", "Linux with emacs-style navigation -- emacs wins in conflicts", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+X o":         KeyNextPanel,
		"Control+X Control+O": KeyNextPanel,
		"Control+X p":         KeyPrevPanel,
		"Control+X Control+P": KeyPrevPanel,
		"Control+X f":         KeyFileOpen,
		"Control+X Control+F": KeyFileOpen,
		"Control+X b":         KeyBufSelect,
		"Control+X Control+B": KeyBufSelect,
		"Control+X s":         KeyBufSave,
		"Control+X Control+S": KeyBufSave,
		"Control+X w":         KeyBufSaveAs,
		"Control+X Control+W": KeyBufSaveAs,
		"Control+X k":         KeyBufClose,
		"Control+X Control+K": KeyBufClose,
		"Control+X c":         KeyExecCmd,
		"Control+X Control+C": KeyExecCmd,
		"Control+C c":         KeyExecCmd,
		"Control+C Control+C": KeyExecCmd,
		"Control+C o":         KeyBufClone,
		"Control+C Control+O": KeyBufClone,
		"Control+X x":         KeyRegCopy,
		"Control+X g":         KeyRegPaste,
		"Control+X Control+X": KeyRectCut,
		"Control+X Control+Y": KeyRectPaste,
		"Control+X Alt+∑":     KeyRectCopy,
		"Control+C k":         KeyCommentOut,
		"Control+C Control+K": KeyCommentOut,
		"Control+X i":         KeyIndent,
		"Control+X Control+I": KeyIndent,
		"Control+X j":         KeyJump,
		"Control+X Control+J": KeyJump,
		"Control+X v":         KeySetSplit,
		"Control+X Control+V": KeySetSplit,
		"Control+X m":         KeyBuildProject,
		"Control+X Control+M": KeyBuildProject,
		"Control+X r":         KeyRunProject,
		"Control+X Control+R": KeyRunProject,
	}},
	{"LinuxStandard", "Standard Linux key map", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+E o":         KeyNextPanel,
		"Control+E Control+O": KeyNextPanel,
		"Control+E p":         KeyPrevPanel,
		"Control+E Control+P": KeyPrevPanel,
		"Control+O":           KeyFileOpen,
		"Control+E f":         KeyFileOpen,
		"Control+E Control+F": KeyFileOpen,
		"Control+E b":         KeyBufSelect,
		"Control+E Control+B": KeyBufSelect,
		"Control+S":           KeyBufSave,
		"Control+Shift+S":     KeyBufSaveAs,
		"Control+E s":         KeyBufSave,
		"Control+E Control+S": KeyBufSave,
		"Control+E w":         KeyBufSaveAs,
		"Control+E Control+W": KeyBufSaveAs,
		"Control+E k":         KeyBufClose,
		"Control+E Control+K": KeyBufClose,
		"Control+B c":         KeyExecCmd,
		"Control+B Control+C": KeyExecCmd,
		"Control+B o":         KeyBufClone,
		"Control+B Control+O": KeyBufClone,
		"Control+E x":         KeyRegCopy,
		"Control+E g":         KeyRegPaste,
		"Control+E Control+X": KeyRectCut,
		"Control+E Control+Y": KeyRectPaste,
		"Control+E Alt+∑":     KeyRectCopy,
		"Control+/":           KeyCommentOut,
		"Control+B k":         KeyCommentOut,
		"Control+B Control+K": KeyCommentOut,
		"Control+E i":         KeyIndent,
		"Control+E Control+I": KeyIndent,
		"Control+E j":         KeyJump,
		"Control+E Control+J": KeyJump,
		"Control+E v":         KeySetSplit,
		"Control+E Control+V": KeySetSplit,
		"Control+E m":         KeyBuildProject,
		"Control+E Control+M": KeyBuildProject,
		"Control+E r":         KeyRunProject,
		"Control+E Control+R": KeyRunProject,
	}},
	{"WindowsStandard", "Standard Windows key map", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+E o":         KeyNextPanel,
		"Control+E Control+O": KeyNextPanel,
		"Control+E p":         KeyPrevPanel,
		"Control+E Control+P": KeyPrevPanel,
		"Control+O":           KeyFileOpen,
		"Control+E f":         KeyFileOpen,
		"Control+E Control+F": KeyFileOpen,
		"Control+E b":         KeyBufSelect,
		"Control+E Control+B": KeyBufSelect,
		"Control+S":           KeyBufSave,
		"Control+Shift+S":     KeyBufSaveAs,
		"Control+E s":         KeyBufSave,
		"Control+E Control+S": KeyBufSave,
		"Control+E w":         KeyBufSaveAs,
		"Control+E Control+W": KeyBufSaveAs,
		"Control+E k":         KeyBufClose,
		"Control+E Control+K": KeyBufClose,
		"Control+B c":         KeyExecCmd,
		"Control+B Control+C": KeyExecCmd,
		"Control+B o":         KeyBufClone,
		"Control+B Control+O": KeyBufClone,
		"Control+E x":         KeyRegCopy,
		"Control+E g":         KeyRegPaste,
		"Control+E Control+X": KeyRectCut,
		"Control+E Control+Y": KeyRectPaste,
		"Control+E Alt+∑":     KeyRectCopy,
		"Control+/":           KeyCommentOut,
		"Control+B k":         KeyCommentOut,
		"Control+B Control+K": KeyCommentOut,
		"Control+E i":         KeyIndent,
		"Control+E Control+I": KeyIndent,
		"Control+E j":         KeyJump,
		"Control+E Control+J": KeyJump,
		"Control+E v":         KeySetSplit,
		"Control+E Control+V": KeySetSplit,
		"Control+E m":         KeyBuildProject,
		"Control+E Control+M": KeyBuildProject,
		"Control+E r":         KeyRunProject,
		"Control+E Control+R": KeyRunProject,
	}},
	{"ChromeStd", "Standard chrome-browser and linux-under-chrome bindings", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+E o":         KeyNextPanel,
		"Control+E Control+O": KeyNextPanel,
		"Control+E p":         KeyPrevPanel,
		"Control+E Control+P": KeyPrevPanel,
		"Control+O":           KeyFileOpen,
		"Control+E f":         KeyFileOpen,
		"Control+E Control+F": KeyFileOpen,
		"Control+E b":         KeyBufSelect,
		"Control+E Control+B": KeyBufSelect,
		"Control+S":           KeyBufSave,
		"Control+Shift+S":     KeyBufSaveAs,
		"Control+E s":         KeyBufSave,
		"Control+E Control+S": KeyBufSave,
		"Control+E w":         KeyBufSaveAs,
		"Control+E Control+W": KeyBufSaveAs,
		"Control+E k":         KeyBufClose,
		"Control+E Control+K": KeyBufClose,
		"Control+B c":         KeyExecCmd,
		"Control+B Control+C": KeyExecCmd,
		"Control+B o":         KeyBufClone,
		"Control+B Control+O": KeyBufClone,
		"Control+E x":         KeyRegCopy,
		"Control+E g":         KeyRegPaste,
		"Control+E Control+X": KeyRectCut,
		"Control+E Control+Y": KeyRectPaste,
		"Control+E Alt+∑":     KeyRectCopy,
		"Control+/":           KeyCommentOut,
		"Control+B k":         KeyCommentOut,
		"Control+B Control+K": KeyCommentOut,
		"Control+E i":         KeyIndent,
		"Control+E Control+I": KeyIndent,
		"Control+E j":         KeyJump,
		"Control+E Control+J": KeyJump,
		"Control+E v":         KeySetSplit,
		"Control+E Control+V": KeySetSplit,
		"Control+E m":         KeyBuildProject,
		"Control+E Control+M": KeyBuildProject,
		"Control+E r":         KeyRunProject,
		"Control+E Control+R": KeyRunProject,
	}},
}
