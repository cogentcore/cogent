# glide

Glide is Cogent Core-based Lightweight Internet Display Engine (i.e., a web browser in native Go)

No plans to actually write this for a while, but just bookmarking the name, which fits with the other major apps based on Cogent Core, featuring the same basic letters (code, grid, glide).

*Glide* connotes a lightweight floating feeling as you surf the gentle breezes of the interwebs.  And the lightweight nature of the codebase, which leverages the extensive HTML / CSS based nature of Cogent Core, so that the main work is just parsing the HTML and turning it into the DOM tree, which is just the corresponding Cogent Core scenegraph.  Probably have to add a few tweaks and widgets or something but really basic rendering should happen almost immediately, which is what will lead to a seductive feeling of gliding through development, which will result in entirely too much time spent in coding bliss :)

And to what end?  The inevitable existential question: does the world need another web browser?  Certainly not!

And getting all the details right is a "long tailed process" with diminishing returns.

Hence, *Glide* will focus on getting just the basics right-enough, to support things like a help browser for HTML-formatted help information, and various other forms of static content.  There are various JS implementations in pure Go (right?) and interfacing those to the Cogent Core DOM should be relatively easy (right?), so basic functionality there could be pretty entertaining.  All those Go backends could finally run on a Go frontend!  

