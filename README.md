# gide

![alt tag](logo/gide_icon.png)

**Gide** is a flexible IDE (integrated development environment) framework in pure Go, using the [GoGi](https://github.com/goki/gi) GUI.

See the [Wiki](https://github.com/goki/gide/wiki) for more docs,   [Install](https://github.com/goki/gide/wiki/Install) instructions (just standard `go get ...`), and [Google Groups goki-gi](https://groups.google.com/forum/#!forum/goki-gi) emailing list.

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/gide)](https://goreportcard.com/report/github.com/goki/gide)
[![GoDoc](https://godoc.org/github.com/goki/gide?status.svg)](https://godoc.org/github.com/goki/gide)

After all these years, nothing beats a great text editor for coding.  Every attempt at more "visual" or structured forms of programming haven't really caught on (though the IDE and GUI-augmented editor can give you the best of both worlds).

And nothing beats coding for efficiently doing just about anything you want to do, whether it is data analysis, AI, etc (and obviously for "regular" coding).

Even writing documents is best done in a markup language (markdown, LaTeX, etc), and needs a great text editor.  In short, virtually your entire workflow as a scientist, researcher, etc depends on the same core functionality.

And yet, the perfect text editor / IDE has yet to be written... *until now!*

* `Sublime` lives up to its name according to many, but it is proprietary..
* `Atom` is open and very popular, but... electron.. javascript.. ugh..
* `Emacs` is.. complicated.. and.. lisp?
* `IntelliJ` is also very well done, but also proprietary and has some kind of [crazy bug](https://intellij-support.jetbrains.com/hc/en-us/community/posts/115000693290-Extreme-lag-and-high-CPU-usage-on-OSX-High-Sierra?page=2#comments) on Mac that has been around for years, driving high CPU loads.. 

Hence, the need for *Gide*, which features:

* Pure opensource Go (golang) implementation, built on top of a brand new, very clean, lightweight, fast cross-platform GUI framework: [GoGi](https://github.com/goki/gi).

* Designed from the ground up to handle a very wide range of use-cases, from core coding to scientific computing to writing documents, etc.

* A powerful text editor with advanced completion / code awareness is the core, but as in `JupyterLab` and other such scientific computing frameworks (`nteract`, R studio, etc), you can easily pop up advanced 2D and 3D graphic, and powerful interactive GUI interfaces to all manner of data types and structures.  The standard IDE tools (debugging, etc) are just one instance of the wide range of add-on functionality that easily be accessed within the gide system.

* Another critical design element is the world's best tab-view framework for holding and efficiently finding and using all the those extra displays and tools.

# Current Status

As of 11/2018, it is fully functional as an editor, but many planned features remain to be implemented.

Basic editing and tooling for `Go`, `C`, `LaTeX` is in reasonably functional and solid.  It is fully self-hosting -- all further development of Gide is happening within Gide!

Near-term major goals (i.e., these are not yet implemented):
* Connect to Python interpreter, run e.g., PyTorch, display graphic output in visualizer tab.
* And same for gonum & gomacro.
* Support for `delve` debugger for Go.  Then `lldb` after that maybe.  And see about python debugging.
* See about our own dynamic parsing framework within GoKi, for general dynamic structured language support.
* Native GoGi 3D and interactive visualizations.

Feel free to file issues for anything you'd like to see that isn't listed here.

# Design Goals

* Although implemented in Go, and that will obviously have most-favored status for language support, the goal is to make it as general as possible, with REPL support for various interpreted languages, and Go via https://github.com/cosmos72/gomacro (similar to https://github.com/gopherdata/gophernotes for `Jupyter` and `nteract`).

* Initially we are relying on basic syntax highlighting via [chroma](https://github.com/alecthomas/chroma), but to provide more advanced IDE-level functionality, a flexible dynamic parsing framework is envisioned, based on the GoKi tree (ki) structures.  This will provide multi-pass robust AST (abstract syntax tree) level parsing of supported languages, and the goal is to make the parser fully GUI editable to support "easy" extension to new languages.

* Again see the [Wiki](https://github.com/goki/gide/wiki) for much more info about installation, usage, etc.

# Screenshots

![Screenshot](screenshot.png?raw=true "Screenshot")

![Screenshot, darker](screenshot_dark.png?raw=true "Screenshot, darker color scheme")
