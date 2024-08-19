# Cogent Author

Author processes Markdown files into different output formats (PDF, HTML, DOCX, EPUB) using the amazing [pandoc](https://pandoc.org/) tool.  It is based originally on the python-based [ebook](https://github.com/bmc/ebook) package by Brian Clapper.  This Go version is more stable relative to the ever-changing dependency hell that comes with python, and it is written using the [cosh](https://github.com/cogentcore/core/tree/main/shell) shell system that we find to be much more obvious and simple to understand and modify.

It supports the following output formats:
* [book](#book) generates an entire book from separate `chapter-*.md` files.
* [article](#article) generates a scientific-style article.

In addition to the markdown content, `pandoc` recognizes `yaml` metadata at the start of the file, and this is a critical element of the processing to specify various options etc.  For `article`, it can be specified at the start of the file  surrounded by `---` block delimiters, and for `book` it is a separate file.  Pandoc requires that all references be included in this metadata header, so a big part of what `author` does is assemble all of that for you.

# References

References to other literature are specified using the following syntax:

```markdown
[@Keil81; @SlomanFernbach18]
```

These are citation keys to a BibTeX `references.bib` file that is automatically generated from a potentially much larger collection of references that must be in a file named `allrefs.bib` (which can e.g., be a link to a shared file somewhere).

It is in general a good idea to commit the references.bib file along with the rest of the source text into your github repository, so it can be built without requiring the `allrefs.bib` source file.

The style of the references is determined by the `csl` property in the `metadata.yaml` file, and defaults to the pandoc default if not otherwise specified.  You can find all manner of styles here: https://github.com/citation-style-language/styles

# Book

You must use specific file names to indicate the functionality and ordering of the content within the book, as follows, with [] indicating optional files:

* `metadata.yaml`: pandoc metadata with various important options.
* `frontmatter.md`: with copyright, dedication, foreward, preface, prologue sections.
* `chapter-*.md` chapters, using 01 etc numbering to put in order.
* `endmatter.md`: includes epilogue, acknowledgements, author
* `[appendix-*.md]` appendicies, using a, b, c, etc labeling.
* `[glossary.md]` optional glossary that generates links in text.
* `allrefs.bib`  source of all references in BibTex format, to use in resolving citations.

# Troubleshooting

* The custom `latex.template` can be a problem when pandoc is updated beyond its compatibility.  The `latex.template.diff` shows the diff relative to the `default.latex` template used, from https://github.com/jgm/pandoc-templates for version 3.3.  It is a good idea to just update to the latest default template and re-apply the diff (using patch or manually).




