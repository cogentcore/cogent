// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// https://www.eclipse.org/pdt/help/html/keymap.htm
// https://resources.jetbrains.com/storage/products/rubymine/docs/RubyMine_ReferenceCard.pdf
// https://docs.microsoft.com/en-us/visualstudio/ide/default-keyboard-shortcuts-in-visual-studio?view=vs-2017
// https://swifteducation.github.io/assets/pdfs/XcodeKeyboardShortcuts.pdf
// https://en.wikipedia.org/wiki/Table_of_keyboard_shortcuts <- great!

// KeyFuns (i.e. gide.KeytFuns) are special functions for the overall control of the system --
// moving between windows, running commands, etc.  Multi-key sequences can be used.
type KeyFuns int32

const (
	KeyFunNil        KeyFuns = iota
	KeyFunNeeds2             // special internal signal returned by KeyFun indicating need for second key
	KeyFunNextPanel          // move to next panel to the right
	KeyFunPrevPanel          // move to prev panel to the left
	KeyFunFileOpen           // open a new file in active textview
	KeyFunBufSelect          // select an open buffer to edit in active textview
	KeyFunBufClone           // open active file in other view
	KeyFunBufSave            // save active textview buffer to its file
	KeyFunBufSaveAs          // save as active textview buffer to its file
	KeyFunBufClose           // close active textview buffer
	KeyFunExecCmd            // execute a command on active textview buffer
	KeyFunRegCopy            // copy selection to named register
	KeyFunRegPaste           // paste selection from named register
	KeyFunCommentOut         // comment out region
	KeyFunIndent             // indent region
	KeyFunJump               // jump to line (same as gi.KeyFunJump)
	KeyFunSetSplit           // set named splitter config
	KeyFunBuildProj          // build overall project
	KeyFunRunProj            // run overall project
	KeyFunsN
)

//go:generate stringer -type=KeyFuns

// KiT_KeyFuns adds a type to the EnumRegistry
var KiT_KeyFuns = kit.Enums.AddEnumAltLower(KeyFunsN, kit.NotBitFlag, nil, "KeyFun")

// MarshalJSON saves the KeyFuns in JSON format
func (kf KeyFuns) MarshalJSON() ([]byte, error) { return kit.EnumMarshalJSON(kf) }

// UnmarshalJSON reads the JSON formatted KeyFun info from file and loads into memory
func (kf *KeyFuns) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(kf, b) }

// KeySeq defines a multiple-key sequence to initiate a key function
type KeySeq struct {
	Key1 key.Chord // first key
	Key2 key.Chord // second key (optional)
}

// String() satisfies fmt.Stringer interface
func (kf KeySeq) String() string {
	return string(kf.Key1 + " " + kf.Key2)
}

// Label satisfies gi.Labeler interface
func (kf KeySeq) Label() string {
	return string(kf.Key1 + " " + kf.Key2)
}

// MarshalText is required for JSON encoding of struct keys
func (kf KeySeq) MarshalText() ([]byte, error) {
	bs := make([][]byte, 2)
	bs[0] = []byte(kf.Key1)
	bs[1] = []byte(kf.Key2)
	b := bytes.Join(bs, []byte(";"))
	return b, nil
}

// UnmarshalText is required for JSON decoding of struct keys
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
// alternative map in Prefs
var ActiveKeyMap *KeySeqMap

// ActiveKeyMapName is the name of the active keymap
var ActiveKeyMapName KeyMapName

// Needs2KeyMap is a map of the starting key sequences that require a second
// key -- auto-generated from active keymap
var Needs2KeyMap gi.KeyMap

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
	km, _, ok := AvailKeyMaps.MapByName(mapnm)
	if ok {
		SetActiveKeyMap(km, mapnm)
	} else {
		log.Printf("gide.SetActiveKeyMapName: key map named: %v not found, using default: %v\n", mapnm, DefaultKeyMap)
		km, _, ok = AvailKeyMaps.MapByName(DefaultKeyMap)
		if ok {
			SetActiveKeyMap(km, DefaultKeyMap)
		} else {
			log.Printf("gide.SetActiveKeyMapName: ok, this is bad: DefaultKeyMap not found either -- size of AvailKeyMaps: %v -- trying first one\n", len(AvailKeyMaps))
			if len(AvailKeyMaps) > 0 {
				skm := AvailKeyMaps[0]
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
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun 2 key seq: %v = %v\n", ks, kfg)
			}
			kf = kfg
		}
	} else if key1 != "" {
		if _, need2 := Needs2KeyMap[key1]; need2 {
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun 1st key in 2key seq: %v\n", key1)
			}
			return KeyFunNeeds2
		}
		if kfg, ok := (*ActiveKeyMap)[ks]; ok {
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun 1 key seq: %v = %v\n", ks, kfg)
			}
			kf = kfg
		}
	}
	return kf
}

// KeyMapItem records one element of the key map -- used for organizing the map.
type KeyMapItem struct {
	Keys KeySeq  `desc:"the key chord sequence that activates a function"`
	Fun  KeyFuns `desc:"the function of that key"`
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
			log.Printf("gide.KeySeqMap: key function is nil -- probably renamed, for key: %v\n", key)
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
					fmt.Printf("gide.KeyMap: %v is missing a key for function: %v\n", kmName, mi)
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
	Needs2KeyMap = make(gi.KeyMap)

	for key := range *km {
		if key.Key2 != "" {
			Needs2KeyMap[key.Key1] = gi.KeyFunNil
		}
	}

	// issue warnings for needs1 with same
	for key, val := range *km {
		if key.Key2 == "" {
			if _, need2 := Needs2KeyMap[key.Key1]; need2 {
				log.Printf("gide.KeySeqMap: single-key case starts with key chord that is used in key sequence (2 keys in a row) in other mappings -- this is not valid and won't be used: Key: %v  Fun: %v\n",
					key, val)
			}
		}
	}
}

/////////////////////////////////////////////////////////////////////////////////
// KeyMaps -- list of KeyMap's

// KeyMapName has an associated ValueView for selecting from the list of
// available key map names, for use in preferences etc.
type KeyMapName string

// DefaultKeyMap is the overall default keymap -- reinitialized in gimain init()
// depending on platform
var DefaultKeyMap = KeyMapName("MacEmacs")

// KeyMapsItem is an entry in a KeyMaps list
type KeyMapsItem struct {
	Name string    `width:"20" desc:"name of keymap"`
	Desc string    `desc:"description of keymap -- good idea to include source it was derived from"`
	Map  KeySeqMap `desc:"to edit key sequence click button and type new key combination; to edit function mapped to key sequence choose from menu"`
}

// Label satisfies the Labeler interface
func (km KeyMapsItem) Label() string {
	return km.Name
}

// KeyMaps is a list of KeyMap's -- users can edit these in Prefs -- to create
// a custom one, just duplicate an existing map, rename, and customize
type KeyMaps []KeyMapsItem

// KiT_KeyMaps registers KeyMaps as a type
var KiT_KeyMaps = kit.Types.AddType(&KeyMaps{}, KeyMapsProps)

// AvailKeyMaps is the current list of available keymaps for use -- can be
// loaded / saved / edited with preferences.  This is set to StdKeyMaps at
// startup.
var AvailKeyMaps KeyMaps

func init() {
	AvailKeyMaps.CopyFrom(StdKeyMaps)
}

// MapByName returns a keymap and index by name -- returns false and emits a
// message to stdout if not found
func (km *KeyMaps) MapByName(name KeyMapName) (*KeySeqMap, int, bool) {
	for i, it := range *km {
		if it.Name == string(name) {
			return &it.Map, i, true
		}
	}
	fmt.Printf("gi.KeyMaps.MapByName: key map named: %v not found\n", name)
	return nil, -1, false
}

// PrefsKeyMapsFileName is the name of the preferences file in App prefs
// directory for saving / loading the default AvailKeyMaps key maps list
var PrefsKeyMapsFileName = "key_maps_prefs.json"

// OpenJSON opens keymaps from a JSON-formatted file.
func (km *KeyMaps) OpenJSON(filename gi.FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "File Not Found", Prompt: err.Error()}, gi.AddOk, gi.NoCancel, nil, nil)
		log.Println(err)
		return err
	}
	*km = make(KeyMaps, 0, 10) // reset
	return json.Unmarshal(b, km)
}

// SaveJSON saves keymaps to a JSON-formatted file.
func (km *KeyMaps) SaveJSON(filename gi.FileName) error {
	b, err := json.MarshalIndent(km, "", "  ")
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

// OpenPrefs opens KeyMaps from App standard prefs directory, using PrefsKeyMapsFileName
func (km *KeyMaps) OpenPrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsKeyMapsFileName)
	AvailKeyMapsChanged = false
	return km.OpenJSON(gi.FileName(pnm))
}

// SavePrefs saves KeyMaps to App standard prefs directory, using PrefsKeyMapsFileName
func (km *KeyMaps) SavePrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsKeyMapsFileName)
	AvailKeyMapsChanged = false
	return km.SaveJSON(gi.FileName(pnm))
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

// RevertToStd reverts this map to using the StdKeyMaps that are compiled into
// the program and have all the lastest key functions bound to standard
// values.
func (km *KeyMaps) RevertToStd() {
	km.CopyFrom(StdKeyMaps)
	AvailKeyMapsChanged = true
}

// ViewStd shows the standard maps that are compiled into the program and have
// all the lastest key functions bound to standard values.  Useful for
// comparing against custom maps.
func (km *KeyMaps) ViewStd() {
	KeyMapsView(&StdKeyMaps)
}

// AvailKeyMapsChanged is used to update giv.KeyMapsView toolbars via
// following menu, toolbar props update methods -- not accurate if editing any
// other map but works for now..
var AvailKeyMapsChanged = false

// KeyMapsProps define the ToolBar and MenuBar for TableView of KeyMaps, e.g., giv.KeyMapsView
var KeyMapsProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenPrefs", ki.Props{}},
			{"SavePrefs", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": func(kmi interface{}, act *gi.Action) {
					act.SetActiveState(AvailKeyMapsChanged && kmi.(*KeyMaps) == &AvailKeyMaps)
				},
			}},
			{"sep-file", ki.BlankProp{}},
			{"OpenJSON", ki.Props{
				"label":    "Open from file",
				"desc":     "You can save and open key maps to / from files to share, experiment, transfer, etc",
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"SaveJSON", ki.Props{
				"label": "Save to file",
				"desc":  "You can save and open key maps to / from files to share, experiment, transfer, etc",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"RevertToStd", ki.Props{
				"desc":    "This reverts the keymaps to using the StdKeyMaps that are compiled into the program and have all the lastest key functions defined.  If you have edited your maps, and are finding things not working, it is a good idea to save your current maps and try this, or at least do ViewStdMaps to see the current standards.  <b>Your current map edits will be lost if you proceed!</b>  Continue?",
				"confirm": true,
			}},
		}},
		{"Edit", "Copy Cut Paste Dupe"},
		{"Window", "Windows"},
	},
	"ToolBar": ki.PropSlice{
		{"SavePrefs", ki.Props{
			"desc": "saves KeyMaps to App standard prefs directory, in file key_maps_prefs.json, which will be loaded automatically at startup if prefs SaveKeyMaps is checked (should be if you're using custom keymaps)",
			"icon": "file-save",
			"updtfunc": func(kmi interface{}, act *gi.Action) {
				act.SetActiveState(AvailKeyMapsChanged && kmi.(*KeyMaps) == &AvailKeyMaps)
			},
		}},
		{"sep-file", ki.BlankProp{}},
		{"OpenJSON", ki.Props{
			"label": "Open from file",
			"icon":  "file-open",
			"desc":  "You can save and open key maps to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", ki.Props{
			"label": "Save to file",
			"icon":  "file-save",
			"desc":  "You can save and open key maps to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"sep-std", ki.BlankProp{}},
		{"ViewStd", ki.Props{
			"desc":    "Shows the standard maps that are compiled into the program and have all the lastest key functions bound to standard key chords.  Useful for comparing against custom maps.",
			"confirm": true,
			"updtfunc": func(kmi interface{}, act *gi.Action) {
				act.SetActiveStateUpdt(kmi.(*KeyMaps) != &StdKeyMaps)
			},
		}},
		{"RevertToStd", ki.Props{
			"icon":    "update",
			"desc":    "This reverts the keymaps to using the StdKeyMaps that are compiled into the program and have all the lastest key functions bound to standard key chords.  If you have edited your maps, and are finding things not working, it is a good idea to save your current maps and try this, or at least do ViewStdMaps to see the current standards.  <b>Your current map edits will be lost if you proceed!</b>  Continue?",
			"confirm": true,
			"updtfunc": func(kmi interface{}, act *gi.Action) {
				act.SetActiveStateUpdt(kmi.(*KeyMaps) != &StdKeyMaps)
			},
		}},
	},
}

// StdKeyMaps is the original compiled-in set of standard keymaps that have
// the lastest key functions bound to standard key chords.
var StdKeyMaps = KeyMaps{
	{"MacStd", "Standard Mac KeyMap", KeySeqMap{
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
	{"LinuxStd", "Standard Linux KeySeqMap", KeySeqMap{
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
	{"WindowsStd", "Standard Windows KeySeqMap", KeySeqMap{
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
