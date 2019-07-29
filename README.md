Meadow
======

*Meadow* is a simple program for Markdown Editing Actively Displayed
On the Web-browser.  The concept is simple: edit a markdown file in a
text editor of your choice, have *Meadow* watch that file, and have
the markdown contents transformed to HTML and displayed in the browser
whenever you save the file.  Given how modern browsers can print to
file as a PDF, along with the ability to add custom CSS, you can
create good looking PDF documents without having to use special word
processing software.

Install
-------

Grab a copy of *Meadow* from the [Releases](https://github.com/0xABAD/meadow/releases)
tab for your specific system.  Alternatively, if you have golang 1.12+ installed then
you can run:

```
$ go get github.com/0xABAD/meadow
$ go install github.com/0xABAD/meadow
```

If you do get the source then you can navigate into the **sample**
directory which contains simple markdown file to take *Meadow* out for
a spin.

Running
-------

Running *Meadow* is simple, just call:

```$ meadow your_markdown_file.md```

By default *Meadow* will apply a default **Github CSS** that is
applied to the HTML output; however, you can pass `-css
your_CSS_file.css` to *Meadow* and that CSS will be applied on top of
the existing default CSS.  This custom CSS file is ordered to come
after the default so you can override any of the existing styles.
Further, the custom CSS file is also watched, along with your markdown
file, so can make interactive changes and see them hot-reloaded into
the browser.

License
-------

Zlib


