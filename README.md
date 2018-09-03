# gide

**gide** is a flexible IDE (integrated development environment) framework in pure Go, using the GoGi GUI.  When CamelCase is needed, use `Gide`, but the command is `gide` etc.

After all these years, nothing beats a great text editor for coding.  All that drag-n-drop, graphical stuff just gets in the way.

And nothing beats coding for efficiently doing just about anything you want to do, whether it is data analysis, AI, etc (and obviously for "regular" coding).

Even writing documents in a markup language (markdown, LaTeX, etc) is best accomplished in a great text editor.

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

As of 9/2018, work has just begun.. This will quickly be self-hosting for all further development, so lots of dog food will be consumed and made ever-tastier!  Hoping to have some kind of actually-usable first pass in a few weeks.

In case it isn't obvious, the features listed above are goals.. :)  Feel free to file issues for anything you'd like to see that isn't listed here.

# Design Goals

* Although implemented in Go, and that will obviously have most-favored status for language support, the goal is to make it as general as possible, with REPL support for various interpreted languages, and Go via https://github.com/cosmos72/gomacro (similar to https://github.com/gopherdata/gophernotes for `Jupyter` and `nteract`.

* Initially will be relying on basic syntax highlighting via https://github.com/alecthomas/chroma, but to provide more advanced IDE-level functionality, a flexible dynamic parsing framework is envisioned, based on the GoKi tree (ki) structures.  This will provide multi-pass robust AST (abstract syntax tree) level parsing of supported languages, and the goal is to make the parser fully GUI editable to support "easy" extension to new languages.

