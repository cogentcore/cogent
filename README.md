# gide

![alt tag](logo/gide_icon.png)

**Gide** is a flexible IDE (integrated development environment) framework in pure Go, using the [GoGi](https://github.com/goki/gi) GUI (for which it serves as a continuous testing platform :) and the [GoPi](https://github.com/goki/pi) interactive parser for syntax highlighting and more advanced IDE code processing.

See the [Wiki](https://github.com/goki/gide/wiki) for more docs,   [Install](https://github.com/goki/gide/wiki/Install) instructions (mostly just `go get ...`), and [Google Groups goki-gi](https://groups.google.com/forum/#!forum/goki-gi) emailing list.

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/gide)](https://goreportcard.com/report/github.com/goki/gide)
[![GoDoc](https://godoc.org/github.com/goki/gide?status.svg)](https://godoc.org/github.com/goki/gide)
[![Travis](https://travis-ci.com/goki/gide.svg?branch=master)](https://travis-ci.com/goki/gide)

There are many existing, excellent choices for text editors and IDE's, but *Gide* is possibly the best option available with just a `go get ./...` command.  The Go language represents a special combination of simplicity, elegance, and power, and is a joy to program in, and is currently the main language fully-supported by Gide.  Our ambition is to capture some of those Go attributes in an IDE.

Some of the main features of *Gide* include:

* Designed to function as both a general-purpose text editor *and* an IDE.  It comes configured with command support for LaTeX, Markdown, and Makefiles, in addition to Go, and the command system is fully extensible to support any command-line tools.

* Provides a tree-based file browser on the left, with builtin support for version control (git, svn, etc) and standard file management functionality through drag-and-drop, etc.  You can look at git logs, diffs, etc through this interface.

* Command actions show output on a tabbed output display on the right, along with other special interfaces such as Find / Replace, Symbols, Debugger, etc.  Thus, the overall design is extensible and new interfaces can be easily added to supply new functionality.  You don't spend any time fiddling around with lots of different panels all over the place, and you always know where to look to find something.  Maybe the result is less fancy and "bespoke" for a given use-case (e.g., Debugging), but our "giding" principle is to use a simple framework that does most things well, much like the Go language itself.

* Strongly keyboard focused, inspired by Emacs -- existing Emacs users should be immediately productive.  However, other common keyboard bindings are also supported, and key bindings are entirely customizable.  If you're considering actually using it, we strongly recommend reading the [Wiki](https://github.com/goki/gide/wiki) tips to get the most out of it, and understand the key design principles (e.g., why there are no tabs for open files!).

# Current Status

As of 3/2020, it is feature complete as a Go IDE, including type-comprehension-based completion, and an integrated GUI debugger, running on top of [delve](https://github.com/go-delve/delve).  It is in daily use by the primary developers, and very stable at this point.

A 1.0 release is upcoming soon (March, 2020) pending final testing and bug fixing, after a major coding spree.

In addition to Issues shown on github, some important features to be added longer-term include:

* More coding automation, refactoring, etc.  We don't want to go too crazy here, prefering the more general-purpose and simple approach, but probably some more could be done.

* Full support for Python, including extending the [GoPi](https://github.com/goki/pi) interactive parser to handle Python, so it will have been worth writing instead of just using the native Go parser.


# Screenshots

![Screenshot](screenshot.png?raw=true "Screenshot")

![Screenshot, darker](screenshot_dark.png?raw=true "Screenshot, darker color scheme")
