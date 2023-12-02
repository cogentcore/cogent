![alt tag](logo/gide_icon.png)

**Gide** is a flexible IDE (integrated development environment) framework in pure Go, using the [GoGi](https://github.com/goki/gi) GUI (for which it serves as a continuous testing platform :) and the [GoPi](https://github.com/goki/pi) interactive parser for syntax highlighting and more advanced IDE code processing.

See the [Wiki](https://goki.dev/gide/v2/wiki) for more docs,   [Install](https://goki.dev/gide/v2/wiki/Install) instructions (`go install goki.dev/gide/v2/cmd/gide@latest` should work if GoGi system libraries are in place), and [Google Groups goki-gi](https://groups.google.com/forum/#!forum/goki-gi) emailing list.

[![Go Report Card](https://goreportcard.com/badge/goki.dev/gide/v2)](https://goreportcard.com/report/goki.dev/gide/v2)
[![GoDoc](https://godoc.org/goki.dev/gide/v2?status.svg)](https://godoc.org/goki.dev/gide/v2)
[![Travis](https://travis-ci.com/goki/gide.svg?branch=master)](https://travis-ci.com/goki/gide)

There are many existing, excellent choices for text editors and IDEs, but *Gide* is possibly the best pure *Go* option available.  The Go language represents a special combination of simplicity, elegance, and power, and is a joy to program in, and is currently the main language fully-supported by Gide.  Our ambition is to capture some of those Go attributes in an IDE.

Some of the main features of *Gide* include:

* Designed to function as both a general-purpose text editor *and* an IDE.  It comes configured with command support for LaTeX, Markdown, and Makefiles, in addition to Go,a and the command system is fully extensible to support any command-line tools.

* Provides a tree-based file browser on the left, with builtin support for version control (git, svn, etc) and standard file management functionality through drag-and-drop, etc.  You can look at git logs, diffs, etc through this interface.

* Command actions show output on a tabbed output display on the right, along with other special interfaces such as Find / Replace, Symbols, Debugger, etc.  Thus, the overall design is extensible and new interfaces can be easily added to supply new functionality.  You don't spend any time fiddling around with lots of different panels all over the place, and you always know where to look to find something.  Maybe the result is less fancy and "bespoke" for a given use-case (e.g., Debugging), but our "giding" principle is to use a simple framework that does most things well, much like the Go language itself.

* Strongly keyboard focused, inspired by Emacs -- existing Emacs users should be immediately productive.  However, other common keyboard bindings are also supported, and key bindings are entirely customizable.  If you're considering actually using it, we strongly recommend reading the [Wiki](https://goki.dev/gide/v2/wiki) tips to get the most out of it, and understand the key design principles (e.g., why there are no tabs for open files!).

# Install

* Wiki instructions: [Install](https://goki.dev/gide/v2/wiki/Install) -- for building directly from source.

* See Releases on this github page for pre-built OS-specific app packages that install the compiled binaries.

* See `install` directory for OS-specific Makefiles to install apps and build packages.

# Current Status

As of April 2020, it is feature complete as a Go IDE, including type-comprehension-based completion, and an integrated GUI debugger, running on top of [delve](https://github.com/go-delve/delve).  It is in daily use by the primary developers, and very stable at this point, with the initial 1.0 release now available.

In addition to Issues shown on github, some important features to be added longer-term include:

* More coding automation, refactoring, etc.  We don't want to go too crazy here, preferring the more general-purpose and simple approach, but probably some more could be done.

* Full support for Python, including extending the [GoPi](https://github.com/goki/pi) interactive parser to handle Python, so it will have been worth writing instead of just using the native Go parser.

# Screenshots

![Screenshot](screenshot.png?raw=true "Screenshot")

![Screenshot, darker](screenshot_dark.png?raw=true "Screenshot, dark mode")

# TODO

* focus in topappbar returning to TAB not going to where command goes.

* chooser needs special support for URI with icons
* more helpers for URI api
* filter function for chooser for URI case
* rename Supported -> Enum or something?

* add icons for symbols and get the merge thing to work
* symbolsview: textview still not jumping to line correctly

* Editable Chooser doesn't work with shift-tab -- requires focus prev fix to work properly!

* Find not selecting first item (sometimes?)

* dialog closing causes old window to show up -- need to update render images
* textview context menu

* fileview global priority key shortcuts and fileview menu
* editor rendering overflow
* lineno horiz scrolling issues -- this is in V1 too!
* filetree --get rid of empty upper level or not?
* sliceview / tableview should activate select and focus on selectidx item in readonly chooser mode -- select is working, but focus is not -- cannot move selection via keyboard

# DONE:

* goki.dev/gi/v2/gi.(*RenderWin).EventLoop.func1()
	/Users/oreilly/goki.dev/gi/v2/gi/renderwin.go:598 +0x158
panic({0x102aa0600?, 0x1038807e0?})
	/opt/homebrew/Cellar/go/1.21.4/libexec/src/runtime/panic.go:914 +0x218
goki.dev/gi/v2/texteditor.(*Editor).PixelToCursor(0x140006a1000, {0x100843ab8?, 0x1400e843ab8?})
	/Users/oreilly/goki.dev/gi/v2/texteditor/render.go:676 +0x114
goki.dev/gi/v2/texteditor.(*Editor).HandleEditorEvents.(*Editor).HandleEditorLinkCursor.func1({0x102c4fb68, 0x14022413ce0})
	/Users/oreilly/goki.dev/gi/v2/texteditor/events.go:652 +0x60
goki.dev/gi/v2/texteditor.(*Editor).HandleEditorEvents.(*Editor).HandleEditorLinkCursor.(*WidgetBase).On.func2({0x102c4fb68, 0x14022413ce0})
	/Users/oreilly/goki.dev/gi/v2/gi/events.go:35 +0x7c
goki.dev/goosi/events.(*Listeners).Call(0x140006a1700, {0x102c4fb68, 0x14022413ce0})
	/Users/oreilly/goki.dev/goosi/events/listeners.go:58 +0xa4
goki.dev/gi/v2/gi.(*WidgetBase).HandleEvent(0x140006a1000, {0x102c4fb68, 0x14022413ce0})
	/Users/oreilly/goki.dev/gi/v2/gi/events.go:196 +0x1a0
goki.dev/gi/v2/gi.(*EventMgr).HandlePosEvent(0x14000234140, {0x102c4fb68, 0x14022413ce0})
	/Users/oreilly/goki.dev/gi/v2/gi/eventmgr.go:315 +0x43c
goki.dev/gi/v2/gi.(*EventMgr).HandleEvent(0x140002e1260?, {0x102c4fb68, 0x14022413ce0})
	/Users/oreilly/goki.dev/gi/v2/gi/eventmgr.go:198 +0x58
goki.dev/gi/v2/gi.(*Stage).MainHandleEvent(0x1400074b200, {0x102c4fb68?, 0x14022413ce0})
	/Users/oreilly/goki.dev/gi/v2/gi/mainstage.go:309 +0x144
goki.dev/gi/v2/gi.(*StageMgr).MainHandleEvent(0x14000440038, {0x102c4fb68, 0x14022413ce0})
	/Users/oreilly/goki.dev/gi/v2/gi/mainstage.go:317 +0x78
goki.dev/gi/v2/gi.(*RenderWin).HandleEvent(0x14000440000, {0x102c4fb68, 0x14022413ce0})
	/Users/oreilly/goki.dev/gi/v2/gi/renderwin.go:652 +0x128
goki.dev/gi/v2/gi.(*RenderWin).EventLoop(0x14000440000)
	/Users/oreilly/goki.dev/gi/v2/gi/renderwin.go:618 +0x50
created by goki.dev/gi/v2/gi.(*RenderWin).GoStartEventLoop in goroutine 50
	/Users/oreilly/goki.dev/gi/v2/gi/renderwin.go:544 +0xa4

