package main

import (
	"image/color"

	"cogentcore.org/core/colors"
)

// TODO: update settings to use Cogent Core settings structure

// Settings are the settings the app has
type Settings struct {
	LineDefaults LineDefaults `view:"no-inline" label:"Line Defaults"`

	GraphDefaults Params `view:"no-inline" label:"Graph Param Defaults"`

	MarbleSettings MarbleSettings `view:"inline" label:"Marble Settings"`

	GraphSize int `label:"Graph Size" min:"100" max:"800"`

	GraphInc int `label:"Line quality in n per x" min:"1" max:"100"`

	NFramesPer   int  `label:"Number of times to update the marbles each render" min:"1" max:"100"`
	LineFontSize int  `label:"Line Font Size"`
	ConfirmQuit  bool `label:"Confirm App Close"`
	PrettyJSON   bool `label:"Save formatted JSON"`
}

// MarbleSettings are the settings for the marbles in the app
type MarbleSettings struct {
	MarbleColor string
	MarbleSize  float64
}

// LineDefaults are the settings for the default line
type LineDefaults struct {
	Expr       string
	GraphIf    string
	Bounce     string
	LineColors LineColors
}

// TrackingSettings contains the tracking line settings
type TrackingSettings struct {
	TrackByDefault bool

	NTrackingFrames int `min:"0" step:"10"`

	Accuracy  int `min:"1" max:"100" step:"5"`
	LineColor color.RGBA
}

// TheSettings is the instance of settings
var TheSettings Settings

/*
// SettingProps is the toolbar for settings
var SettingProps = ki.Props{
	"ToolBar": ki.PropSlice{
		{Name: "Reset", Value: ki.Props{
			"label": "Reset Settings",
		},
		},
	}}

// KiTTheSettings is for the toolbar
var KiTTheSettings = kit.Types.AddType(&TheSettings, SettingProps)

// Reset resets the settings to defaults
func (se *Settings) Reset() {
	se.Defaults()
	se.Save()
}

// Get gets the settings from localdata/settings.json
func (se *Settings) Get() {
	b, err := os.ReadFile(filepath.Join(GetMarblesFolder(), "localData/settings.json"))
	if err != nil {
		se.Defaults()
		se.Save()
		return
	}
	err = json.Unmarshal(b, se)
	if err != nil {
		se.Defaults()
		se.Save()
		return
	}
	if se.LineDefaults.Expr == "" {
		se.LineDefaults.BasicDefaults()
		se.Save()
	}
	if se.MarbleSettings.MarbleColor == "" {
		se.MarbleSettings.Defaults()
		se.Save()
	}
	if se.ColorSettings.BackgroundColor == gist.NilColor {
		se.ColorSettings.Defaults()
		se.Save()
	}

}

// Save saves the settings to localData/settings.json
func (se *Settings) Save() {
	var b []byte
	var err error
	if TheSettings.PrettyJSON {
		b, err = json.MarshalIndent(se, "", "  ")
	} else {
		b, err = json.Marshal(se)
	}
	if HandleError(err) {
		return
	}
	err = os.WriteFile(filepath.Join(GetMarblesFolder(), "localData/settings.json"), b, os.ModePerm)
	HandleError(err)
}
*/

// Defaults defaults the settings
func (se *Settings) Defaults() {
	se.LineDefaults.BasicDefaults()
	se.GraphDefaults.BasicDefaults()
	se.MarbleSettings.Defaults()
	se.GraphSize = 700
	se.GraphInc = 40
	se.NFramesPer = 1
	se.LineFontSize = 24
	se.ConfirmQuit = true
	se.PrettyJSON = false
}

// Defaults sets the default settings for the tracking lines.
func (ts *TrackingSettings) Defaults() {
	ts.TrackByDefault = false
	ts.NTrackingFrames = 300
	ts.Accuracy = 20
	ts.LineColor = colors.White
}

// BasicDefaults sets the line defaults to their defaults
func (ln *LineDefaults) BasicDefaults() {
	ln.Expr = "sinx"
	ln.LineColors.Color = colors.White
	ln.Bounce = "0.95"
	ln.GraphIf = "true"
	ln.LineColors.ColorSwitch = colors.White
}

// Defaults sets the marble settings to their defaults
func (ms *MarbleSettings) Defaults() {
	ms.MarbleColor = "default"
	ms.MarbleSize = 0.1
}
