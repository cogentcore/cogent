// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"encoding/json"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/styles"
)

// Preferences is the overall Vector preferences
type Preferences struct { //gti:add

	// default physical size, when app is started without opening a file
	Size PhysSize

	// active color preferences
	Colors ColorSettings

	// named color schemes -- has Light and Dark schemes by default
	ColorSchemes map[string]*ColorSettings

	// default shape styles
	ShapeStyle styles.Paint

	// default text styles
	TextStyle styles.Paint

	// default line styles
	PathStyle styles.Paint

	// default line styles
	LineStyle styles.Paint

	// turns on the grid display
	VectorDisp bool

	// snap positions and sizes to underlying grid
	SnapVector bool

	// snap positions and sizes to line up with other elements
	SnapGuide bool

	// snap node movements to align with guides
	SnapNodes bool

	// number of screen pixels around target point (in either direction) to snap
	SnapTol int `min:"1"`

	// named-split config in use for configuring the splitters
	SplitName SplitName

	// environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app
	EnvVars map[string]string

	// flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" xml:"-"`
}

func (pf *Preferences) Defaults() {
	pf.Size.Defaults()
	pf.Colors.Defaults()
	pf.ColorSchemes = DefaultColorSchemes()
	pf.ShapeStyle.Defaults()
	pf.ShapeStyle.FontStyle.Family = "Arial"
	pf.ShapeStyle.FontStyle.Size.Px(12)
	// pf.ShapeStyle.FillStyle.Color.SetName("blue")
	// pf.ShapeStyle.StrokeStyle.On = true // todo: image
	// pf.ShapeStyle.FillStyle.On = true
	pf.TextStyle.Defaults()
	pf.TextStyle.FontStyle.Family = "Arial"
	pf.TextStyle.FontStyle.Size.Px(12)
	// pf.TextStyle.StrokeStyle.On = false
	// pf.TextStyle.FillStyle.On = true
	pf.PathStyle.Defaults()
	pf.PathStyle.FontStyle.Family = "Arial"
	pf.PathStyle.FontStyle.Size.Px(12)
	// pf.PathStyle.StrokeStyle.On = true
	// pf.PathStyle.FillStyle.On = false
	pf.LineStyle.Defaults()
	pf.LineStyle.FontStyle.Family = "Arial"
	pf.LineStyle.FontStyle.Size.Px(12)
	// pf.LineStyle.StrokeStyle.On = true
	// pf.LineStyle.FillStyle.On = false
	pf.VectorDisp = true
	pf.SnapTol = 3
	pf.SnapVector = true
	pf.SnapGuide = true
	pf.SnapNodes = true
	home := core.SystemSettings.User.HomeDir
	pf.EnvVars = map[string]string{
		"PATH": home + "/bin:" + home + "/go/bin:/usr/local/bin:/opt/homebrew/bin:/opt/homebrew/shbin:/Library/TeX/texbin:/usr/bin:/bin:/usr/sbin:/sbin",
	}
}

func (pf *Preferences) Update() {
	pf.Size.Update()
}

// Settings are the overall Vector preferences
var Settings = Preferences{}

// InitSettings must be called at startup in mainrun()
func InitSettings() {
	core.TheApp.SetName("Cogent Vector")
	Settings.Defaults()
	Settings.Open()
	// OpenPaths() // todo
}

// SettingsFileName is the name of the preferences file in app settings directory
var SettingsFileName = "grid_prefs.json"

// Open preferences from app standard prefs directory, and applies them
func (pf *Preferences) Open() error {
	pdir := goosi.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SettingsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	// AvailSplits.OpenSettings() // todo
	pf.ApplyEnvVars()
	pf.Changed = false
	return err
}

// Save Preferences to app standard prefs directory
func (pf *Preferences) Save() error {
	pdir := goosi.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SettingsFileName)
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(pnm, b, 0644)
	if err != nil {
		log.Println(err)
	}
	AvailableSplits.SaveSettings()
	pf.Changed = false
	return err
}

// ApplyEnvVars applies environment variables set in EnvVars
func (pf *Preferences) ApplyEnvVars() {
	for k, v := range pf.EnvVars {
		os.Setenv(k, v)
	}
}

// LightMode sets colors to light mode
func (pf *Preferences) LightMode() {
	lc, ok := pf.ColorSchemes["Light"]
	if !ok {
		log.Printf("Light ColorScheme not found\n")
		return
	}
	pf.Colors = *lc
	pf.Save()
	pf.UpdateAll()
}

// DarkMode sets colors to dark mode
func (pf *Preferences) DarkMode() {
	lc, ok := pf.ColorSchemes["Dark"]
	if !ok {
		log.Printf("Dark ColorScheme not found\n")
		return
	}
	pf.Colors = *lc
	pf.Save()
	pf.UpdateAll()
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (pf *Preferences) EditSplits() {
	SplitsView(&AvailableSplits)
}

// UpdateAll updates all open windows with current preferences -- triggers
// rebuild of default styles.
func (pf *Preferences) UpdateAll() {
	// gist.RebuildDefaultStyles = true
	// color.ColorSpecCache = nil
	// gist.StyleTemplates = nil
	// for _, w := range core.AllWindows {  // no need and just messes stuff up!
	// 	w.SetSize(w.OSWin.Size())
	// }
	// // needs another pass through to get it right..
	// for _, w := range core.AllWindows {
	// 	w.FullReRender()
	// }
	// gist.RebuildDefaultStyles = false
	// // and another without rebuilding?  yep all are required
	// for _, w := range core.AllWindows {
	// 	w.FullReRender()
	// }
}

/*
// PreferencesProperties define the Toolbar and MenuBar for StructView, e.g., views.SettingsView
var PreferencesProperties = tree.Properties{
	"MainMenu": tree.Propertieslice{
		{"AppMenu", tree.BlankProp{}},
		{"File", tree.Propertieslice{
			{"Open", tree.Properties{
				"shortcut": "Command+O",
			}},
			{"Save", tree.Properties{
				"shortcut": "Command+S",
				"updtfunc": views.ActionUpdateFunc(func(pfi any, act *core.Button) {
					pf := pfi.(*Preferences)
					act.SetActiveState(pf.Changed)
				}),
			}},
			{"sep-color", tree.BlankProp{}},
			{"LightMode", tree.Properties{}},
			{"DarkMode", tree.Properties{}},
			{"sep-close", tree.BlankProp{}},
			{"Close Window", tree.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
	"Toolbar": tree.Propertieslice{
		{"Save", tree.Properties{
			"desc": "Saves current preferences to standard prefs.json file, which is auto-loaded at startup.",
			"icon": "file-save",
			"updtfunc": views.ActionUpdateFunc(func(pfi any, act *core.Button) {
				pf := pfi.(*Preferences)
				act.SetActiveStateUpdate(pf.Changed)
			}),
		}},
		{"sep-color", tree.BlankProp{}},
		{"LightMode", tree.Properties{
			"desc": "Set color mode to Light mode as defined in ColorSchemes -- automatically does Save and UpdateAll ",
			"icon": "color",
		}},
		{"DarkMode", tree.Properties{
			"desc": "Set color mode to Dark mode as defined in ColorSchemes -- automatically does Save and UpdateAll",
			"icon": "color",
		}},
		{"sep-misc", tree.BlankProp{}},
		{"VersionInfo", tree.Properties{
			"desc":        "shows current Vector version information",
			"icon":        "info",
			"show-return": true,
		}},
		{"sep-key", tree.BlankProp{}},
		{"EditSplits", tree.Properties{
			"icon": "file-binary",
			"desc": "opens the SplitsView editor of saved named splitter settings.  Current customized settings are saved and loaded with preferences automatically.",
		}},
	},
}
*/

/////////////////////////////////////////////////////////////////////////////////
//   ColorSettings

// ColorSettings for
type ColorSettings struct { //gti:add

	// drawing background color
	Background color.Color

	// border color of the drawing
	Border color.Color

	// grid line color
	Vector color.Color
}

func (pf *ColorSettings) Defaults() {
	pf.Background = colors.White
	pf.Border = colors.Black
	pf.Vector = color.RGBA{220, 220, 220, 255}
}

func (pf *ColorSettings) DarkDefaults() {
	pf.Background = colors.Black
	pf.Border = color.RGBA{102, 102, 102, 255}
	pf.Vector = color.RGBA{40, 40, 40, 255}
}

func DefaultColorSchemes() map[string]*ColorSettings {
	cs := map[string]*ColorSettings{}
	lc := &ColorSettings{}
	lc.Defaults()
	cs["Light"] = lc
	dc := &ColorSettings{}
	dc.DarkDefaults()
	cs["Dark"] = dc
	return cs
}

// OpenJSON opens colors from a JSON-formatted file.
func (pf *ColorSettings) OpenJSON(filename core.Filename) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		core.ErrorDialog(nil, err, "File Not Found")
		log.Println(err)
		return err
	}
	return json.Unmarshal(b, pf)
}

// SaveJSON saves colors to a JSON-formatted file.
func (pf *ColorSettings) SaveJSON(filename core.Filename) error {
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		core.ErrorDialog(nil, err, "Could not Save to File")
		log.Println(err)
	}
	return err
}

// SetToSettings sets this color scheme as the current active setting in overall
// default prefs.
func (pf *ColorSettings) SetToSettings() {
	Settings.Colors = *pf
	Settings.UpdateAll()
}

/*
// ColorSettingsProperties defines the Toolbar
var ColorSettingsProperties = tree.Properties{
	"Toolbar": tree.Propertieslice{
		{"OpenJSON", tree.Properties{
			"label": "Open...",
			"icon":  "file-open",
			"desc":  "open set of colors from a json-formatted file",
			"Args": tree.Propertieslice{
				{"Color File Name", tree.Properties{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", tree.Properties{
			"label": "Save As...",
			"desc":  "Saves colors to JSON formatted file.",
			"icon":  "file-save",
			"Args": tree.Propertieslice{
				{"Color File Name", tree.Properties{
					"ext": ".json",
				}},
			},
		}},
		{"SetToSettings", tree.Properties{
			"desc": "Sets this color scheme as the current active color scheme in Settings.",
			"icon": "reset",
		}},
	},
}

*/
