// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
)

// https://www.eclipse.org/pdt/help/html/keymap.htm
// https://resources.jetbrains.com/storage/products/rubymine/docs/RubyMine_ReferenceCard.pdf
// https://docs.microsoft.com/en-us/visualstudio/ide/default-keyboard-shortcuts-in-visual-studio?view=vs-2017
// https://swifteducation.github.io/assets/pdfs/XcodeKeyboardShortcuts.pdf
// https://en.wikipedia.org/wiki/Table_of_keyboard_shortcuts <- great!

// KeyFunctions are special functions for the overall control of the
// system: moving between windows, running commands, etc. Multi-key sequences can be used.
type KeyFunctions int32 //enums:enum -trim-prefix Key

const (
	KeyNone KeyFunctions = iota
	// special internal signal returned by KeyFunction indicating need for second key
	KeyNeeds2
	// move to next panel to the right
	KeyNextPanel
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

// KeySeq defines a multiple-key sequence to initiate a key function
type KeySeq struct {
	Key1 key.Chord // first key
	Key2 key.Chord // second key (optional)
}

// String() satisfies fmt.Stringer interface
func (kf KeySeq) String() string {
	return string(kf.Key1 + " " + kf.Key2)
}

// Label satisfies core.Labeler interface
func (kf KeySeq) Label() string {
	return string(kf.Key1 + " " + kf.Key2)
}

// MarshalText is required for encoding of struct keys
func (kf KeySeq) MarshalText() ([]byte, error) {
	bs := make([][]byte, 2)
	bs[0] = []byte(kf.Key1)
	bs[1] = []byte(kf.Key2)
	b := bytes.Join(bs, []byte(";"))
	return b, nil
}

// UnmarshalText is required for decoding of struct keys
func (kf *KeySeq) UnmarshalText(b []byte) error {
	bs := bytes.Split(b, []byte(";"))
	kf.Key1 = key.Chord(string(bs[0]))
	kf.Key2 = key.Chord(string(bs[1]))
	return nil
}

// KeySeqMap is a map between a multi-key sequence (multiple chords) and a
// specific key function.  This mapping must be unique, in that each chord
// has a unique key function, but multiple chords can trigger the same function.
type KeySeqMap map[KeySeq]KeyFunctions

// ActiveKeyMap points to the active map -- users can set this to an
// alternative map in Settings
var ActiveKeyMap *KeySeqMap

// ActiveKeyMapName is the name of the active keymap
var ActiveKeyMapName KeyMapName

// Needs2KeyMap is a map of the starting key sequences that require a second
// key -- auto-generated from active keymap
var Needs2KeyMap keymap.Map

// SetActiveKeyMap sets the current ActiveKeyMap, calling Update on the map
// prior to setting it to ensure that it is a valid, complete map
func SetActiveKeyMap(km *KeySeqMap, kmName KeyMapName) {
	km.Update(kmName)
	ActiveKeyMap = km
	ActiveKeyMapName = kmName
}

// SetActiveKeyMapName sets the current ActiveKeyMap by name from those
// defined in AvailKeyMaps, calling Update on the map prior to setting it to
// ensure that it is a valid, complete map
func SetActiveKeyMapName(mapnm KeyMapName) {
	km, _, ok := AvailableKeyMaps.MapByName(mapnm)
	if ok {
		SetActiveKeyMap(km, mapnm)
	} else {
		log.Printf("code.SetActiveKeyMapName: key map named: %v not found, using default: %v\n", mapnm, DefaultKeyMap)
		km, _, ok = AvailableKeyMaps.MapByName(DefaultKeyMap)
		if ok {
			SetActiveKeyMap(km, DefaultKeyMap)
		} else {
			log.Printf("code.SetActiveKeyMapName: ok, this is bad: DefaultKeyMap not found either -- size of AvailKeyMaps: %v -- trying first one\n", len(AvailableKeyMaps))
			if len(AvailableKeyMaps) > 0 {
				skm := AvailableKeyMaps[0]
				SetActiveKeyMap(&skm.Map, KeyMapName(skm.Name))
			}
		}
	}
}

// KeyFunction translates chord into keyboard function; use [events.Event.KeyChord] to
// get chord; it returns KeyFunNeeds2 if the key sequence requires 2 keys to
// be entered, and only the first is present
func KeyFunction(key1, key2 key.Chord) KeyFunctions {
	kf := KeyNone
	ks := KeySeq{key1, key2}
	if key1 != "" && key2 != "" {
		if kfg, ok := (*ActiveKeyMap)[ks]; ok {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun 2 key seq: %v = %v\n", ks, kfg)
			}
			kf = kfg
		}
	} else if key1 != "" {
		if _, need2 := Needs2KeyMap[key1]; need2 {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun 1st key in 2key seq: %v\n", key1)
			}
			return KeyNeeds2
		}
		if kfg, ok := (*ActiveKeyMap)[ks]; ok {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun 1 key seq: %v = %v\n", ks, kfg)
			}
			kf = kfg
		}
	}
	return kf
}

// KeyMapItem records one element of the key map -- used for organizing the map.
type KeyMapItem struct {

	// the key chord sequence that activates a function
	Keys KeySeq

	// the function of that key
	Fun KeyFunctions
}

// ToSlice copies this keymap to a slice of KeyMapItem's
func (km *KeySeqMap) ToSlice() []KeyMapItem {
	kms := make([]KeyMapItem, len(*km))
	idx := 0
	for key, fun := range *km {
		kms[idx] = KeyMapItem{key, fun}
		idx++
	}
	return kms
}

// ChordForFunction returns first key sequence trigger for given KeyFunctions in map
func (km *KeySeqMap) ChordForFunction(kf KeyFunctions) KeySeq {
	if km == nil {
		return KeySeq{}
	}
	for key, fun := range *km {
		if fun == kf {
			return key
		}
	}
	return KeySeq{}
}

// ChordForFunction returns first key sequence trigger for given KeyFunctions in ActiveKeyMap
func ChordForFunction(kf KeyFunctions) KeySeq {
	return ActiveKeyMap.ChordForFunction(kf)
}

// Update ensures that the given keymap has at least one entry for every
// defined KeyFunctions, grabbing ones from the default map if not, and also
// eliminates any Nil entries which might reflect out-of-date functions
func (km *KeySeqMap) Update(kmName KeyMapName) {
	for key, val := range *km {
		if val == KeyNone {
			log.Printf("code.KeySeqMap: key function is nil -- probably renamed, for key: %v\n", key)
			delete(*km, key)
		}
	}
	kms := km.ToSlice()
	addkm := make([]KeyMapItem, 0)

	sort.Slice(kms, func(i, j int) bool {
		return kms[i].Fun < kms[j].Fun
	})

	lfun := KeyNeeds2
	for _, ki := range kms {
		fun := ki.Fun
		if fun != lfun {
			del := fun - lfun
			if del > 1 {
				for mi := lfun + 1; mi < fun; mi++ {
					fmt.Printf("code.KeyMap: %v is missing a key for function: %v\n", kmName, mi)
					s := mi.String()
					s = strings.TrimPrefix(s, "KeyFun")
					s = "- Not Set - " + s
					nski := KeyMapItem{Keys: KeySeq{Key1: key.Chord(s)}, Fun: mi}
					addkm = append(addkm, nski)
				}
			}
			lfun = fun
		}
	}

	for _, ai := range addkm {
		(*km)[ai.Keys] = ai.Fun
	}

	// now collect all the Needs2 cases, and make sure there aren't any
	// "needs1" that start with needs2!
	Needs2KeyMap = make(keymap.Map)

	for key := range *km {
		if key.Key2 != "" {
			Needs2KeyMap[key.Key1] = keymap.None
		}
	}

	// issue warnings for needs1 with same
	for key, val := range *km {
		if key.Key2 == "" {
			if _, need2 := Needs2KeyMap[key.Key1]; need2 {
				log.Printf("code.KeySeqMap: single-key case starts with key chord that is used in key sequence (2 keys in a row) in other mappings -- this is not valid and won't be used: Key: %v  Fun: %v\n",
					key, val)
			}
		}
	}
}

/////////////////////////////////////////////////////////////////////////////////
// KeyMaps -- list of KeyMap's

// KeyMapName has an associated ValueView for selecting from the list of
// available key map names, for use in settings etc.
type KeyMapName string

// DefaultKeyMap is the overall default keymap -- reinitialized in gimain init()
// depending on platform
var DefaultKeyMap = KeyMapName("MacEmacs")

// KeyMapsItem is an entry in a KeyMaps list
type KeyMapsItem struct {

	// name of keymap
	Name string `width:"20"`

	// description of keymap -- good idea to include source it was derived from
	Desc string

	// to edit key sequence click button and type new key combination; to edit function mapped to key sequence choose from menu
	Map KeySeqMap
}

// Label satisfies the Labeler interface
func (km KeyMapsItem) Label() string {
	return km.Name
}

// KeyMaps is a list of KeyMap's -- users can edit these in Settings -- to create
// a custom one, just duplicate an existing map, rename, and customize
type KeyMaps []KeyMapsItem //types:add

// AvailableKeyMaps is the current list of available keymaps for use -- can be
// loaded / saved / edited with settings.  This is set to StdKeyMaps at
// startup.
var AvailableKeyMaps KeyMaps

func init() {
	AvailableKeyMaps.CopyFrom(StandardKeyMaps)
}

// MapByName returns a keymap and index by name -- returns false and emits a
// message to stdout if not found
func (km *KeyMaps) MapByName(name KeyMapName) (*KeySeqMap, int, bool) {
	for i, it := range *km {
		if it.Name == string(name) {
			return &it.Map, i, true
		}
	}
	fmt.Printf("core.KeyMaps.MapByName: key map named: %v not found\n", name)
	return nil, -1, false
}

// KeyMapSettingsFilename is the name of the settings file in the app settings
// directory for saving / loading the default AvailableKeyMaps key maps list
var KeyMapSettingsFilename = "key-map-settings.json"

// Open opens keymaps from a json-formatted file.
func (km *KeyMaps) Open(filename core.Filename) error { //types:add
	*km = make(KeyMaps, 0, 10) // reset
	return errors.Log(jsonx.Open(km, string(filename)))
}

// Save saves keymaps to a json-formatted file.
func (km *KeyMaps) Save(filename core.Filename) error { //types:add
	return errors.Log(jsonx.Save(km, string(filename)))
}

// OpenSettings opens the KeyMaps from the app settings directory,
// using KeyMapSettingsFilename.
func (km *KeyMaps) OpenSettings() error { //types:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, KeyMapSettingsFilename)
	AvailableKeyMapsChanged = false
	return km.Open(core.Filename(pnm))
}

// SaveSettings saves the KeyMaps to the app setttings directory, using
// KeyMapSettingsFilename.
func (km *KeyMaps) SaveSettings() error { //types:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, KeyMapSettingsFilename)
	AvailableKeyMapsChanged = false
	return km.Save(core.Filename(pnm))
}

// CopyFrom copies keymaps from given other map
func (km *KeyMaps) CopyFrom(cp KeyMaps) {
	*km = make(KeyMaps, 0, len(cp)) // reset
	b, err := json.Marshal(cp)
	if err != nil {
		fmt.Printf("json err: %v\n", err.Error())
	}
	json.Unmarshal(b, km)
}

// RevertToStandard reverts this map to using the StdKeyMaps that are compiled into
// the program and have all the lastest key functions bound to standard
// values.  If you have edited your maps, and are finding things not working,
// it is a good idea to save your current maps and try this,
// or at least do ViewStdMaps to see the current standards.
// <b>Your current map edits will be lost if you proceed!</b>
func (km *KeyMaps) RevertToStandard() { //types:add
	km.CopyFrom(StandardKeyMaps)
	AvailableKeyMapsChanged = true
}

// ViewStandard shows the standard maps that are compiled into the program and have
// all the lastest key functions bound to standard values.  Useful for
// comparing against custom maps.
func (km *KeyMaps) ViewStandard() { //types:add
	KeyMapsView(&StandardKeyMaps)
}

// AvailableKeyMapsChanged is used to update views.KeyMapsView toolbars via
// following menu, toolbar properties update methods -- not accurate if editing any
// other map but works for now..
var AvailableKeyMapsChanged = false

// StandardKeyMaps is the original compiled-in set of standard keymaps that have
// the lastest key functions bound to standard key chords.
var StandardKeyMaps = KeyMaps{
	{"MacStandard", "Standard Mac KeyMap", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyNextPanel,
		KeySeq{"Control+Shift+Tab", ""}:  KeyPrevPanel,
		KeySeq{"Control+M", "o"}:         KeyNextPanel,
		KeySeq{"Control+M", "Control+O"}: KeyNextPanel,
		KeySeq{"Control+M", "p"}:         KeyPrevPanel,
		KeySeq{"Control+M", "Control+P"}: KeyPrevPanel,
		KeySeq{"Control+O", ""}:          KeyFileOpen,
		KeySeq{"Control+M", "f"}:         KeyFileOpen,
		KeySeq{"Control+M", "Control+F"}: KeyFileOpen,
		KeySeq{"Control+M", "b"}:         KeyBufSelect,
		KeySeq{"Control+M", "Control+B"}: KeyBufSelect,
		KeySeq{"Control+S", ""}:          KeyBufSave,
		KeySeq{"Control+Shift+S", ""}:    KeyBufSaveAs,
		KeySeq{"Control+M", "s"}:         KeyBufSave,
		KeySeq{"Control+M", "Control+S"}: KeyBufSave,
		KeySeq{"Control+M", "w"}:         KeyBufSaveAs,
		KeySeq{"Control+M", "Control+W"}: KeyBufSaveAs,
		KeySeq{"Control+M", "k"}:         KeyBufClose,
		KeySeq{"Control+M", "Control+K"}: KeyBufClose,
		KeySeq{"Control+M", "c"}:         KeyExecCmd,
		KeySeq{"Control+M", "Control+C"}: KeyExecCmd,
		KeySeq{"Control+M", "n"}:         KeyBufClone,
		KeySeq{"Control+M", "Control+N"}: KeyBufClone,
		KeySeq{"Control+M", "x"}:         KeyRegCopy,
		KeySeq{"Control+M", "g"}:         KeyRegPaste,
		KeySeq{"Control+M", "Control+X"}: KeyRectCut,
		KeySeq{"Control+M", "Control+Y"}: KeyRectPaste,
		KeySeq{"Control+M", "Alt+∑"}:     KeyRectCopy,
		KeySeq{"Control+/", ""}:          KeyCommentOut,
		KeySeq{"Control+M", "t"}:         KeyCommentOut,
		KeySeq{"Control+M", "Control+T"}: KeyCommentOut,
		KeySeq{"Control+M", "i"}:         KeyIndent,
		KeySeq{"Control+M", "Control+I"}: KeyIndent,
		KeySeq{"Control+M", "j"}:         KeyJump,
		KeySeq{"Control+M", "Control+J"}: KeyJump,
		KeySeq{"Control+M", "v"}:         KeySetSplit,
		KeySeq{"Control+M", "Control+V"}: KeySetSplit,
		KeySeq{"Control+M", "m"}:         KeyBuildProject,
		KeySeq{"Control+M", "Control+M"}: KeyBuildProject,
		KeySeq{"Control+M", "r"}:         KeyRunProject,
		KeySeq{"Control+M", "Control+R"}: KeyRunProject,
	}},
	{"MacEmacs", "Mac with emacs-style navigation -- emacs wins in conflicts", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyNextPanel,
		KeySeq{"Control+Shift+Tab", ""}:  KeyPrevPanel,
		KeySeq{"Control+X", "o"}:         KeyNextPanel,
		KeySeq{"Control+X", "Control+O"}: KeyNextPanel,
		KeySeq{"Control+X", "p"}:         KeyPrevPanel,
		KeySeq{"Control+X", "Control+P"}: KeyPrevPanel,
		KeySeq{"Control+X", "f"}:         KeyFileOpen,
		KeySeq{"Control+X", "Control+F"}: KeyFileOpen,
		KeySeq{"Control+X", "b"}:         KeyBufSelect,
		KeySeq{"Control+X", "Control+B"}: KeyBufSelect,
		KeySeq{"Control+X", "s"}:         KeyBufSave,
		KeySeq{"Control+X", "Control+S"}: KeyBufSave,
		KeySeq{"Control+X", "w"}:         KeyBufSaveAs,
		KeySeq{"Control+X", "Control+W"}: KeyBufSaveAs,
		KeySeq{"Control+X", "k"}:         KeyBufClose,
		KeySeq{"Control+X", "Control+K"}: KeyBufClose,
		KeySeq{"Control+X", "c"}:         KeyExecCmd,
		KeySeq{"Control+X", "Control+C"}: KeyExecCmd,
		KeySeq{"Control+C", "c"}:         KeyExecCmd,
		KeySeq{"Control+C", "Control+C"}: KeyExecCmd,
		KeySeq{"Control+C", "o"}:         KeyBufClone,
		KeySeq{"Control+C", "Control+O"}: KeyBufClone,
		KeySeq{"Control+X", "x"}:         KeyRegCopy,
		KeySeq{"Control+X", "g"}:         KeyRegPaste,
		KeySeq{"Control+X", "Control+X"}: KeyRectCut,
		KeySeq{"Control+X", "Control+Y"}: KeyRectPaste,
		KeySeq{"Control+X", "Alt+∑"}:     KeyRectCopy,
		KeySeq{"Control+C", "k"}:         KeyCommentOut,
		KeySeq{"Control+C", "Control+K"}: KeyCommentOut,
		KeySeq{"Control+X", "i"}:         KeyIndent,
		KeySeq{"Control+X", "Control+I"}: KeyIndent,
		KeySeq{"Control+X", "j"}:         KeyJump,
		KeySeq{"Control+X", "Control+J"}: KeyJump,
		KeySeq{"Control+X", "v"}:         KeySetSplit,
		KeySeq{"Control+X", "Control+V"}: KeySetSplit,
		KeySeq{"Control+X", "m"}:         KeyBuildProject,
		KeySeq{"Control+X", "Control+M"}: KeyBuildProject,
		KeySeq{"Control+X", "r"}:         KeyRunProject,
		KeySeq{"Control+X", "Control+R"}: KeyRunProject,
	}},
	{"LinuxEmacs", "Linux with emacs-style navigation -- emacs wins in conflicts", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyNextPanel,
		KeySeq{"Control+Shift+Tab", ""}:  KeyPrevPanel,
		KeySeq{"Control+X", "o"}:         KeyNextPanel,
		KeySeq{"Control+X", "Control+O"}: KeyNextPanel,
		KeySeq{"Control+X", "p"}:         KeyPrevPanel,
		KeySeq{"Control+X", "Control+P"}: KeyPrevPanel,
		KeySeq{"Control+X", "f"}:         KeyFileOpen,
		KeySeq{"Control+X", "Control+F"}: KeyFileOpen,
		KeySeq{"Control+X", "b"}:         KeyBufSelect,
		KeySeq{"Control+X", "Control+B"}: KeyBufSelect,
		KeySeq{"Control+X", "s"}:         KeyBufSave,
		KeySeq{"Control+X", "Control+S"}: KeyBufSave,
		KeySeq{"Control+X", "w"}:         KeyBufSaveAs,
		KeySeq{"Control+X", "Control+W"}: KeyBufSaveAs,
		KeySeq{"Control+X", "k"}:         KeyBufClose,
		KeySeq{"Control+X", "Control+K"}: KeyBufClose,
		KeySeq{"Control+X", "c"}:         KeyExecCmd,
		KeySeq{"Control+X", "Control+C"}: KeyExecCmd,
		KeySeq{"Control+C", "c"}:         KeyExecCmd,
		KeySeq{"Control+C", "Control+C"}: KeyExecCmd,
		KeySeq{"Control+C", "o"}:         KeyBufClone,
		KeySeq{"Control+C", "Control+O"}: KeyBufClone,
		KeySeq{"Control+X", "x"}:         KeyRegCopy,
		KeySeq{"Control+X", "g"}:         KeyRegPaste,
		KeySeq{"Control+X", "Control+X"}: KeyRectCut,
		KeySeq{"Control+X", "Control+Y"}: KeyRectPaste,
		KeySeq{"Control+X", "Alt+∑"}:     KeyRectCopy,
		KeySeq{"Control+C", "k"}:         KeyCommentOut,
		KeySeq{"Control+C", "Control+K"}: KeyCommentOut,
		KeySeq{"Control+X", "i"}:         KeyIndent,
		KeySeq{"Control+X", "Control+I"}: KeyIndent,
		KeySeq{"Control+X", "j"}:         KeyJump,
		KeySeq{"Control+X", "Control+J"}: KeyJump,
		KeySeq{"Control+X", "v"}:         KeySetSplit,
		KeySeq{"Control+X", "Control+V"}: KeySetSplit,
		KeySeq{"Control+M", "m"}:         KeyBuildProject,
		KeySeq{"Control+M", "Control+M"}: KeyBuildProject,
		KeySeq{"Control+M", "r"}:         KeyRunProject,
		KeySeq{"Control+M", "Control+R"}: KeyRunProject,
	}},
	{"LinuxStandard", "Standard Linux KeySeqMap", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyNextPanel,
		KeySeq{"Control+Shift+Tab", ""}:  KeyPrevPanel,
		KeySeq{"Control+M", "o"}:         KeyNextPanel,
		KeySeq{"Control+M", "Control+O"}: KeyNextPanel,
		KeySeq{"Control+M", "p"}:         KeyPrevPanel,
		KeySeq{"Control+M", "Control+P"}: KeyPrevPanel,
		KeySeq{"Control+O", ""}:          KeyFileOpen,
		KeySeq{"Control+M", "f"}:         KeyFileOpen,
		KeySeq{"Control+M", "Control+F"}: KeyFileOpen,
		KeySeq{"Control+M", "b"}:         KeyBufSelect,
		KeySeq{"Control+M", "Control+B"}: KeyBufSelect,
		KeySeq{"Control+S", ""}:          KeyBufSave,
		KeySeq{"Control+Shift+S", ""}:    KeyBufSaveAs,
		KeySeq{"Control+M", "s"}:         KeyBufSave,
		KeySeq{"Control+M", "Control+S"}: KeyBufSave,
		KeySeq{"Control+M", "w"}:         KeyBufSaveAs,
		KeySeq{"Control+M", "Control+W"}: KeyBufSaveAs,
		KeySeq{"Control+M", "k"}:         KeyBufClose,
		KeySeq{"Control+M", "Control+K"}: KeyBufClose,
		KeySeq{"Control+M", "c"}:         KeyExecCmd,
		KeySeq{"Control+M", "Control+C"}: KeyExecCmd,
		KeySeq{"Control+M", "n"}:         KeyBufClone,
		KeySeq{"Control+M", "Control+N"}: KeyBufClone,
		KeySeq{"Control+M", "x"}:         KeyRegCopy,
		KeySeq{"Control+M", "g"}:         KeyRegPaste,
		KeySeq{"Control+M", "Control+X"}: KeyRectCut,
		KeySeq{"Control+M", "Control+Y"}: KeyRectPaste,
		KeySeq{"Control+M", "Alt+∑"}:     KeyRectCopy,
		KeySeq{"Control+/", ""}:          KeyCommentOut,
		KeySeq{"Control+M", "t"}:         KeyCommentOut,
		KeySeq{"Control+M", "Control+T"}: KeyCommentOut,
		KeySeq{"Control+M", "i"}:         KeyIndent,
		KeySeq{"Control+M", "Control+I"}: KeyIndent,
		KeySeq{"Control+M", "j"}:         KeyJump,
		KeySeq{"Control+M", "Control+J"}: KeyJump,
		KeySeq{"Control+M", "v"}:         KeySetSplit,
		KeySeq{"Control+M", "Control+V"}: KeySetSplit,
		KeySeq{"Control+M", "m"}:         KeyBuildProject,
		KeySeq{"Control+M", "Control+M"}: KeyBuildProject,
		KeySeq{"Control+M", "r"}:         KeyRunProject,
		KeySeq{"Control+M", "Control+R"}: KeyRunProject,
	}},
	{"WindowsStandard", "Standard Windows KeySeqMap", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyNextPanel,
		KeySeq{"Control+Shift+Tab", ""}:  KeyPrevPanel,
		KeySeq{"Control+M", "o"}:         KeyNextPanel,
		KeySeq{"Control+M", "Control+O"}: KeyNextPanel,
		KeySeq{"Control+M", "p"}:         KeyPrevPanel,
		KeySeq{"Control+M", "Control+P"}: KeyPrevPanel,
		KeySeq{"Control+O", ""}:          KeyFileOpen,
		KeySeq{"Control+M", "f"}:         KeyFileOpen,
		KeySeq{"Control+M", "Control+F"}: KeyFileOpen,
		KeySeq{"Control+M", "b"}:         KeyBufSelect,
		KeySeq{"Control+M", "Control+B"}: KeyBufSelect,
		KeySeq{"Control+S", ""}:          KeyBufSave,
		KeySeq{"Control+Shift+S", ""}:    KeyBufSaveAs,
		KeySeq{"Control+M", "s"}:         KeyBufSave,
		KeySeq{"Control+M", "Control+S"}: KeyBufSave,
		KeySeq{"Control+M", "w"}:         KeyBufSaveAs,
		KeySeq{"Control+M", "Control+W"}: KeyBufSaveAs,
		KeySeq{"Control+M", "k"}:         KeyBufClose,
		KeySeq{"Control+M", "Control+K"}: KeyBufClose,
		KeySeq{"Control+M", "c"}:         KeyExecCmd,
		KeySeq{"Control+M", "Control+C"}: KeyExecCmd,
		KeySeq{"Control+M", "n"}:         KeyBufClone,
		KeySeq{"Control+M", "Control+N"}: KeyBufClone,
		KeySeq{"Control+M", "x"}:         KeyRegCopy,
		KeySeq{"Control+M", "g"}:         KeyRegPaste,
		KeySeq{"Control+M", "Control+X"}: KeyRectCut,
		KeySeq{"Control+M", "Control+Y"}: KeyRectPaste,
		KeySeq{"Control+M", "Alt+∑"}:     KeyRectCopy,
		KeySeq{"Control+/", ""}:          KeyCommentOut,
		KeySeq{"Control+M", "t"}:         KeyCommentOut,
		KeySeq{"Control+M", "Control+T"}: KeyCommentOut,
		KeySeq{"Control+M", "i"}:         KeyIndent,
		KeySeq{"Control+M", "Control+I"}: KeyIndent,
		KeySeq{"Control+M", "j"}:         KeyJump,
		KeySeq{"Control+M", "Control+J"}: KeyJump,
		KeySeq{"Control+M", "v"}:         KeySetSplit,
		KeySeq{"Control+M", "Control+V"}: KeySetSplit,
		KeySeq{"Control+M", "m"}:         KeyBuildProject,
		KeySeq{"Control+M", "Control+M"}: KeyBuildProject,
		KeySeq{"Control+M", "r"}:         KeyRunProject,
		KeySeq{"Control+M", "Control+R"}: KeyRunProject,
	}},
	{"ChromeStd", "Standard chrome-browser and linux-under-chrome bindings", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyNextPanel,
		KeySeq{"Control+Shift+Tab", ""}:  KeyPrevPanel,
		KeySeq{"Control+M", "o"}:         KeyNextPanel,
		KeySeq{"Control+M", "Control+O"}: KeyNextPanel,
		KeySeq{"Control+M", "p"}:         KeyPrevPanel,
		KeySeq{"Control+M", "Control+P"}: KeyPrevPanel,
		KeySeq{"Control+O", ""}:          KeyFileOpen,
		KeySeq{"Control+M", "f"}:         KeyFileOpen,
		KeySeq{"Control+M", "Control+F"}: KeyFileOpen,
		KeySeq{"Control+M", "b"}:         KeyBufSelect,
		KeySeq{"Control+M", "Control+B"}: KeyBufSelect,
		KeySeq{"Control+S", ""}:          KeyBufSave,
		KeySeq{"Control+Shift+S", ""}:    KeyBufSaveAs,
		KeySeq{"Control+M", "s"}:         KeyBufSave,
		KeySeq{"Control+M", "Control+S"}: KeyBufSave,
		KeySeq{"Control+M", "w"}:         KeyBufSaveAs,
		KeySeq{"Control+M", "Control+W"}: KeyBufSaveAs,
		KeySeq{"Control+M", "k"}:         KeyBufClose,
		KeySeq{"Control+M", "Control+K"}: KeyBufClose,
		KeySeq{"Control+M", "c"}:         KeyExecCmd,
		KeySeq{"Control+M", "Control+C"}: KeyExecCmd,
		KeySeq{"Control+M", "n"}:         KeyBufClone,
		KeySeq{"Control+M", "Control+N"}: KeyBufClone,
		KeySeq{"Control+M", "x"}:         KeyRegCopy,
		KeySeq{"Control+M", "g"}:         KeyRegPaste,
		KeySeq{"Control+M", "Control+X"}: KeyRectCut,
		KeySeq{"Control+M", "Control+Y"}: KeyRectPaste,
		KeySeq{"Control+M", "Alt+∑"}:     KeyRectCopy,
		KeySeq{"Control+/", ""}:          KeyCommentOut,
		KeySeq{"Control+M", "t"}:         KeyCommentOut,
		KeySeq{"Control+M", "Control+T"}: KeyCommentOut,
		KeySeq{"Control+M", "i"}:         KeyIndent,
		KeySeq{"Control+M", "Control+I"}: KeyIndent,
		KeySeq{"Control+M", "j"}:         KeyJump,
		KeySeq{"Control+M", "Control+J"}: KeyJump,
		KeySeq{"Control+M", "v"}:         KeySetSplit,
		KeySeq{"Control+M", "Control+V"}: KeySetSplit,
		KeySeq{"Control+M", "m"}:         KeyBuildProject,
		KeySeq{"Control+M", "Control+M"}: KeyBuildProject,
		KeySeq{"Control+M", "r"}:         KeyRunProject,
		KeySeq{"Control+M", "Control+R"}: KeyRunProject,
	}},
}
