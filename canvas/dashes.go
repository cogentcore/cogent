// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"cogentcore.org/core/icons"
)

// DashMulWidth returns the dash array multiplied by the line width -- what is actually set
func DashMulWidth(lwidth float64, dary []float64) []float64 {
	mary := slices.Clone(dary)
	for i := range mary {
		mary[i] *= lwidth
	}
	return mary
}

// DashString returns string of dash array values
func DashString(dary []float64) string {
	ds := ""
	for i := range dary {
		ds += fmt.Sprintf("%g,", dary[i])
	}
	ds = strings.TrimSuffix(ds, ",")
	return ds
}

// DashMatchArray returns the matching dash pattern for given array and line width.
// divides array and matches with wide tolerance.
// returns true if no match and thus new dash pattern was added, else false.
func DashMatchArray(lwidth float64, dary []float64) (bool, string) {
	if lwidth == 0 { // no div-by-0
		lwidth = 1
	}
	sz := len(dary)
	if sz == 0 {
		return false, "dash-solid"
	}
	mary := slices.Clone(dary)
	for i := range mary {
		mary[i] /= lwidth
	}
	for k, v := range AllDashesMap {
		if len(v) != sz {
			continue
		}
		match := true
		for i := range mary {
			if math.Abs(v[i]-mary[i]) > 0.5 {
				match = false
				break
			}
		}
		if match {
			return false, k
		}
	}
	// new beast -- add it
	return true, AddNewDash(mary)
}

// AddNewDash adds new dash pattern to available list, creating name based on pattern,
// which is returned.  the given array is copied before storing, just in case.
func AddNewDash(dary []float64) string {
	nm := "custom"
	for i := range dary {
		nm += fmt.Sprintf("-%g", dary[i])
	}
	AllDashNames = append(AllDashNames, nm)
	AllDashesMap[nm] = slices.Clone(dary)
	return nm
}

// StandardDashNames are standard dash patterns
var StandardDashNames = []string{
	"dash-solid",
	"dash-1-1",
	"dash-1-2",
	"dash-1-3",
	"dash-1-4",
	"dash-1-6",
	"dash-1-8",
	"dash-1-12",
	"dash-1-24",
	"dash-1-48",
	"dash-empty",
	"dash-2-1",
	"dash-3-1",
	"dash-4-1",
	"dash-6-1",
	"dash-8-1",
	"dash-12-1",
	"dash-24-1",
	"dash-2-2",
	"dash-3-3",
	"dash-4-4",
	"dash-6-6",
	"dash-8-8",
	"dash-12-12",
	"dash-24-24",
	"dash-2-4",
	"dash-4-2",
	"dash-2-6",
	"dash-6-2",
	"dash-4-8",
	"dash-8-4",
	"dash-2-1-012-1",
	"dash-4-2-1-2",
	"dash-8-2-1-2",
	"dash-012-012",
	"dash-014-014",
	"dash-0110-0110",
}

// StandardDashesMap are standard dash patterns
var StandardDashesMap = map[string][]float64{
	"dash-solid":     {},
	"dash-1-1":       {1, 1},
	"dash-1-2":       {1, 2},
	"dash-1-3":       {1, 3},
	"dash-1-4":       {1, 4},
	"dash-1-6":       {1, 6},
	"dash-1-8":       {1, 8},
	"dash-1-12":      {1, 12},
	"dash-1-24":      {1, 24},
	"dash-1-48":      {1, 48},
	"dash-empty":     {0, 11},
	"dash-2-1":       {2, 1},
	"dash-3-1":       {3, 1},
	"dash-4-1":       {4, 1},
	"dash-6-1":       {6, 1},
	"dash-8-1":       {8, 1},
	"dash-12-1":      {12, 1},
	"dash-24-1":      {24, 1},
	"dash-2-2":       {2, 2},
	"dash-3-3":       {3, 3},
	"dash-4-4":       {4, 4},
	"dash-6-6":       {6, 6},
	"dash-8-8":       {8, 8},
	"dash-12-12":     {12, 12},
	"dash-24-24":     {24, 24},
	"dash-2-4":       {2, 4},
	"dash-4-2":       {4, 2},
	"dash-2-6":       {2, 6},
	"dash-6-2":       {6, 2},
	"dash-4-8":       {4, 8},
	"dash-8-4":       {8, 4},
	"dash-2-1-012-1": {2, 1, 0.5, 1},
	"dash-4-2-1-2":   {4, 2, 1, 2},
	"dash-8-2-1-2":   {8, 2, 1, 2},
	"dash-012-012":   {0.5, 0.5},
	"dash-014-014":   {0.25, 0.25},
	"dash-0110-0110": {0.1, 0.1},
}

// AllDashesMap contains all of the available Dashes.
// it is initialized from StdDashesMap
var AllDashesMap map[string][]float64

// AllDashNames contains all of the available dash names.
// it is initialized from StdDashNames.
var AllDashNames []string

// AllDashIcons contains all of the available dash names as
// Icons -- for chooser.
var AllDashIcons []icons.Icon

// DashIconsInited records whether the dashes have been initialized into
// Icons for use in selectors: see DashIconsInit()
var DashIconsInited = false

// DashIconsInit ensures that the dashes have been turned into icons
// for selectors, with same name (dash- prefix).  Call this after
// startup, when configuring a gui element that needs it.
func DashIconsInit() {
	if DashIconsInited {
		return
	}

	AllDashIcons = make([]icons.Icon, len(AllDashNames))
	for i, v := range AllDashNames {
		AllDashIcons[i] = icons.Icon(v)
	}

	// for k, v := range AllDashesMap {
	// 	ic := &core.SVG{}
	// 	ic.InitName(ic, k)
	// 	ic.SetProp("width", units.Ch(20))
	// 	ic.SVG.Root.ViewBox.Size = math32.Vec2(1, 1)
	// 	p := svg.NewPath(ic, "p", "M 0.05 0.5 .95 0.5 Z")
	// 	p.SetProp("stroke-width", units.Pw(2))
	// 	p.SetProp("stroke-dasharray", DashString(DashMulWidth(.05, v)))
	// 	// svg.CurIconSet[ic.Nm] = ic
	// }
	DashIconsInited = true
}

func init() {
	AllDashesMap = make(map[string][]float64, len(StandardDashesMap))
	AllDashNames = make([]string, len(StandardDashNames))
	maps.Copy(AllDashesMap, StandardDashesMap)
	copy(AllDashNames, StandardDashNames)
}
