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

* control knobs are sprites, need to add sprite events in gi so we can get mouse events on them.

* figure out clipping eventually.

# TODO:

* initial resize to fit content, and button to do that in any case

* paint editor

* svg.Text bbox is wrong

* svg.Node ToPath -- convert any node to a path


