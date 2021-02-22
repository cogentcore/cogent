# grid

Grid is a GoGi-based SVG vector drawing program, with design based on Inkscape.

If an acronym is required, how about: "Go-rendered interactive drawing" program.

# Behavior

* multiple select actions keep doing down even inside groups, so it is easy to operate inside groups but group is the "default"

* Alt on control knobs -> rotate instead of clicking again to get rotation knobs -- this is compatible with above and better :)

# Design

Similar to inkscape in overall layout, and read / write inkscape compatible SVG files.

* Main horiz toolbar(s) across top -- top one is static, bottom one is dynamic based on selection / tool mode.

* Left vert toolbar with drawing tools

* Left panel with drawing structure.  This is just like GiEditor tree -- provides direct access as needed.  In particular this provides layer-level organization -- always have a layer group by default, and can drag items into different layers, and also have view and freeze flags on layers.  usu only show layer level, and selection there determines which layer things are added to!

* Main middle panel with drawing.  Grid underlay is a separate image that is drawn first, updated with any changes.

* Right tab panel with all the controls, just like gide in terms of tab & Inkscape overall. tabs are easier to find vs. inkscape.

* code in main grid package provides all the editors for right tabs.

# Status

Minimally functional with making new basic shapes (rect, ellipse) and reshaping anything,
color, fill and line width editor, and full undo / redo.

# TODO:

* Text edit panel -- finish toolbar, add NewEl.

* fix url finding bug on inkscape.svg and track down gi.StyleSheet error

* esc aborts new el drag

* import svg -- same as marker

* changed bit, autosave, prompt before quit

* rest of shortcuts

* make zoom stay centered on mouse point -- subtract mouse pos then add back..

* svg render needs to use visibility flag from layers to not render stuff.
* generic display: flag -- not same as setting visible -- all levels
  need to process that flag.

* resize to fit content button

* save prefs as "base" thing per inkscape

* svg.Node ToPath -- convert any node to a path
* node editor -- big job but needed for making basic bezier curves..

* grid -- multiscale if spacing between grid items below some amount, then zoom out grid to 6x larger?

* svg.Text align Center, etc affects different tspans within overall text block
* svg.Text scale, rotate affects transform -- transform goes into style!

* some kind of mutex hang in undo / redo ?

* cut / paste not updating tree reliably.  more tree update debugging fun!

* use grid to render all new icons!

* figure out mask clipping eventually.

# LINKS

Inkscape special flags

https://wiki.inkscape.org/wiki/index.php/Inkscape_SVG_vs._plain_SVG


