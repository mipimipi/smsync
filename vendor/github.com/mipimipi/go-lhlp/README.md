# go-lhlp

"Go's little helper": Practical and handy functions that are useful in many Go projects, but which are not part of the standards Go libraries, such as

* `Contains` checks if an array contains a certain element

* `CopyFile` copies a file

* `FileExists` checks if a file exists

* `FindFiles` traverses the directory tree to find files that fulfill certain filter criteria (similar to the [Unix find command](https://linux.die.net/man/1/find "man pages for find")). To check the filter criteria, the user must pass a filter function to `FindFiles`.

and more.

[![GoDoc](https://godoc.org/github.com/mipimipi/go-lhlp?status.svg)](https://godoc.org/github.com/mipimipi/go-lhlp)

## Installation

To install `go-lhlp` use the `go` tool and simply execute

    go get github.com/mipimipi/go-lhlp