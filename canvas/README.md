![alt tag](cmd/canvas/icon.svg)

Cogent Canvas is a SVG vector drawing program, with basic capabilities similar to [Inkscape](https://inkscape.org), although it currently lacks many of the more advanced features. The native file format is SVG.

The Canvas interface is designed to make the high-frequency operations obvious and easy to access, and compared to Inkscape or Adobe Illustrator, it should generally be easier to use by a naive user. It also provides a full tree view into the underlying SVG structure, so you can easily directly manipulate it.

Because it is written in [Cogent Core](https://cogentcore.org), it also runs on the web and mobile devices.

## Behavior

* Multiple select actions keep doing down even inside groups, and you can always perform operations on elements inside groups, so you don't need to ungroup and re-group.

* To support the above behavior, you need to hold down the `Alt` key on control knobs to rotate, whereas Inkscape uses further clicking to switch between move and rotate.

## Install

The simple Go install command should work:

```bash
$ go install cogentcore.org/cogent/canvas/cmd/canvas@main
```

Exporting to PDF currently depends on [Inkscape](https://inkscape.org). On the Mac you need to make a link to `/usr/local/bin` and likewise for Linux:

```bash
$ sudo ln -s /Applications/Inkscape.app/Contents/MacOS/inkscape /usr/local/bin/
```

## Status

* June, 2025: full basic functionality now in place, including drawing new paths and editing path control points.

# TODO:

* figure out alternatives to modifier keys for for ipad.

* selection based on more detailed contains logic: should be in tdewolff canvas somewhere?

* add group / ungroup to context menu (conditional on selection n etc).

* implement the full transform panel for numerical rotate, scale etc.

* dropper = grab style from containsnode, apply to selection -- don't affect selection!

* svg.Node ToPath -- convert any node to a path.

* add distribute to Align

* better ways of managing Text with multiple tspan elements.

* implement clip mask in `core.SVG`.

* import svg -- same as marker (copy paste is now working across drawings, so that is good).

## Links

* Inkscape special flags: https://wiki.inkscape.org/wiki/index.php/Inkscape_SVG_vs._plain_SVG


