# grid

Grid is a GoGi-based SVG vector drawing program, with design based on Inkscape.

If an acronym is required, how about: "Go-rendered interactive drawing" program.

# Behavior

* multiple select actions keep doing down even inside groups, so it is easy to operate inside groups but group is the "default"

* Alt on control knobs -> rotate instead of clicking again to get rotation knobs -- this is compatible with above and better :)

# Install

The simple Go install command should work, for modules mode: 

```bash
$ go get github.com/goki/grid/cmd/grid
```

See [GoKi Install](https://github.com/goki/gi/wiki/Install) for more information and prereqs for different platforms -- if you encounter any difficulties, ensure that GoKi works first.

Exporting PNG and PDF depends on https://cairosvg.org (which depends on python), install as follows:

```bash
$ pip3 install cairosvg
```

# Design

Similar to inkscape in overall layout, and read / write inkscape compatible SVG files.

* Main horiz toolbar(s) across top -- top one is static, bottom one is dynamic based on selection / tool mode.

* Left vert toolbar with drawing tools

* Left panel with drawing structure.  This is just like GiEditor tree -- provides direct access as needed.  In particular this provides layer-level organization -- always have a layer group by default, and can drag items into different layers, and also have view and freeze flags on layers.  usu only show layer level, and selection there determines which layer things are added to!

* Main middle panel with drawing.  Grid underlay is a separate image that is drawn first, updated with any changes.

* Right tab panel with all the controls, just like gide in terms of tab & Inkscape overall. tabs are easier to find vs. inkscape.

* code in main grid package provides all the editors for right tabs.

# Status

Basic functionality now in place:

* create: rect, ellipse, line, text, import image

* full basic paint settings (gradients, markers, etc), and text properties, editing

* dynamic guide alignment, Align panel

* basic node editor -- can move the main points, not the extra control points

* full undo / redo for everything.

* Preferences 

# TODO:

* import svg -- same as marker

* implement the full transform panel

* render gradient icons

* show selected path node in diff color..  red?  node sel map

* rest of line props -- easy

* dropper = grab style from containsnode, apply to selection -- don't affect selection!

* svg.Node ToPath -- convert any node to a path
* node editor -- big job but needed for making basic bezier curves..

* esc aborts new el drag

* add distribute to Align

* svg render needs to use visibility flag from layers to not render stuff.
* generic display: flag -- not same as setting visible -- all levels
  need to process that flag.

* svg.Text align Center, etc affects different tspans within overall text block
* Text edit panel -- finish toolbar

* cut / paste not updating tree reliably.  more tree update debugging fun!

* use grid to render all new icons!

* figure out mask clipping eventually.


# LINKS

Inkscape special flags

https://wiki.inkscape.org/wiki/index.php/Inkscape_SVG_vs._plain_SVG


