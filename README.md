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

# TODO:

* resize to fit content button

* display filename in window header (update after opening file)

* SaveAs

* Right click with option to show in tree

* splits code import from gide, and do prefs, add to menu, etc

* save prefs as "base" thing per inkscape

* svg.Text bbox is wrong

* objects with their own unique gradient transforms need the gradient updated when repositioned.  
  gradientUnits = "userSpaceOnUse"

* svg.Node ToPath -- convert any node to a path

* shift + move = pan image, else rubber-band select items in box 
    rubber band is just 4 separate sprites arranged in box, render dashed for contrast

* continued clicking = select deeper items
    
* existing sprites not showing white outline contrast

* alignview

* grid

* dynamic alignment: precompute slice of key X coords, Y coords separately, acts as a kind of grid.

* rasterx radial gradients are in wrong position: maybe specific to ellipse / circle?  should be 
  in center.  finally debug this!

* figure out clipping eventually.


