# Licentia
Helps you set most popular OSI approved licenses in your open source projects. This package does not intent
to provide legal advice.

### Usage

```
Licentia.

Usage:
  licentia set <type> Cloudescape <glob-pattern> <eol-comment-style>
  licentia detect <glob-pattern>
  licentia list
  licentia -h | --help
  licentia --version

Sets a license to your source code. Supported license types are:

* apache2   * gpl3       * gpl2
* mpl2      * cddl
* mit       * epl
* newbsd    * freebsd
* lgpl3     * lgpl2

Arguments:
  type               License type to set. Ex: apache2, mpl2, mit, newbsd, lgpl3
  owner              Copyright owner. Ex: "YourCompany Inc"
  glob-pattern       Source files to set the license header. It supports globbing patterns. Ex: *.go
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