# gide

![alt tag](logo/gide_icon.png)

**Gide** is a flexible IDE (integrated development environment) framework in pure Go, using the [GoGi](https://github.com/goki/gi) GUI and the [GoPi](https://github.com/goki/pi) interactive parser for syntax highlighting and more advanced IDE code processing.

See the [Wiki](https://github.com/goki/gide/wiki) for more docs,   [Install](https://github.com/goki/gide/wiki/Install) instructions (just standard `go get ...`), and [Google Groups goki-gi](https://groups.google.com/forum/#!forum/goki-gi) emailing list.

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/gide)](https://goreportcard.com/report/github.com/goki/gide)
[![GoDoc](https://godoc.org/github.com/goki/gide?status.svg)](https://godoc.org/github.com/goki/gide)
[![Travis](https://travis-ci.com/goki/gide.svg?branch=master)](https://travis-ci.com/goki/gide)

After all these years, nothing beats a great text editor for coding.  Every attempt at more "visual" or structured forms of programming haven't really caught on (though the IDE and GUI-augmented editor can give you the best of both worlds).

And nothing beats coding for efficiently doing just about anything you want to do, whether it is data analysis, AI, etc (and obviously for "regular" coding).

Even writing documents is best done in a markup language (markdown, LaTeX, etc), and needs a great text editor.  In short, virtually your entire workflow as a scientist, researcher, etc depends on the same core functionality.

But does the world need yet-another-IDE? Probably not, but at one point we thought it was on our critical path for developing our mission-critical [emergent](https://github.com/emer/emergent) neural network simulation environment, so here it is!

Some of the main current / planned features of *Gide* include:

* Pure opensource Go (golang) implementation, built on top of a Go-based cross-platform GUI framework: [GoGi](https://github.com/goki/gi).  Go means you can read and understand the code, and it compiles in seconds, so if you want to customize or fork to make your own personal favorite IDE, this could be a good starting point!

* Designed from the ground up to handle a wide range of use-cases, from core coding to scientific computing to writing documents, etc.  Handles heterogenous file types all mixed together in a given project, supports customizable commands that can be associated with different file types, etc.

* Centered around a `FileTree`-based file browser on the left, which supports full version control and standard file management functionality etc.  The plan is to make this a highly useful but still simple and straightforward file browser interface that can handle all the basic tasks.

* Command actions show output on a tabbed output display on the right, along with other special interfaces such as Find / Replace, Symbols, etc.  Overall design works well without having to constantly move widgets and panels around, although we'll have to see how well it scales.

* Very keyboard focused, and especially usable with standard Emacs command sequences -- emacs users will feel very at home, but also perhaps excited to have a more complete GUI experience as well.  Everything is customizable to fit whatever framework you're familiar with (and file tickets if we can me it better).

* The original plan was to add functionality comparable to `JupyterLab` and other such scientific computing frameworks (`nteract`, R studio, etc), where you can easily pop up advanced 2D and 3D graphics, and powerful interactive GUI interfaces to all manner of data types and structures.  Not sure if that will actually happen...

# Current Status

As of 11/2018, it is fully functional as an editor, but many planned features remain to be implemented (still true as of 6/2019 -- will get back to it after all the other higher-priority stuff :)

Basic editing and tooling for `Go`, `C`, `LaTeX`, `Markdown` is reasonably functional and solid.  It is fully self-hosting -- further development of Gide (and GoGi etc) is happening within Gide!

Near-term major goals (i.e., these are not yet implemented):
* Fix the `FileTree` to be rock solid.  Currently has a few issues..
* Support for `delve` debugger for Go.  Then `lldb` after that maybe.  And see about python debugging.
* Extend the [GoPi](https://github.com/goki/pi) interactive parser to handle Python and maybe C / C++, so it will have been worth writing instead of just using the native Go parser.

Maybe / maybe not:
* `Jupyter Lab` / `nteract` level functionality.  How hard would it be?  Probably need to get GoGi plugged in as a backend to matplotlib etc -- could be a big job..

Feel free to file issues for anything you'd like to see that isn't listed here.

# Screenshots

![Screenshot](screenshot.png?raw=true "Screenshot")

![Screenshot, darker](screenshot_dark.png?raw=true "Screenshot, darker color scheme")
