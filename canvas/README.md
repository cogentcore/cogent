![alt tag](cmd/canvas/icon.svg)

[Cogent Canvas](https://cogentcore.org/cogent/canvas) is an SVG-based vector drawing program, with basic capabilities similar to [Inkscape](https://inkscape.org), although it currently lacks many of the more advanced features (see [TODO](#todo) list below for plans). The native file format is SVG, with optional Inkscape-based metadata to encode advanced style properties.

The Canvas interface is designed to make the high-frequency operations obvious and easy to access, and compared to Inkscape or Adobe Illustrator, it should generally be easier to use by a naive user. It also provides a full tree view into the underlying SVG structure, so you can easily directly manipulate it.

Because it is written in [Cogent Core](https://cogentcore.org), it also runs on the web and mobile devices.

## Behavior

* Multiple select actions keep doing down even inside groups, and you can always perform operations on elements inside groups, so you don't need to ungroup and re-group.

* To support the above behavior, you need to hold down the `Alt` key on control knobs to rotate, whereas Inkscape uses further clicking to switch between move and rotate.

## Install

You can use the [deployed web version](https://cogentcore.org/cogent/canvas). You can also use the simple Go install command:

```bash
$ go install cogentcore.org/cogent/canvas/cmd/canvas@main
```

Exporting to PDF currently depends on [Inkscape](https://inkscape.org). On the Mac you need to make a link to `/usr/local/bin` and likewise for Linux:

```bash
$ sudo ln -s /Applications/Inkscape.app/Contents/MacOS/inkscape /usr/local/bin/
```

## Status

* June, 2025: full basic functionality now in place, including drawing new paths and editing path control points.

## Tips

* Use layers! Incredibly useful for toggling editability and visibility of whole sets of elements. It is best to make the layers at the start and add as you go. Click on the layer to set which layer new elements are made in by default (duplicates always go into the same layer as the original).

## TODO

Bugs:

* images in svg are not clipping to bounds -- renderer should do this in pimage.

* still getting zero slider not sending change in setting a black color

### Simpler, near term

* ArcTo support in node editor, and arc tool.

* Gradient editor edits gradient control points.

* Figure out alternatives to modifier keys for for ipad.

* Transform panel for numerical rotate, scale etc.

* Dropper = grab style from containing node, apply to selection -- don't affect selection!

* Convert shape nodes to path: add `svg.Node` ToPath.

* Align panel: add distribute function.

* Better ways of managing Text with multiple tspan elements: styles for tspan, generate full Text and spans from a rich text source with line wrapping and HTML markup -- need bidirectional support to / from existing tspans etc. Basic pos movement of sub-tspan would be useful.

* Clip mask in `core.SVG` finally.

* Import svg -- same as marker (copy/paste is now working across drawings, so that is good).

* Path effects menu / chooser and add calls to the various existing ppath `intersect` and `stroke` functions.

### Longer term

* More natural drawing modes like freehand and calligraphy.

* More advanced path effects.

* More advanced drawing tools like grids, connected diagram elements (key!), etc.

## Links

* Inkscape special flags: https://wiki.inkscape.org/wiki/index.php/Inkscape_SVG_vs._plain_SVG


