// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"bytes"
	"io"
	"log"
	"maps"
	"strings"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/svg"
)

// MarkerFromNodeProp returns the marker name (canonicalized -- no id)
// and id and color type
func MarkerFromNodeProp(kn ki.Ki, prop string) (string, int, MarkerColors) {
	if kn == nil {
		return "", 0, MarkerDef
	}
	p := kn.Prop(prop)
	if p == nil {
		return "", 0, MarkerDef
	}
	ms := laser.ToString(p)
	if ms == "" {
		return "", 0, MarkerDef
	}
	mc := MarkerDef
	nm, id := svg.SplitNameIDDig(svg.NameFromURL(ms))
	if id > 0 {
		_, sid := svg.SplitNameIDDig(kn.Name())
		if id == sid { // if match, then copy
			mc = MarkerCopy
		} else { // then custom
			mc = MarkerCust
		}
	}
	return nm, id, mc
}

// RecycleMarker ensures that given marker name and id exists in SVG,
// making a new one, copying from standard markers if not.
// if mc is MarkerCopyColor then sets marker colors to node colors.
func RecycleMarker(sg *svg.SVG, sii svg.Node, name string, id int, mc MarkerColors) *svg.Marker {
	nmeff := svg.NameID(name, id)
	mk := sg.FindDefByName(nmeff)
	fc := laser.ToString(sii.Prop("fill"))
	sc := laser.ToString(sii.Prop("stroke"))
	var mmk *svg.Marker
	newmk := false
	if mk != nil {
		mmk = mk.(*svg.Marker)
	} else {
		mmk = NewMarker(sg, name, id)
		newmk = true
	}
	switch mc {
	case MarkerDef:
		if newmk {
			MarkerDeleteCtxtColors(mmk) // get rid of those context-stroke etc
		}
	case MarkerCopy:
		MarkerSetColors(mmk, fc, sc)
	case MarkerCust:
		if newmk {
			MarkerSetColors(mmk, "blue", "red")
		}
	}
	return mmk
}

// MarkerSetColors sets color properties in each element
func MarkerSetColors(mk *svg.Marker, fill, stroke string) {
	mk.WalkPre(func(k ki.Ki) bool {
		fp := k.Prop("fill")
		if fp != nil {
			if strings.HasPrefix(mk.Nm, "Empty") {
				k.SetProp("fill", fill)
			} else {
				k.SetProp("fill", stroke)
			}
		}
		sp := k.Prop("stroke")
		if sp != nil {
			if strings.HasPrefix(mk.Nm, "Distance") {
				k.SetProp("stroke", fill)
			} else {
				k.SetProp("stroke", stroke)
			}
		}
		return ki.Continue
	})
}

// MarkerDeleteCtxtColors deletes context-* color names from standard code
func MarkerDeleteCtxtColors(mk *svg.Marker) {
	mk.WalkPre(func(k ki.Ki) bool {
		fp := k.Prop("fill")
		if fp != nil {
			fps := laser.ToString(fp)
			if strings.HasPrefix(fps, "context-") {
				k.DeleteProp("fill")
			}
		}
		sp := k.Prop("stroke")
		if sp != nil {
			sps := laser.ToString(sp)
			if strings.HasPrefix(sps, "context-") {
				k.DeleteProp("stroke")
			}
		}
		return ki.Continue
	})
}

// NewMarkerFromXML makes a new marker from given XML source.
func NewMarkerFromXML(name, xml string) *svg.Marker {
	tmpsvg := svg.NewSVG(0, 0)
	b := bytes.NewBufferString(xml)
	err := tmpsvg.ReadXML(b)
	if err != nil && err != io.EOF {
		log.Printf("NewMarkerFromXML: failure loading marker XML for %s", name)
		log.Println(err)
		return nil
	}
	if tmpsvg.Root.NumChildren() != 1 {
		log.Printf("NewMarkerFromXML: kids != 1 (%d) after loading marker XML for %s", tmpsvg.Root.NumChildren(), name)
		return nil
	}
	mk := tmpsvg.Root.Child(0).(*svg.Marker)
	mk.SetName(name)
	ki.UniquifyNamesAll(mk) // critical b/c doing copy!
	return mk
}

// NewMarker makes a new marker of given name and id in given svg,
// copying from existing one in AllMarkersSVGMap.
func NewMarker(sg *svg.SVG, name string, id int) *svg.Marker {
	mk, ok := AllMarkersSVGMap[name]
	if !ok {
		log.Printf("NewMarker: marker named %s not found in AllMarkersSVGMap; will likely crash!\n", name)
		return nil
	}
	// updt := sg.UpdateStart()
	nmk := &svg.Marker{}
	fnm := svg.NameID(name, id)
	nmk.InitName(nmk, fnm)
	nmk.CopyFrom(mk)
	mk.SetName(fnm) // double check
	sg.Root.SetChildAdded()
	sg.Defs.AddChild(nmk)
	// sg.UpdateEnd(updt)
	return nmk
}

// MarkerSetProp sets marker property for given node to given marker name (canonical)
func MarkerSetProp(sg *svg.SVG, sii svg.Node, prop, name string, mc MarkerColors) {
	onm, oid, omc := MarkerFromNodeProp(sii, prop)
	_, nid := svg.SplitNameIDDig(sii.Name())
	if onm == name && oid == nid && omc == mc {
		return
	}
	if name == "" || name == "-" {
		if onm != "" && omc == MarkerCopy {
			sg.Defs.DeleteChildByName(svg.NameID(onm, oid), ki.DestroyKids)
		}
		sii.DeleteProp(prop)
		return
	}
	if omc == MarkerCopy && omc != mc { // implies onm != ""
		sg.Defs.DeleteChildByName(svg.NameID(onm, oid), ki.DestroyKids)
	}

	_, ok := AllMarkersXMLMap[name]
	if !ok {
		log.Printf("MarkerSetProp: marker named %s not found in AllMarkersXMLMap\n", name)
		return
	}

	var nmk *svg.Marker
	switch mc {
	case MarkerDef:
		nmk = RecycleMarker(sg, sii, name, 0, mc)
	case MarkerCopy:
		nmk = RecycleMarker(sg, sii, name, nid, mc)
	case MarkerCust:
		id := oid
		if onm != name || id == 0 {
			id = sg.NewUniqueID()
		}
		nmk = RecycleMarker(sg, sii, name, id, mc)
	}
	sii.SetProp(prop, svg.NameToURL(nmk.Nm))
}

// MarkerUpdateColorProp updates marker color for given marker property
func MarkerUpdateColorProp(sg *svg.SVG, sii svg.Node, prop string) {
	nm, id, mc := MarkerFromNodeProp(sii, prop)
	if nm == "" || mc != MarkerCopy {
		return
	}
	RecycleMarker(sg, sii, nm, id, mc)
}

// MarkerColors are the drawing tools
type MarkerColors int32 //enums:enum -trim-prefix Marker

const (
	// use the default color of marker (typically black)
	MarkerDef MarkerColors = iota

	// copy color of object using marker (create separate marker object per element)
	MarkerCopy

	// marker has its own separate custom color
	MarkerCust
)

//////////////////////////////////////////////////////////////////////////
//  AllMarkers Collection

// AllMarkersXMLMap contains all of the available Markers as XML
// source.  It is initialized from StdMarkersMap
var AllMarkersXMLMap map[string]string

// AllMarkersSVGMap contains all of the available Markers
// as *svg.Marker elements that have been converted from
// the XML source.
var AllMarkersSVGMap map[string]*svg.Marker

// AllMarkerNames contains all of the available marker names.
// it is initialized from StdMarkerNames.
var AllMarkerNames []string

// AllMarkerIconNames contains all of the available marker names as
// IconNames -- for chooser.  All names have marker- prefix in addition
// to regular marker names.
var AllMarkerIconNames []icons.Icon

func init() {
	AllMarkersXMLMap = make(map[string]string, len(StdMarkersMap))
	AllMarkerNames = make([]string, len(StdMarkerNames))
	maps.Copy(AllMarkersXMLMap, StdMarkersMap)
	copy(AllMarkerNames, StdMarkerNames)
}

// IconToMarkerName converts a icons.Icon (as an interface{})
// to a marker name suitable for use (removes marker- prefix)
func IconToMarkerName(icnm any) string {
	return strings.TrimPrefix(laser.ToString(icnm), "marker-")
}

// MarkerNameToIcon converts a marker name to a icons.Icon
func MarkerNameToIcon(nm string) icons.Icon {
	return icons.Icon("marker-" + nm)
}

// MarkerIconsInited records whether the dashes have been initialized into
// Icons for use in selectors: see MarkerIconsInit()
var MarkerIconsInited = false

// MarkerIconsInit ensures that the markers have been turned into icons
// for selectors, with marker- preix.  Call this after startup,
// when configuring a gui element that needs it.
// It also initializes the AllMarkersSVGMap.
func MarkerIconsInit() {
	if MarkerIconsInited {
		return
	}

	AllMarkerIconNames = make([]icons.Icon, len(AllMarkerNames))
	for i, v := range AllMarkerNames {
		AllMarkerIconNames[i] = icons.Icon("marker-" + v)
	}

	AllMarkersSVGMap = make(map[string]*svg.Marker, len(AllMarkersXMLMap))

	// for k, v := range AllMarkersXMLMap {
	// 	empty := true
	// 	if v != "" {
	// 		mk := NewMarkerFromXML(k, v)
	// 		if mk == nil { // badness
	// 			continue
	// 		}
	// 		empty = false
	// 		AllMarkersSVGMap[k] = mk
	// 	}
	// 	ic := &gi.Icon{}
	// 	ic.InitName(ic, "marker-"+k) // keep it distinct with marker- prefix
	// 	ic.Styles.Min.X.Ch(6)
	// 	ic.Styles.Min.Y.Em(2)
	// 	ic.SVG.Root.ViewBox.Size = mat32.V2(1, 1)
	// 	var p *svg.Path
	// 	lk := strings.ToLower(k)
	// 	start := true
	// 	switch {
	// 	case empty:
	// 		p = svg.NewPath(ic, "p", "M 0.1 0.5 0.9 0.5 Z")
	// 	case strings.Contains(lk, "end"):
	// 		start = false
	// 		p = svg.NewPath(ic, "p", "M 0.8 0.5 0.9 0.5 Z")
	// 	case strings.Contains(lk, "start"):
	// 		p = svg.NewPath(ic, "p", "M 0.1 0.5 0.2 0.5 Z")
	// 	default:
	// 		p = svg.NewPath(ic, "p", "M 0.4 0.5 0.5 0.5 Z")
	// 	}
	// 	p.SetProp("stroke-width", units.Pw(5))
	// 	if !empty {
	// 		mk := NewMarker(&ic.SVG, k, 0)
	// 		MarkerDeleteCtxtColors(mk) // get rid of those context-stroke etc
	// 		if start {
	// 			p.SetProp("marker-start", svg.NameToURL(k))
	// 		} else {
	// 			p.SetProp("marker-end", svg.NameToURL(k))
	// 		}
	// 	}
	// 	// svg.CurIconSet[ic.Nm] = ic
	// }
	MarkerIconsInited = true
}

//////////////////////////////////////////////////

// StdMarkerNames is an ordered list of marker names
var StdMarkerNames = []string{
	"-",

	"Arrow1Sstart",
	"Arrow1Send",
	"Arrow1Mstart",
	"Arrow1Mend",
	"Arrow1Lstart",
	"Arrow1Lend",

	"Arrow2Sstart",
	"Arrow2Send",
	"Arrow2Mstart",
	"Arrow2Mend",
	"Arrow2Lstart",
	"Arrow2Lend",

	"Tail",

	"DistanceStart",
	"DistanceEnd",

	"DotS",
	"DotM",
	"DotL",

	"SquareS",
	"SquareM",
	"SquareL",

	"DiamondS",
	"DiamondM",
	"DiamondL",

	"DiamondSstart",
	"DiamondSend",
	"DiamondMstart",
	"DiamondMend",
	"DiamondLstart",
	"DiamondLend",

	"EmptyDiamondS",
	"EmptyDiamondM",
	"EmptyDiamondL",

	"EmptyDiamondSstart",
	"EmptyDiamondSend",
	"EmptyDiamondMstart",
	"EmptyDiamondMend",
	"EmptyDiamondLstart",
	"EmptyDiamondLend",

	"TriangleInS",
	"TriangleInM",
	"TriangleInL",

	"TriangleOutS",
	"TriangleOutM",
	"TriangleOutL",

	"EmptyTriangleInS",
	"EmptyTriangleInM",
	"EmptyTriangleInL",

	"EmptyTriangleOutS",
	"EmptyTriangleOutM",
	"EmptyTriangleOutL",

	"StopS",
	"StopM",
	"StopL",

	"SemiCircleIn",
	"SemiCircleOut",

	"CurveIn",
	"CurveOut",
	"CurvyCross",

	"Scissors",
	"Legs",
	"Torso",
	"Club",
	"RazorWire",

	"InfiniteLineStart",
	"InfiniteLineEnd",
}

// StdMarkersMap is a map of the standard markers
var StdMarkersMap = map[string]string{
	"-": "",
	"Arrow1Lstart": `<marker style="overflow:visible" id="Arrow1Lstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow1Lstart">
      <path transform="scale(0.8) translate(12.5,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0.0,0.0 L 5.0,-5.0 L -12.5,0.0 L 5.0,5.0 L 0.0,0.0 z "/>
    </marker>`,
	"Arrow1Lend": `<marker style="overflow:visible;" id="Arrow1Lend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow1Lend">
      <path transform="scale(0.8) rotate(180) translate(12.5,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt;" d="M 0.0,0.0 L 5.0,-5.0 L -12.5,0.0 L 5.0,5.0 L 0.0,0.0 z "/>
    </marker>`,

	"Arrow1Mstart": `<marker style="overflow:visible" id="Arrow1Mstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow1Mstart">
      <path transform="scale(0.4) translate(10,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0.0,0.0 L 5.0,-5.0 L -12.5,0.0 L 5.0,5.0 L 0.0,0.0 z "/>
    </marker>`,
	"Arrow1Mend": `<marker style="overflow:visible;" id="Arrow1Mend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow1Mend">
      <path transform="scale(0.4) rotate(180) translate(10,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt;" d="M 0.0,0.0 L 5.0,-5.0 L -12.5,0.0 L 5.0,5.0 L 0.0,0.0 z "/>
    </marker>`,

	"Arrow1Sstart": `<marker style="overflow:visible" id="Arrow1Sstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow1Sstart">
      <path transform="scale(0.2) translate(6,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0.0,0.0 L 5.0,-5.0 L -12.5,0.0 L 5.0,5.0 L 0.0,0.0 z "/>
    </marker>`,
	"Arrow1Send": `<marker style="overflow:visible;" id="Arrow1Send" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow1Send">
      <path transform="scale(0.2) rotate(180) translate(6,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt;" d="M 0.0,0.0 L 5.0,-5.0 L -12.5,0.0 L 5.0,5.0 L 0.0,0.0 z "/>
    </marker>`,

	"Arrow2Lstart": `<marker style="overflow:visible" id="Arrow2Lstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow2Lstart">
      <path transform="scale(1.1) translate(1,0)" d="M 8.7185878,4.0337352 L -2.2072895,0.016013256 L 8.7185884,-4.0017078 C 6.9730900,-1.6296469 6.9831476,1.6157441 8.7185878,4.0337352 z " style="fill-rule:evenodd;fill:context-stroke;stroke-width:0.62500000;stroke-linejoin:round"/>
    </marker>`,
	"Arrow2Lend": `<marker style="overflow:visible;" id="Arrow2Lend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow2Lend">
      <path transform="scale(1.1) rotate(180) translate(1,0)" d="M 8.7185878,4.0337352 L -2.2072895,0.016013256 L 8.7185884,-4.0017078 C 6.9730900,-1.6296469 6.9831476,1.6157441 8.7185878,4.0337352 z " style="fill-rule:evenodd;fill:context-stroke;stroke-width:0.62500000;stroke-linejoin:round;"/>
    </marker>`,

	"Arrow2Mstart": `<marker style="overflow:visible" id="Arrow2Mstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow2Mstart">
      <path transform="scale(0.6) translate(0,0)" d="M 8.7185878,4.0337352 L -2.2072895,0.016013256 L 8.7185884,-4.0017078 C 6.9730900,-1.6296469 6.9831476,1.6157441 8.7185878,4.0337352 z " style="fill-rule:evenodd;fill:context-stroke;stroke-width:0.62500000;stroke-linejoin:round"/>
    </marker>`,
	"Arrow2Mend": `<marker style="overflow:visible;" id="Arrow2Mend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow2Mend">
      <path transform="scale(0.6) rotate(180) translate(0,0)" d="M 8.7185878,4.0337352 L -2.2072895,0.016013256 L 8.7185884,-4.0017078 C 6.9730900,-1.6296469 6.9831476,1.6157441 8.7185878,4.0337352 z " style="fill-rule:evenodd;fill:context-stroke;stroke-width:0.62500000;stroke-linejoin:round;"/>
    </marker>`,

	"Arrow2Sstart": `<marker style="overflow:visible" id="Arrow2Sstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow2Sstart">
      <path transform="scale(0.3) translate(-2.3,0)" d="M 8.7185878,4.0337352 L -2.2072895,0.016013256 L 8.7185884,-4.0017078 C 6.9730900,-1.6296469 6.9831476,1.6157441 8.7185878,4.0337352 z " style="fill-rule:evenodd;fill:context-stroke;stroke-width:0.62500000;stroke-linejoin:round"/>
    </marker>`,
	"Arrow2Send": `<marker style="overflow:visible;" id="Arrow2Send" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Arrow2Send">
      <path transform="scale(0.3) rotate(180) translate(-2.3,0)" d="M 8.7185878,4.0337352 L -2.2072895,0.016013256 L 8.7185884,-4.0017078 C 6.9730900,-1.6296469 6.9831476,1.6157441 8.7185878,4.0337352 z " style="fill-rule:evenodd;fill:context-stroke;stroke-width:0.62500000;stroke-linejoin:round;"/>
    </marker>`,

	"Tail": `<marker style="overflow:visible" id="Tail" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Tail">
      <g transform="scale(-1.2)">
        <path style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:0.8;stroke-linecap:round" d="M -3.8048674,-3.9585227 L 0.54352094,0"/>
        <path style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:0.8;stroke-linecap:round" d="M -1.2866832,-3.9585227 L 3.0617053,0"/>
        <path style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:0.8;stroke-linecap:round" d="M 1.3053582,-3.9585227 L 5.6537466,0"/>
        <path style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:0.8;stroke-linecap:round" d="M -3.8048674,4.1775838 L 0.54352094,0.21974226"/>
        <path style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:0.8;stroke-linecap:round" d="M -1.2866832,4.1775838 L 3.0617053,0.21974226"/>
        <path style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:0.8;stroke-linecap:round" d="M 1.3053582,4.1775838 L 5.6537466,0.21974226"/>
      </g>
    </marker>`,

	"DistanceStart": `<marker inkscape:stockid="DistanceStart" orient="auto" refY="0.0" refX="0.0" id="DistanceStart" style="overflow:visible">
      <g id="g2300">
        <path id="path2306" d="M 0,0 L 2,0" style="fill:none;stroke:context-fill;stroke-width:1.15;stroke-linecap:square"/>
        <path id="path2302" d="M 0,0 L 13,4 L 9,0 13,-4 L 0,0 z " style="fill:context-stroke;fill-rule:evenodd;stroke:none"/>
        <path id="path2304" d="M 0,-4 L 0,40" style="fill:none;stroke:context-stroke;stroke-width:1;stroke-linecap:square"/>
      </g>
    </marker>`,
	"DistanceEnd": `<marker inkscape:stockid="DistanceEnd" orient="auto" refY="0.0" refX="0.0" id="DistanceEnd" style="overflow:visible">
      <g id="g2301">
        <path id="path2316" d="M 0,0 L -2,0" style="fill:none;stroke:context-fill;stroke-width:1.15;stroke-linecap:square"/>
        <path id="path2312" d="M 0,0 L -13,4 L -9,0 -13,-4 L 0,0 z " style="fill:context-stroke;fill-rule:evenodd;stroke:none"/>
        <path id="path2314" d="M 0,-4 L 0,40" style="fill:none;stroke:context-stroke;stroke-width:1;stroke-linecap:square"/>
      </g>
    </marker>`,

	"DotL": `<marker style="overflow:visible" id="DotL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DotL">
      <path transform="scale(0.8) translate(7.4, 1)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M -2.5,-1.0 C -2.5,1.7600000 -4.7400000,4.0 -7.5,4.0 C -10.260000,4.0 -12.5,1.7600000 -12.5,-1.0 C -12.5,-3.7600000 -10.260000,-6.0 -7.5,-6.0 C -4.7400000,-6.0 -2.5,-3.7600000 -2.5,-1.0 z "/>
    </marker>`,
	"DotM": `<marker style="overflow:visible" id="DotM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DotM">
      <path transform="scale(0.4) translate(7.4, 1)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M -2.5,-1.0 C -2.5,1.7600000 -4.7400000,4.0 -7.5,4.0 C -10.260000,4.0 -12.5,1.7600000 -12.5,-1.0 C -12.5,-3.7600000 -10.260000,-6.0 -7.5,-6.0 C -4.7400000,-6.0 -2.5,-3.7600000 -2.5,-1.0 z "/>
    </marker>`,
	"DotS": `<marker style="overflow:visible" id="DotS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DotS">
      <path transform="scale(0.2) translate(7.4, 1)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M -2.5,-1.0 C -2.5,1.7600000 -4.7400000,4.0 -7.5,4.0 C -10.260000,4.0 -12.5,1.7600000 -12.5,-1.0 C -12.5,-3.7600000 -10.260000,-6.0 -7.5,-6.0 C -4.7400000,-6.0 -2.5,-3.7600000 -2.5,-1.0 z "/>
    </marker>`,

	"SquareL": `<marker style="overflow:visible" id="SquareL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="SquareL">
      <path transform="scale(0.8)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M -5.0,-5.0 L -5.0,5.0 L 5.0,5.0 L 5.0,-5.0 L -5.0,-5.0 z "/>
    </marker>`,
	"SquareM": `<marker style="overflow:visible" id="SquareM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="SquareM">
      <path transform="scale(0.4)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M -5.0,-5.0 L -5.0,5.0 L 5.0,5.0 L 5.0,-5.0 L -5.0,-5.0 z "/>
    </marker>`,
	"SquareS": `<marker style="overflow:visible" id="SquareS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="SquareS">
      <path transform="scale(0.2)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M -5.0,-5.0 L -5.0,5.0 L 5.0,5.0 L 5.0,-5.0 L -5.0,-5.0 z "/>
    </marker>`,

	"DiamondL": `<marker style="overflow:visible" id="DiamondL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondL">       
      <path transform="scale(0.8)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"DiamondM": `<marker style="overflow:visible" id="DiamondM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondM">
      <path transform="scale(0.4)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"DiamondS": `<marker style="overflow:visible" id="DiamondS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondS">
      <path transform="scale(0.2)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,

	"DiamondLstart": `<marker style="overflow:visible" id="DiamondLstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondLstart">       
      <path transform="scale(0.8) translate(7,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"DiamondMstart": `<marker style="overflow:visible" id="DiamondMstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondMstart">
      <path transform="scale(0.4) translate(6.5,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"DiamondSstart": `<marker style="overflow:visible" id="DiamondSstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondSstart">
      <path transform="scale(0.2) translate(6,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,

	"DiamondLend": `<marker style="overflow:visible" id="DiamondLend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondLend">       
      <path transform="scale(0.8) translate(-7,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"DiamondMend": `<marker style="overflow:visible" id="DiamondMend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondMend">
      <path transform="scale(0.4) translate(-6.5,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"DiamondSend": `<marker style="overflow:visible" id="DiamondSend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="DiamondSend">
      <path transform="scale(0.2) translate(-6,0)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,

	"EmptyDiamondL": `<marker style="overflow:visible" id="EmptyDiamondL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondL">       
      <path transform="scale(0.8)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"EmptyDiamondM": `<marker style="overflow:visible" id="EmptyDiamondM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondM">
      <path transform="scale(0.4)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"EmptyDiamondS": `<marker style="overflow:visible" id="EmptyDiamondS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondS">
      <path transform="scale(0.2)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,

	"EmptyDiamondLstart": `<marker style="overflow:visible" id="EmptyDiamondLstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondLstart">       
      <path transform="scale(0.8) translate(7,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"EmptyDiamondMstart": `<marker style="overflow:visible" id="EmptyDiamondMstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondMstart">
      <path transform="scale(0.4) translate(6.5,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"EmptyDiamondSstart": `<marker style="overflow:visible" id="EmptyDiamondSstart" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondSstart">
      <path transform="scale(0.2) translate(6,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"EmptyDiamondLend": `<marker style="overflow:visible" id="EmptyDiamondLend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondLend">       
      <path transform="scale(0.8) translate(-7,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"EmptyDiamondMend": `<marker style="overflow:visible" id="EmptyDiamondMend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondMend">
      <path transform="scale(0.4) translate(-6.5,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,
	"EmptyDiamondSend": `<marker style="overflow:visible" id="EmptyDiamondSend" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyDiamondSend">
      <path transform="scale(0.2) translate(-6,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 0,-7.0710768 L -7.0710894,0 L 0,7.0710589 L 7.0710462,0 L 0,-7.0710768 z "/>
    </marker>`,

	"TriangleInL": `<marker style="overflow:visible" id="TriangleInL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="TriangleInL">
      <path transform="scale(-0.8)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"TriangleInM": `<marker style="overflow:visible" id="TriangleInM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="TriangleInM">
      <path transform="scale(-0.4)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"TriangleInS": `<marker style="overflow:visible" id="TriangleInS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="TriangleInS">
      <path transform="scale(-0.2)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"TriangleOutL": `<marker style="overflow:visible" id="TriangleOutL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="TriangleOutL">
      <path transform="scale(0.8)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"TriangleOutM": `<marker style="overflow:visible" id="TriangleOutM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="TriangleOutM">
      <path transform="scale(0.4)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"TriangleOutS": `<marker style="overflow:visible" id="TriangleOutS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="TriangleOutS">
      <path transform="scale(0.2)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,

	"EmptyTriangleInL": `<marker style="overflow:visible" id="EmptyTriangleInL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyTriangleInL">
      <path transform="scale(-0.8) translate(-6,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"EmptyTriangleInM": `<marker style="overflow:visible" id="EmptyTriangleInM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyTriangleInM">
      <path transform="scale(-0.4) translate(-4.5,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"EmptyTriangleInS": `<marker style="overflow:visible" id="EmptyTriangleInS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyTriangleInS">
      <path transform="scale(-0.2) translate(-3.0,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"EmptyTriangleOutL": `<marker style="overflow:visible" id="EmptyTriangleOutL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyTriangleOutL">
      <path transform="scale(0.8) translate(-6,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"EmptyTriangleOutM": `<marker style="overflow:visible" id="EmptyTriangleOutM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyTriangleOutM">
      <path transform="scale(0.4) translate(-4.5,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,
	"EmptyTriangleOutS": `<marker style="overflow:visible" id="EmptyTriangleOutS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="EmptyTriangleOutS">
      <path transform="scale(0.2) translate(-3.0,0)" style="fill-rule:evenodd;fill:context-fill;stroke:context-stroke;stroke-width:1.0pt" d="M 5.77,0.0 L -2.88,5.0 L -2.88,-5.0 L 5.77,0.0 z "/>
    </marker>`,

	"StopL": `<marker style="overflow:visible" id="StopL" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="StopL">
      <path transform="scale(0.8)" style="fill:none;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0pt" d="M 0.0,5.65 L 0.0,-5.65"/>
    </marker>`,
	"StopM": `<marker style="overflow:visible" id="StopM" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="StopM">
      <path transform="scale(0.4)" style="fill:none;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0pt" d="M 0.0,5.65 L 0.0,-5.65"/>
    </marker>`,
	"StopS": `<marker style="overflow:visible" id="StopS" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="StopS">
      <path transform="scale(0.2)" style="fill:none;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0pt" d="M 0.0,5.65 L 0.0,-5.65"/>
    </marker>`,

	"SemiCircleIn": `<marker style="overflow:visible" id="SemiCircleIn" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="SemiCircleIn">
      <path transform="scale(0.6)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M -0.37450702,-0.045692580 C -0.37450702,2.7143074 1.8654930,4.9543074 4.6254930,4.9543074 L 4.6254930,-5.0456926 C 1.8654930,-5.0456926 -0.37450702,-2.8056926 -0.37450702,-0.045692580 z "/>
    </marker>`,
	"SemiCircleOut": `<marker style="overflow:visible" id="SemiCircleOut" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="SemiCircleOut">
      <path transform="scale(0.6) translate(7.125493,0.763446)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:1.0pt" d="M -2.5,-0.80913858 C -2.5,1.9508614 -4.7400000,4.1908614 -7.5,4.1908614 L -7.5,-5.8091386 C -4.7400000,-5.8091386 -2.5,-3.5691386 -2.5,-0.80913858 z "/>
    </marker>`,

	"CurveIn": `<marker style="overflow:visible" id="CurveIn" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="CurveIn">
      <path transform="scale(0.6)" style="fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0pt;fill:none" d="M 4.6254930,-5.0456926 C 1.8654930,-5.0456926 -0.37450702,-2.8056926 -0.37450702,-0.045692580 C -0.37450702,2.7143074 1.8654930,4.9543074 4.6254930,4.9543074"/>
    </marker>`,
	"CurveOut": `<marker style="overflow:visible" id="CurveOut" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="CurveOut">
      <path transform="scale(0.6)" style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0pt" d="M -5.4129913,-5.0456926 C -2.6529913,-5.0456926 -0.41299131,-2.8056926 -0.41299131,-0.045692580 C -0.41299131,2.7143074 -2.6529913,4.9543074 -5.4129913,4.9543074"/>
    </marker>`,
	"CurvyCross": `<marker style="overflow:visible" id="CurvyCross" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="CurvyCross">
      <g transform="scale(0.6)">
        <path style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0pt" d="M 4.6254930,-5.0456926 C 1.8654930,-5.0456926 -0.37450702,-2.8056926 -0.37450702,-0.045692580 C -0.37450702,2.7143074 1.8654930,4.9543074 4.6254930,4.9543074"/>
        <path style="fill:none;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0pt" d="M -5.4129913,-5.0456926 C -2.6529913,-5.0456926 -0.41299131,-2.8056926 -0.41299131,-0.045692580 C -0.41299131,2.7143074 -2.6529913,4.9543074 -5.4129913,4.9543074"/>
      </g>
    </marker>`,

	"Scissors": `<marker style="overflow:visible" id="Scissors" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Scissors">
      <path style="fill:context-stroke;" d="M 9.0898857,-3.6061018 C 8.1198849,-4.7769976 6.3697607,-4.7358294 5.0623558,-4.2327734 L -3.1500488,-1.1548705 C -5.5383421,-2.4615840 -7.8983361,-2.0874077 -7.8983361,-2.7236578 C -7.8983361,-3.2209742 -7.4416699,-3.1119800 -7.5100293,-4.4068519 C -7.5756648,-5.6501286 -8.8736064,-6.5699315 -10.100428,-6.4884954 C -11.327699,-6.4958500 -12.599867,-5.5553341 -12.610769,-4.2584343 C -12.702194,-2.9520479 -11.603560,-1.7387447 -10.304005,-1.6532027 C -8.7816644,-1.4265411 -6.0857470,-2.3487593 -4.8210600,-0.082342643 C -5.7633447,1.6559151 -7.4350844,1.6607341 -8.9465707,1.5737277 C -10.201445,1.5014928 -11.708664,1.8611256 -12.307219,3.0945882 C -12.885586,4.2766744 -12.318421,5.9591904 -10.990470,6.3210002 C -9.6502788,6.8128279 -7.8098011,6.1912892 -7.4910978,4.6502760 C -7.2454393,3.4624530 -8.0864637,2.9043186 -7.7636052,2.4731223 C -7.5199917,2.1477623 -5.9728246,2.3362771 -3.2164999,1.0982979 L 5.6763468,4.2330688 C 6.8000164,4.5467672 8.1730685,4.5362646 9.1684433,3.4313614 L -0.051640930,-0.053722219 L 9.0898857,-3.6061018 z M -9.2179159,-5.5066058 C -7.9233569,-4.7838060 -8.0290767,-2.8230356 -9.3743431,-2.4433169 C -10.590861,-2.0196559 -12.145370,-3.2022863 -11.757521,-4.5207817 C -11.530373,-5.6026336 -10.104134,-6.0014137 -9.2179159,-5.5066058 z M -9.1616516,2.5107591 C -7.8108215,3.0096239 -8.0402087,5.2951947 -9.4138723,5.6023681 C -10.324932,5.9187072 -11.627422,5.4635705 -11.719569,4.3902287 C -11.897178,3.0851737 -10.363484,1.9060805 -9.1616516,2.5107591 z " id="schere"/>
    </marker>`,

	"Legs": `<marker style="overflow:visible" id="Legs" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Legs">
      <g transform="scale(-0.7)">
        <g transform="matrix(0,-1.000000,-1.000000,0,20.70862,21.31391)">
          <path style="fill:none;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0000000pt" d="M 21.221250,20.675360 C 14.311099,25.396517 18.766725,27.282204 15.380179,34.118595"/>
          <path style="fill:none;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0000000pt" d="M 21.398110,20.548120 C 20.037601,28.895644 24.934182,29.318060 25.903151,34.373078"/>
        </g>
        <path style="fill:context-stroke;fill-rule:evenodd;stroke-width:1.0000000pt" d="M -14.090070,-6.7318716 L -15.012238,-2.6884886 L -11.049487,-3.9115586 L -14.090070,-6.7318716 z "/>
        <path style="fill:context-stroke;fill-rule:evenodd;stroke-width:1.0000000pt" d="M -15.215679,4.5567534 L -13.341552,8.2563664 L -11.074678,4.7835114 L -15.215679,4.5567534 z "/>
      </g>
    </marker>`,

	"Torso": `<marker style="overflow:visible" id="Torso" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Torso">
      <g transform="scale(0.7)">
        <path style="fill:none;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.2500000" d="M -4.7792281,-3.2395420 C -2.4288541,-2.8736027 0.52103922,-1.3019943 0.25792722,0.38794346 C -0.0051877922,2.0778819 -2.2126741,2.6176539 -4.5630471,2.2517169 C -6.9134221,1.8857769 -8.5210350,0.75201414 -8.2579220,-0.93792336 C -7.9948090,-2.6278615 -7.1296041,-3.6054813 -4.7792281,-3.2395420 z "/>
        <path style="fill:none;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0000000pt" d="M 4.4598789,0.088665736 C -2.5564571,-4.3783320 5.2248769,-3.9061806 -0.84829578,-8.7197331"/>
        <path style="fill:none;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0000000pt" d="M 4.9298719,0.057520736 C -1.3872731,1.7494689 1.8027579,5.4782079 -4.9448731,7.5462725"/>
        <rect style="fill-rule:evenodd;fill:context-stroke;stroke-width:1.0000000pt" width="2.6366582" height="2.7608147" x="-10.391706" y="-1.7408575" transform="matrix(0.527536,-0.849533,0.887668,0.460484,0,0)"/>
        <rect style="fill-rule:evenodd;fill:context-stroke;stroke-width:1.0000000pt" width="2.7327356" height="2.8614161" x="4.9587269" y="-7.9629307" transform="matrix(0.671205,-0.741272,0.790802,0.612072,0,0)"/>
        <path style="fill:#ff0000;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0000000pt" d="M 16.779951 -28.685045 A 0.60731727 0.60731727 0 1 0 15.565317,-28.685045 A 0.60731727 0.60731727 0 1 0 16.779951 -28.685045 z" transform="matrix(0,-1.109517,1.109517,0,25.96648,19.71619)"/>
        <path style="fill:#ff0000;fill-opacity:0.75000000;fill-rule:evenodd;stroke:context-stroke;stroke-width:1.0000000pt" d="M 16.779951 -28.685045 A 0.60731727 0.60731727 0 1 0 15.565317,-28.685045 A 0.60731727 0.60731727 0 1 0 16.779951 -28.685045 z" transform="matrix(0,-1.109517,1.109517,0,26.82450,16.99126)"/>
      </g>
    </marker>`,

	"Club": `<marker style="overflow:visible" id="Club" refX="0.0" refY="0.0" orient="auto" inkscape:stockid="Club">
      <path transform="scale(0.6)" style="fill-rule:evenodd;fill:context-stroke;stroke:context-stroke;stroke-width:0.74587913pt" d="M -1.5971367,-7.0977635 C -3.4863874,-7.0977635 -5.0235187,-5.5606321 -5.0235187,-3.6713813 C -5.0235187,-3.0147015 -4.7851656,-2.4444556 -4.4641095,-1.9232271 C -4.5028609,-1.8911157 -4.5437814,-1.8647646 -4.5806531,-1.8299921 C -5.2030765,-2.6849849 -6.1700514,-3.2751330 -7.3077730,-3.2751330 C -9.1970245,-3.2751331 -10.734155,-1.7380016 -10.734155,0.15124914 C -10.734155,2.0404999 -9.1970245,3.5776313 -7.3077730,3.5776313 C -6.3143268,3.5776313 -5.4391540,3.1355702 -4.8137404,2.4588126 C -4.9384274,2.8137041 -5.0235187,3.1803000 -5.0235187,3.5776313 C -5.0235187,5.4668819 -3.4863874,7.0040135 -1.5971367,7.0040135 C 0.29211394,7.0040135 1.8292454,5.4668819 1.8292454,3.5776313 C 1.8292454,2.7842354 1.5136868,2.0838028 1.0600576,1.5031550 C 2.4152718,1.7663868 3.7718375,2.2973711 4.7661444,3.8340272 C 4.0279463,3.0958289 3.5540908,1.7534117 3.5540908,-0.058529361 L 2.9247554,-0.10514681 L 3.5074733,-0.12845553 C 3.5074733,-1.9403966 3.9580199,-3.2828138 4.6962183,-4.0210121 C 3.7371277,-2.5387813 2.4390549,-1.9946496 1.1299838,-1.7134486 C 1.5341802,-2.2753578 1.8292454,-2.9268556 1.8292454,-3.6713813 C 1.8292454,-5.5606319 0.29211394,-7.0977635 -1.5971367,-7.0977635 z "/>
    </marker>`,

	"RazorWire": `<marker style="overflow:visible" orient="auto" refY="0" refX="0" id="RazorWire" inkscape:stockid="RazorWire">
      <path d="M 0.022727273,-0.74009011 L 0.022727273,0.69740989 L -7.7585227,3.0099099 L 10.678977,3.0099099 L 3.4914773,0.69740989 L 3.4914773,-0.74009011 L 10.741477,-2.8963401 L -7.7272727,-2.8963401 L 0.022727273,-0.74009011 z " style="fill:#808080;fill-opacity:1;fill-rule:evenodd;stroke:context-stroke;stroke-width:0.1pt" transform="scale(0.8,0.8)"/>
    </marker>`,

	"InfiniteLineEnd": `<marker orient="auto" refY="0" refX="0" id="InfiniteLineEnd" inkscape:stockid="InfiniteLineEnd" style="overflow:visible">
      <g style="fill:context-stroke">
        <circle cx="3" cy="0" r="0.8"/>
        <circle cx="6.5" cy="0" r="0.8"/>
        <circle cx="10" cy="0" r="0.8"/>
      </g>
    </marker>`,
	"InfiniteLineStart": `<marker orient="auto" refY="0" refX="0" id="InfiniteLineStart" inkscape:stockid="InfiniteLineStart" style="overflow:visible">
      <g transform="translate(-13,0)" style="fill:context-stroke">
        <circle cx="3" cy="0" r="0.8"/>
        <circle cx="6.5" cy="0" r="0.8"/>
        <circle cx="10" cy="0" r="0.8"/>
      </g>
    </marker>`,
}
