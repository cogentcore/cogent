# gide

![alt tag](logo/gide_icon.png)

**Gide** is a flexible IDE (integrated development environment) framework in pure Go, using the GoGi GUI.

See the [Wiki](https://github.com/goki/gide/wiki) for more docs, discussion, etc.

After all these years, nothing beats a great text editor for coding.  All that drag-n-drop, graphical stuff just gets in the way.

And nothing beats coding for efficiently doing just about anything you want to do, whether it is data analysis, AI, etc (and obviously for "regular" coding).

Even writing documents is best done in a markup language (markdown, LaTeX, etc), and needs a great text editor.  In short, virtually your entire workflow as a scientist, researcher, etc depends on the same core functionality.

And yet, the perfect text editor / IDE has yet to be written... *until now!* (or at least *N* years hence.. :)

* `Sublime` lives up to its name according to many, but it is proprietary..
* `Atom` is open and very popular, but... electron.. javascript.. ugh..
* `Emacs` is.. complicated.. and.. lisp?
* `IntelliJ` is also very well done, but also proprietary and has some kind of crazy bug on Mac that has been around for years, driving high CPU loads.. https://intellij-support.jetbrains.com/hc/en-us/community/posts/115000693290-Extreme-lag-and-high-CPU-usage-on-OSX-High-Sierra?page=2#comments

Hence, the need for *gide*, which features:

* Pure opensource Go (golang) implementation, built on top of brand new, very clean, lightweight, fast cross-platform GUI framework: GoGi (https://github.com/goki/gi).

* Designed from the ground up to handle a very wide range of use-cases, from core coding to scientific computing to writing documents, etc.

* A powerful text editor with advanced completion / code awareness is the core, but as in `JupyterLab` and other such scientific computing frameworks (`nteract`, R studio, etc), you can easily pop up advanced 2D and 3D graphic, and powerful interactive GUI interfaces to all manner of data types and structures.  The standard IDE tools (debugging, etc) are just one instance of the wide range of add-on functionality that easily be accessed within the gide system.

* Another critical design element is the world's best tab-view framework for holding and efficiently finding and using all the those extra displays and tools.

# Current Status

As of mid 10/2018, *nearly* ready for an initial beta release along with rest of GoGi.

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

* Initially will be relying on basic syntax highlighting via https://github.com/alecthomas/chroma, but to provide more advanced IDE-level functionality, a flexible dynamic parsing framework is envisioned, based on the GoKi tree (ki) structures.  This will provide multi-pass robust AST (abstract syntax tree) level parsing of supported languages, and the goal is to make the parser fully GUI editable to support "easy" extension to new languages.

# TODO

* running commands needs to be within project!  currently global.
