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

	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/iox/jsonx"
	"cogentcore.org/core/keyfun"
)

// https://www.eclipse.org/pdt/help/html/keymap.htm
// https://resources.jetbrains.com/storage/products/rubymine/docs/RubyMine_ReferenceCard.pdf
// https://docs.microsoft.com/en-us/visualstudio/ide/default-keyboard-shortcuts-in-visual-studio?view=vs-2017
// https://swifteducation.github.io/assets/pdfs/XcodeKeyboardShortcuts.pdf
// https://en.wikipedia.org/wiki/Table_of_keyboard_shortcuts <- great!

// KeyFuns (i.e. code.KeytFuns) are special functions for the overall control of the system --
// moving between windows, running commands, etc.  Multi-key sequences can be used.
type KeyFuns int32 //enums:enum -trim-prefix KeyFun

const (
	KeyFunNil KeyFuns = iota
	// special internal signal returned by KeyFun indicating need for second key
	KeyFunNeeds2
	// move to next panel to the right
	KeyFunNextPanel
	// move to prev panel to the left
	KeyFunPrevPanel
	// open a new file in active texteditor
	KeyFunFileOpen
	// select an open buffer to edit in active texteditor
	KeyFunBufSelect
	// open active file in other view
	KeyFunBufClone
	// save active texteditor buffer to its file
	KeyFunBufSave
	// save as active texteditor buffer to its file
	KeyFunBufSaveAs
	// close active texteditor buffer
	KeyFunBufClose
	// execute a command on active texteditor buffer
	KeyFunExecCmd
	// copy rectangle
	KeyFunRectCopy
	// cut rectangle
	KeyFunRectCut
	// paste rectangle
	KeyFunRectPaste
	// copy selection to named register
	KeyFunRegCopy
	// paste selection from named register
	KeyFunRegPaste
	// comment out region
	KeyFunCommentOut
	// indent region
	KeyFunIndent
	// jump to line (same as keyfun.Jump)
	KeyFunJump
	// set named splitter config
	KeyFunSetSplit
	// build overall project
	KeyFunBuildProj
	// run overall project
	KeyFunRunProj
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
// specific KeyFun function.  This mapping must be unique, in that each chord
// has unique KeyFun, but multiple chords can trigger the same function.
type KeySeqMap map[KeySeq]KeyFuns

// ActiveKeyMap points to the active map -- users can set this to an
// alternative map in Settings
var ActiveKeyMap *KeySeqMap

// ActiveKeyMapName is the name of the active keymap
var ActiveKeyMapName KeyMapName

// Needs2KeyMap is a map of the starting key sequences that require a second
// key -- auto-generated from active keymap
var Needs2KeyMap keyfun.Map

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

// KeyFun translates chord into keyboard function -- use oswin key.Chord to
// get chord -- it returns KeyFunNeeds2 if the key sequence requires 2 keys to
// be entered, and only the first is present
func KeyFun(key1, key2 key.Chord) KeyFuns {
	kf := KeyFunNil
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
			return KeyFunNeeds2
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
	Fun KeyFuns
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

// ChordForFun returns first key sequence trigger for given KeyFun in map
func (km *KeySeqMap) ChordForFun(kf KeyFuns) KeySeq {
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

// ChordForFun returns first key sequence trigger for given KeyFun in ActiveKeyMap
func ChordForFun(kf KeyFuns) KeySeq {
	return ActiveKeyMap.ChordForFun(kf)
}

// Update ensures that the given keymap has at least one entry for every
// defined KeyFun, grabbing ones from the default map if not, and also
// eliminates any Nil entries which might reflect out-of-date functions
func (km *KeySeqMap) Update(kmName KeyMapName) {
	for key, val := range *km {
		if val == KeyFunNil {
			log.Printf("code.KeySeqMap: key function is nil -- probably renamed, for key: %v\n", key)
			delete(*km, key)
		}
	}
	kms := km.ToSlice()
	addkm := make([]KeyMapItem, 0)

	sort.Slice(kms, func(i, j int) bool {
		return kms[i].Fun < kms[j].Fun
	})

	lfun := KeyFunNeeds2
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
	Needs2KeyMap = make(keyfun.Map)

	for key := range *km {
		if key.Key2 != "" {
			Needs2KeyMap[key.Key1] = keyfun.Nil
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
type KeyMaps []KeyMapsItem //gti:add

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
func (km *KeyMaps) Open(filename core.Filename) error { //gti:add
	*km = make(KeyMaps, 0, 10) // reset
	return errors.Log(jsonx.Open(km, string(filename)))
}

// Save saves keymaps to a json-formatted file.
func (km *KeyMaps) Save(filename core.Filename) error { //gti:add
	return errors.Log(jsonx.Save(km, string(filename)))
}

// OpenSettings opens the KeyMaps from the app settings directory,
// using KeyMapSettingsFilename.
func (km *KeyMaps) OpenSettings() error { //gti:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, KeyMapSettingsFilename)
	AvailableKeyMapsChanged = false
	return km.Open(core.Filename(pnm))
}

// SaveSettings saves the KeyMaps to the app setttings directory, using
// KeyMapSettingsFilename.
func (km *KeyMaps) SaveSettings() error { //gti:add
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
func (km *KeyMaps) RevertToStandard() { //gti:add
	km.CopyFrom(StandardKeyMaps)
	AvailableKeyMapsChanged = true
}

// ViewStandard shows the standard maps that are compiled into the program and have
// all the lastest key functions bound to standard values.  Useful for
// comparing against custom maps.
func (km *KeyMaps) ViewStandard() { //gti:add
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
		KeySeq{"Control+Tab", ""}:        KeyFunNextPanel,
		KeySeq{"Shift+Control+Tab", ""}:  KeyFunPrevPanel,
		KeySeq{"Control+M", "o"}:         KeyFunNextPanel,
		KeySeq{"Control+M", "Control+O"}: KeyFunNextPanel,
		KeySeq{"Control+M", "p"}:         KeyFunPrevPanel,
		KeySeq{"Control+M", "Control+P"}: KeyFunPrevPanel,
		KeySeq{"Control+O", ""}:          KeyFunFileOpen,
		KeySeq{"Control+M", "f"}:         KeyFunFileOpen,
		KeySeq{"Control+M", "Control+F"}: KeyFunFileOpen,
		KeySeq{"Control+M", "b"}:         KeyFunBufSelect,
		KeySeq{"Control+M", "Control+B"}: KeyFunBufSelect,
		KeySeq{"Control+S", ""}:          KeyFunBufSave,
		KeySeq{"Shift+Control+S", ""}:    KeyFunBufSaveAs,
		KeySeq{"Control+M", "s"}:         KeyFunBufSave,
		KeySeq{"Control+M", "Control+S"}: KeyFunBufSave,
		KeySeq{"Control+M", "w"}:         KeyFunBufSaveAs,
		KeySeq{"Control+M", "Control+W"}: KeyFunBufSaveAs,
		KeySeq{"Control+M", "k"}:         KeyFunBufClose,
		KeySeq{"Control+M", "Control+K"}: KeyFunBufClose,
		KeySeq{"Control+M", "c"}:         KeyFunExecCmd,
		KeySeq{"Control+M", "Control+C"}: KeyFunExecCmd,
		KeySeq{"Control+M", "n"}:         KeyFunBufClone,
		KeySeq{"Control+M", "Control+N"}: KeyFunBufClone,
		KeySeq{"Control+M", "x"}:         KeyFunRegCopy,
		KeySeq{"Control+M", "g"}:         KeyFunRegPaste,
		KeySeq{"Control+M", "Control+X"}: KeyFunRectCut,
		KeySeq{"Control+M", "Control+Y"}: KeyFunRectPaste,
		KeySeq{"Control+M", "Alt+∑"}:     KeyFunRectCopy,
		KeySeq{"Control+/", ""}:          KeyFunCommentOut,
		KeySeq{"Control+M", "t"}:         KeyFunCommentOut,
		KeySeq{"Control+M", "Control+T"}: KeyFunCommentOut,
		KeySeq{"Control+M", "i"}:         KeyFunIndent,
		KeySeq{"Control+M", "Control+I"}: KeyFunIndent,
		KeySeq{"Control+M", "j"}:         KeyFunJump,
		KeySeq{"Control+M", "Control+J"}: KeyFunJump,
		KeySeq{"Control+M", "v"}:         KeyFunSetSplit,
		KeySeq{"Control+M", "Control+V"}: KeyFunSetSplit,
		KeySeq{"Control+M", "m"}:         KeyFunBuildProj,
		KeySeq{"Control+M", "Control+M"}: KeyFunBuildProj,
		KeySeq{"Control+M", "r"}:         KeyFunRunProj,
		KeySeq{"Control+M", "Control+R"}: KeyFunRunProj,
	}},
	{"MacEmacs", "Mac with emacs-style navigation -- emacs wins in conflicts", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyFunNextPanel,
		KeySeq{"Shift+Control+Tab", ""}:  KeyFunPrevPanel,
		KeySeq{"Control+X", "o"}:         KeyFunNextPanel,
		KeySeq{"Control+X", "Control+O"}: KeyFunNextPanel,
		KeySeq{"Control+X", "p"}:         KeyFunPrevPanel,
		KeySeq{"Control+X", "Control+P"}: KeyFunPrevPanel,
		KeySeq{"Control+X", "f"}:         KeyFunFileOpen,
		KeySeq{"Control+X", "Control+F"}: KeyFunFileOpen,
		KeySeq{"Control+X", "b"}:         KeyFunBufSelect,
		KeySeq{"Control+X", "Control+B"}: KeyFunBufSelect,
		KeySeq{"Control+X", "s"}:         KeyFunBufSave,
		KeySeq{"Control+X", "Control+S"}: KeyFunBufSave,
		KeySeq{"Control+X", "w"}:         KeyFunBufSaveAs,
		KeySeq{"Control+X", "Control+W"}: KeyFunBufSaveAs,
		KeySeq{"Control+X", "k"}:         KeyFunBufClose,
		KeySeq{"Control+X", "Control+K"}: KeyFunBufClose,
		KeySeq{"Control+X", "c"}:         KeyFunExecCmd,
		KeySeq{"Control+X", "Control+C"}: KeyFunExecCmd,
		KeySeq{"Control+C", "c"}:         KeyFunExecCmd,
		KeySeq{"Control+C", "Control+C"}: KeyFunExecCmd,
		KeySeq{"Control+C", "o"}:         KeyFunBufClone,
		KeySeq{"Control+C", "Control+O"}: KeyFunBufClone,
		KeySeq{"Control+X", "x"}:         KeyFunRegCopy,
		KeySeq{"Control+X", "g"}:         KeyFunRegPaste,
		KeySeq{"Control+X", "Control+X"}: KeyFunRectCut,
		KeySeq{"Control+X", "Control+Y"}: KeyFunRectPaste,
		KeySeq{"Control+X", "Alt+∑"}:     KeyFunRectCopy,
		KeySeq{"Control+C", "k"}:         KeyFunCommentOut,
		KeySeq{"Control+C", "Control+K"}: KeyFunCommentOut,
		KeySeq{"Control+X", "i"}:         KeyFunIndent,
		KeySeq{"Control+X", "Control+I"}: KeyFunIndent,
		KeySeq{"Control+X", "j"}:         KeyFunJump,
		KeySeq{"Control+X", "Control+J"}: KeyFunJump,
		KeySeq{"Control+X", "v"}:         KeyFunSetSplit,
		KeySeq{"Control+X", "Control+V"}: KeyFunSetSplit,
		KeySeq{"Control+X", "m"}:         KeyFunBuildProj,
		KeySeq{"Control+X", "Control+M"}: KeyFunBuildProj,
		KeySeq{"Control+X", "r"}:         KeyFunRunProj,
		KeySeq{"Control+X", "Control+R"}: KeyFunRunProj,
	}},
	{"LinuxEmacs", "Linux with emacs-style navigation -- emacs wins in conflicts", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyFunNextPanel,
		KeySeq{"Shift+Control+Tab", ""}:  KeyFunPrevPanel,
		KeySeq{"Control+X", "o"}:         KeyFunNextPanel,
		KeySeq{"Control+X", "Control+O"}: KeyFunNextPanel,
		KeySeq{"Control+X", "p"}:         KeyFunPrevPanel,
		KeySeq{"Control+X", "Control+P"}: KeyFunPrevPanel,
		KeySeq{"Control+X", "f"}:         KeyFunFileOpen,
		KeySeq{"Control+X", "Control+F"}: KeyFunFileOpen,
		KeySeq{"Control+X", "b"}:         KeyFunBufSelect,
		KeySeq{"Control+X", "Control+B"}: KeyFunBufSelect,
		KeySeq{"Control+X", "s"}:         KeyFunBufSave,
		KeySeq{"Control+X", "Control+S"}: KeyFunBufSave,
		KeySeq{"Control+X", "w"}:         KeyFunBufSaveAs,
		KeySeq{"Control+X", "Control+W"}: KeyFunBufSaveAs,
		KeySeq{"Control+X", "k"}:         KeyFunBufClose,
		KeySeq{"Control+X", "Control+K"}: KeyFunBufClose,
		KeySeq{"Control+X", "c"}:         KeyFunExecCmd,
		KeySeq{"Control+X", "Control+C"}: KeyFunExecCmd,
		KeySeq{"Control+C", "c"}:         KeyFunExecCmd,
		KeySeq{"Control+C", "Control+C"}: KeyFunExecCmd,
		KeySeq{"Control+C", "o"}:         KeyFunBufClone,
		KeySeq{"Control+C", "Control+O"}: KeyFunBufClone,
		KeySeq{"Control+X", "x"}:         KeyFunRegCopy,
		KeySeq{"Control+X", "g"}:         KeyFunRegPaste,
		KeySeq{"Control+X", "Control+X"}: KeyFunRectCut,
		KeySeq{"Control+X", "Control+Y"}: KeyFunRectPaste,
		KeySeq{"Control+X", "Alt+∑"}:     KeyFunRectCopy,
		KeySeq{"Control+C", "k"}:         KeyFunCommentOut,
		KeySeq{"Control+C", "Control+K"}: KeyFunCommentOut,
		KeySeq{"Control+X", "i"}:         KeyFunIndent,
		KeySeq{"Control+X", "Control+I"}: KeyFunIndent,
		KeySeq{"Control+X", "j"}:         KeyFunJump,
		KeySeq{"Control+X", "Control+J"}: KeyFunJump,
		KeySeq{"Control+X", "v"}:         KeyFunSetSplit,
		KeySeq{"Control+X", "Control+V"}: KeyFunSetSplit,
		KeySeq{"Control+M", "m"}:         KeyFunBuildProj,
		KeySeq{"Control+M", "Control+M"}: KeyFunBuildProj,
		KeySeq{"Control+M", "r"}:         KeyFunRunProj,
		KeySeq{"Control+M", "Control+R"}: KeyFunRunProj,
	}},
	{"LinuxStandard", "Standard Linux KeySeqMap", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyFunNextPanel,
		KeySeq{"Shift+Control+Tab", ""}:  KeyFunPrevPanel,
		KeySeq{"Control+M", "o"}:         KeyFunNextPanel,
		KeySeq{"Control+M", "Control+O"}: KeyFunNextPanel,
		KeySeq{"Control+M", "p"}:         KeyFunPrevPanel,
		KeySeq{"Control+M", "Control+P"}: KeyFunPrevPanel,
		KeySeq{"Control+O", ""}:          KeyFunFileOpen,
		KeySeq{"Control+M", "f"}:         KeyFunFileOpen,
		KeySeq{"Control+M", "Control+F"}: KeyFunFileOpen,
		KeySeq{"Control+M", "b"}:         KeyFunBufSelect,
		KeySeq{"Control+M", "Control+B"}: KeyFunBufSelect,
		KeySeq{"Control+S", ""}:          KeyFunBufSave,
		KeySeq{"Shift+Control+S", ""}:    KeyFunBufSaveAs,
		KeySeq{"Control+M", "s"}:         KeyFunBufSave,
		KeySeq{"Control+M", "Control+S"}: KeyFunBufSave,
		KeySeq{"Control+M", "w"}:         KeyFunBufSaveAs,
		KeySeq{"Control+M", "Control+W"}: KeyFunBufSaveAs,
		KeySeq{"Control+M", "k"}:         KeyFunBufClose,
		KeySeq{"Control+M", "Control+K"}: KeyFunBufClose,
		KeySeq{"Control+M", "c"}:         KeyFunExecCmd,
		KeySeq{"Control+M", "Control+C"}: KeyFunExecCmd,
		KeySeq{"Control+M", "n"}:         KeyFunBufClone,
		KeySeq{"Control+M", "Control+N"}: KeyFunBufClone,
		KeySeq{"Control+M", "x"}:         KeyFunRegCopy,
		KeySeq{"Control+M", "g"}:         KeyFunRegPaste,
		KeySeq{"Control+M", "Control+X"}: KeyFunRectCut,
		KeySeq{"Control+M", "Control+Y"}: KeyFunRectPaste,
		KeySeq{"Control+M", "Alt+∑"}:     KeyFunRectCopy,
		KeySeq{"Control+/", ""}:          KeyFunCommentOut,
		KeySeq{"Control+M", "t"}:         KeyFunCommentOut,
		KeySeq{"Control+M", "Control+T"}: KeyFunCommentOut,
		KeySeq{"Control+M", "i"}:         KeyFunIndent,
		KeySeq{"Control+M", "Control+I"}: KeyFunIndent,
		KeySeq{"Control+M", "j"}:         KeyFunJump,
		KeySeq{"Control+M", "Control+J"}: KeyFunJump,
		KeySeq{"Control+M", "v"}:         KeyFunSetSplit,
		KeySeq{"Control+M", "Control+V"}: KeyFunSetSplit,
		KeySeq{"Control+M", "m"}:         KeyFunBuildProj,
		KeySeq{"Control+M", "Control+M"}: KeyFunBuildProj,
		KeySeq{"Control+M", "r"}:         KeyFunRunProj,
		KeySeq{"Control+M", "Control+R"}: KeyFunRunProj,
	}},
	{"WindowsStandard", "Standard Windows KeySeqMap", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyFunNextPanel,
		KeySeq{"Shift+Control+Tab", ""}:  KeyFunPrevPanel,
		KeySeq{"Control+M", "o"}:         KeyFunNextPanel,
		KeySeq{"Control+M", "Control+O"}: KeyFunNextPanel,
		KeySeq{"Control+M", "p"}:         KeyFunPrevPanel,
		KeySeq{"Control+M", "Control+P"}: KeyFunPrevPanel,
		KeySeq{"Control+O", ""}:          KeyFunFileOpen,
		KeySeq{"Control+M", "f"}:         KeyFunFileOpen,
		KeySeq{"Control+M", "Control+F"}: KeyFunFileOpen,
		KeySeq{"Control+M", "b"}:         KeyFunBufSelect,
		KeySeq{"Control+M", "Control+B"}: KeyFunBufSelect,
		KeySeq{"Control+S", ""}:          KeyFunBufSave,
		KeySeq{"Shift+Control+S", ""}:    KeyFunBufSaveAs,
		KeySeq{"Control+M", "s"}:         KeyFunBufSave,
		KeySeq{"Control+M", "Control+S"}: KeyFunBufSave,
		KeySeq{"Control+M", "w"}:         KeyFunBufSaveAs,
		KeySeq{"Control+M", "Control+W"}: KeyFunBufSaveAs,
		KeySeq{"Control+M", "k"}:         KeyFunBufClose,
		KeySeq{"Control+M", "Control+K"}: KeyFunBufClose,
		KeySeq{"Control+M", "c"}:         KeyFunExecCmd,
		KeySeq{"Control+M", "Control+C"}: KeyFunExecCmd,
		KeySeq{"Control+M", "n"}:         KeyFunBufClone,
		KeySeq{"Control+M", "Control+N"}: KeyFunBufClone,
		KeySeq{"Control+M", "x"}:         KeyFunRegCopy,
		KeySeq{"Control+M", "g"}:         KeyFunRegPaste,
		KeySeq{"Control+M", "Control+X"}: KeyFunRectCut,
		KeySeq{"Control+M", "Control+Y"}: KeyFunRectPaste,
		KeySeq{"Control+M", "Alt+∑"}:     KeyFunRectCopy,
		KeySeq{"Control+/", ""}:          KeyFunCommentOut,
		KeySeq{"Control+M", "t"}:         KeyFunCommentOut,
		KeySeq{"Control+M", "Control+T"}: KeyFunCommentOut,
		KeySeq{"Control+M", "i"}:         KeyFunIndent,
		KeySeq{"Control+M", "Control+I"}: KeyFunIndent,
		KeySeq{"Control+M", "j"}:         KeyFunJump,
		KeySeq{"Control+M", "Control+J"}: KeyFunJump,
		KeySeq{"Control+M", "v"}:         KeyFunSetSplit,
		KeySeq{"Control+M", "Control+V"}: KeyFunSetSplit,
		KeySeq{"Control+M", "m"}:         KeyFunBuildProj,
		KeySeq{"Control+M", "Control+M"}: KeyFunBuildProj,
		KeySeq{"Control+M", "r"}:         KeyFunRunProj,
		KeySeq{"Control+M", "Control+R"}: KeyFunRunProj,
	}},
	{"ChromeStd", "Standard chrome-browser and linux-under-chrome bindings", KeySeqMap{
		KeySeq{"Control+Tab", ""}:        KeyFunNextPanel,
		KeySeq{"Shift+Control+Tab", ""}:  KeyFunPrevPanel,
		KeySeq{"Control+M", "o"}:         KeyFunNextPanel,
		KeySeq{"Control+M", "Control+O"}: KeyFunNextPanel,
		KeySeq{"Control+M", "p"}:         KeyFunPrevPanel,
		KeySeq{"Control+M", "Control+P"}: KeyFunPrevPanel,
		KeySeq{"Control+O", ""}:          KeyFunFileOpen,
		KeySeq{"Control+M", "f"}:         KeyFunFileOpen,
		KeySeq{"Control+M", "Control+F"}: KeyFunFileOpen,
		KeySeq{"Control+M", "b"}:         KeyFunBufSelect,
		KeySeq{"Control+M", "Control+B"}: KeyFunBufSelect,
		KeySeq{"Control+S", ""}:          KeyFunBufSave,
		KeySeq{"Shift+Control+S", ""}:    KeyFunBufSaveAs,
		KeySeq{"Control+M", "s"}:         KeyFunBufSave,
		KeySeq{"Control+M", "Control+S"}: KeyFunBufSave,
		KeySeq{"Control+M", "w"}:         KeyFunBufSaveAs,
		KeySeq{"Control+M", "Control+W"}: KeyFunBufSaveAs,
		KeySeq{"Control+M", "k"}:         KeyFunBufClose,
		KeySeq{"Control+M", "Control+K"}: KeyFunBufClose,
		KeySeq{"Control+M", "c"}:         KeyFunExecCmd,
		KeySeq{"Control+M", "Control+C"}: KeyFunExecCmd,
		KeySeq{"Control+M", "n"}:         KeyFunBufClone,
		KeySeq{"Control+M", "Control+N"}: KeyFunBufClone,
		KeySeq{"Control+M", "x"}:         KeyFunRegCopy,
		KeySeq{"Control+M", "g"}:         KeyFunRegPaste,
		KeySeq{"Control+M", "Control+X"}: KeyFunRectCut,
		KeySeq{"Control+M", "Control+Y"}: KeyFunRectPaste,
		KeySeq{"Control+M", "Alt+∑"}:     KeyFunRectCopy,
		KeySeq{"Control+/", ""}:          KeyFunCommentOut,
		KeySeq{"Control+M", "t"}:         KeyFunCommentOut,
		KeySeq{"Control+M", "Control+T"}: KeyFunCommentOut,
		KeySeq{"Control+M", "i"}:         KeyFunIndent,
		KeySeq{"Control+M", "Control+I"}: KeyFunIndent,
		KeySeq{"Control+M", "j"}:         KeyFunJump,
		KeySeq{"Control+M", "Control+J"}: KeyFunJump,
		KeySeq{"Control+M", "v"}:         KeyFunSetSplit,
		KeySeq{"Control+M", "Control+V"}: KeyFunSetSplit,
		KeySeq{"Control+M", "m"}:         KeyFunBuildProj,
		KeySeq{"Control+M", "Control+M"}: KeyFunBuildProj,
		KeySeq{"Control+M", "r"}:         KeyFunRunProj,
		KeySeq{"Control+M", "Control+R"}: KeyFunRunProj,
	}},
}
