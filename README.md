# grid

Grid is a GoGi-based SVG vector drawing program, with design based on Inkscape.

If an acronym is required, how about: "Go-rendered interactive drawing" program.

# plans

Similar to inkscape in overall layout, and read / write inkscape compatible SVG files.

* Main horiz toolbar(s) across top -- top one is static, bottom one is dynamic based on selection / tool mode.

* Left vert toolbar with drawing tools

* Left panel with drawing structure.  This is just like GiEditor tree -- should be trivial, and provides direct access as needed.  In particular this provides layer-level organization -- always have a layer group by default, and can drag items into different layers, and also have view and freeze flags on layers.  usu only show layer level, and selection there determines which layer things are added to!

* Main middle panel with drawing.  Have an optional grid underlay?  Need to figure out good gpu-based way to do that -- texture is required, similar to scene?  do this later.

* Right tab panel with all the controls, just like gide in terms of tab & Inkscape overall. tabs are easier to find vs. inkscape.

* code in main grid package provides all the editors for right tabs.

# Status

Minimally functional with making new basic shapes (rect, ellipse) and reshaping anything,
color, fill and line width editor, and full undo / redo.

# TODO:

* layer flags, layer select via context menu -- see inkscape for key props

* svg render needs to use visibility flag from layers to not render stuff.

* move, reshape transforms on rotated obj not correct (pink guy)

* resize to fit content button

* Right click with option to show in tree

* splits code import from gide, and do prefs, add to menu, etc

* save prefs as "base" thing per inkscape

* objects with their own unique gradient transforms need the gradient updated when repositioned.  
  gradientUnits = "userSpaceOnUse" -- gradients refer to a master that defines stops, and then each
  object has their own specific gradient instance.  kinda lame, but probably inevitable without using bbox.
    + gradient editor can enable naming

* svg.Node ToPath -- convert any node to a path

* rubber-band select items in box 
    rubber band is just 4 separate sprites arranged in box, render dashed for contrast

* continued clicking = select deeper items
    
* alignview

* grid -- multiscale if spacing between grid items below some amount, then zoom out grid to 6x larger?

* dynamic alignment: precompute slice of key X coords, Y coords separately, acts as a kind of grid.

* svg.Text align Center, etc affects different tspans within overall text block
* svg.Text scale, rotate affects transform -- transform goes into style!

* figure out mask clipping eventually.

# LINKS

Inkscape special flags

https://wiki.inkscape.org/wiki/index.php/Inkscape_SVG_vs._plain_SVG


