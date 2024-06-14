**Cogent Code** is a flexible IDE (integrated development environment) framework in pure Go, using the [Cogent Core](https://cogentcore.org/core) GUI (for which it serves as a continuous testing platform) and the [parse](https://pkg.go.dev/cogentcore.org/core/parse) interactive parser for syntax highlighting and more advanced IDE code processing.

There are many existing, excellent choices for text editors and IDEs, but *Code* is possibly the best pure *Go* option available.  The Go language represents a special combination of simplicity, elegance, and power, and is a joy to program in, and is currently the main language fully supported by Code.  Our ambition is to capture some of those Go attributes in an IDE.

Some of the main features of *Code* include:

* Designed to function as both a general-purpose text editor *and* an IDE.  It comes configured with command support for LaTeX, Markdown, and Makefiles, in addition to Go,a and the command system is fully extensible to support any command-line tools.

* Provides a tree-based file browser on the left, with builtin support for version control (git, svn, etc) and standard file management functionality through drag-and-drop, etc.  You can look at git logs, diffs, etc through this interface.

* Command actions show output on a tabbed output display on the right, along with other special interfaces such as Find / Replace, Symbols, Debugger, etc.  Thus, the overall design is extensible and new interfaces can be easily added to supply new functionality.  You don't spend any time fiddling around with lots of different panels all over the place, and you always know where to look to find something.  Maybe the result is less fancy and "bespoke" for a given use-case (e.g., Debugging), but our guiding principle is to use a simple framework that does most things well, much like the Go language itself.

* Strongly keyboard focused, inspired by Emacs -- existing Emacs users should be immediately productive.  However, other common keyboard bindings are also supported, and key bindings are entirely customizable.  If you're considering actually using it, we strongly recommend reading the [Wiki](https://cogentcore.org/cogent/code/wiki) tips to get the most out of it, and understand the key design principles (e.g., why there are no tabs for open files!).

# Install

* Wiki instructions: [Install](https://cogentcore.org/cogent/code/wiki/Install) -- for building directly from source.

* See Releases on this github page for pre-built OS-specific app packages that install the compiled binaries.

* See `install` directory for OS-specific Makefiles to install apps and build packages.

# Current Status

As of April 2020, it is feature complete as a Go IDE, including type-comprehension-based completion, and an integrated GUI debugger, running on top of [delve](https://github.com/go-delve/delve).  It is in daily use by the primary developers, and very stable at this point, with the initial 1.0 release now available.

In addition to Issues shown on github, some important features to be added longer-term include:

* More coding automation, refactoring, etc.  We don't want to go too crazy here, preferring the more general-purpose and simple approach, but probably some more could be done.

* Full support for Python, including extending the [parse](https://pkg.go.dev/cogentcore.org/core/parse) interactive parser to handle Python, so it will have been worth writing instead of just using the native Go parser.

# Screenshots

![Screenshot](screenshot.png?raw=true "Screenshot")

![Screenshot, darker](screenshot_dark.png?raw=true "Screenshot, dark mode")

# TODO

* symbolspanel icons not updating immediately -- fix from filenodedid not fix here

* line number too conservative in not rendering bottom
* don't render top text, line number if out of range
* always display cursor when typing!
* drag-n-drop table

* color highlighting for diff output in commandshell!
* outbuf use textview markup in addition to link formatting and other formatting.  tried but failed

* more helpers for URI api
* filter function for chooser for URI case

* Editable Chooser doesn't work with shift-tab -- requires focus prev fix to work properly!

* Find not selecting first item (sometimes?)

* filepicker global priority key shortcuts
* editor rendering overflow
* filetree --get rid of empty upper level or not?
* list / table should activate select and focus on selectidx item in readonly chooser mode -- select is working, but focus is not -- cannot move selection via keyboard


# DONE:




