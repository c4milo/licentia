# Licentia
[![GoDoc](https://godoc.org/github.com/c4milo/licentia?status.svg)](https://godoc.org/github.com/c4milo/licentia)
[![Build Status](https://travis-ci.org/c4milo/licentia.svg?branch=master)](https://travis-ci.org/c4milo/licentia)

Helps you manage the open source licenses of your projects quickly and effortlessly.

### Features
Licentia allows you to:

* Easily manage your opensource licenses across several files
* Update the year of your copyright notice across several files
* Change the license of a subset of files by using glob patterns

### Installation
`go get github.com/c4milo/licentia`

### Usage

```
Licentia.

Usage:
  licentia set <type> <owner> <files> <eol-comment-style>
  licentia unset <type> <owner> <files> <eol-comment-style>
  licentia detect <files>
  licentia dump <type> <owner>
  licentia list
  licentia -h | --help
  licentia --version

Supported license types:

* apache2   * gpl3       * gpl2
* mpl2      * cddl       * unlicense
* mit       * epl
* newbsd    * freebsd
* lgpl3     * lgpl2

Actions:
  set                Sets a license header to the specified files
  unset              Removes license header from the specified files
  detect             Detects license type for the specified files
  dump               Dumps to stdout a given license using the specified owner and the current year
  list               List supported licenses

Arguments:
  type               License type to set. Ex: apache2, mpl2, mit, newbsd, lgpl3
  owner              Copyright owner. Ex: "YourCompany Inc"
  files              Source files to set the license header. It supports globbing patterns as well as specifying individual files. Ex: *.go, myfile.go, **/*.go
  eol-comment-style  End-of-line comment style. Ex: #, ;, //, --, ', etc.

Options:
  -h --help     Show this screen.
  --version     Show version.
```

### Licenses supported
* Apache License 2.0
* Mozilla Public License 2.0
* MIT License
* GNU General Public License (GPL)
* GNU Library or "Lesser" General Public License (LGPL)
* BSD 2-Clause "Simplified" or "FreeBSD" license
* BSD 3-Clause "New" or "Revised" license
* Common Development and Distribution License
* Eclipse Public Licenses
* Unlicense

