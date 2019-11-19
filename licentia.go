// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/ryanuber/go-license"

	_ "github.com/c4milo/licentia/statik"
	"github.com/rakyll/statik/fs"
)

//go:generate go get github.com/rakyll/statik
//go:generate statik -f -src licenses

var Version string
var statikFS http.FileSystem

func main() {
	usage := `Licentia.

Usage:
  licentia set [--replace] <type> <owner> <eol-comment-style> <files>...
  licentia unset <type> <owner> <eol-comment-style> <files>...
  licentia detect <files>...
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
  files              Source files to set the license header. It supports globbing patterns, individual files or folders. Ex: *.go
  eol-comment-style  End-of-line comment style. Ex: #, ;, //, --, ', etc.

Options:
  -h --help     Show this screen.
  --version     Show version.
  --replace     Try to replace the old license with the new one in "set".
`

	args, err := docopt.Parse(usage, nil, true, Version, false)
	if err != nil {
		panic(err)
	}

	if statikFS, err = fs.New(); err != nil {
		panic(err)
	}

	var files []string
	if val, ok := args["set"]; ok && val.(bool) {
		if files, err = globFiles(args["<files>"].([]string)); err == nil {
			config := &Config{
				LicenseType:     LicenseType(args["<type>"].(string)),
				CopyrightOwner:  args["<owner>"].(string),
				EOLCommentStyle: args["<eol-comment-style>"].(string),
				Files:           files,
				Replace:         args["--replace"].(bool),
			}
			err = Set(config)
		}
	}

	if val, ok := args["unset"]; ok && val.(bool) {
		if files, err = globFiles(args["<files>"].([]string)); err == nil {
			config := &Config{
				LicenseType:     LicenseType(args["<type>"].(string)),
				CopyrightOwner:  args["<owner>"].(string),
				EOLCommentStyle: args["<eol-comment-style>"].(string),
				Files:           files,
			}
			err = Unset(config)
		}
	}

	if val, ok := args["list"]; ok && val.(bool) {
		var types []string
		types, err = List()

		fmt.Println("Supported licenses: ")
		for _, t := range types {
			fmt.Println("* " + t)
		}
	}

	if val, ok := args["dump"]; ok && val.(bool) {
		var license string
		license, err = Dump(LicenseType(args["<type>"].(string)), args["<owner>"].(string))
		fmt.Println(license)
	}

	if val, ok := args["detect"]; ok && val.(bool) {
		if files, err = globFiles(args["<files>"].([]string)); err == nil {
			config := &Config{Files: files}
			var types []fileLicense
			types, err = Detect(config)
			for _, elt := range types {
				fmt.Printf("%s:\t%s\n", elt.file, elt.license)
			}
		}
	}

	if err != nil {
		fmt.Println(err)
	}
}

// License type
type LicenseType string

const (
	Apache2   LicenseType = "apache2"
	Freebsd   LicenseType = "freebsd"
	LGPL3     LicenseType = "lgpl3"
	LGPL2     LicenseType = "lgpl2"
	MIT       LicenseType = "mit"
	MPL2      LicenseType = "mpl2"
	NewBSD    LicenseType = "newbsd"
	GPL3      LicenseType = "gpl3"
	GPL2      LicenseType = "gpl2"
	CDDL      LicenseType = "cddl"
	EPL       LicenseType = "epl"
	UNLICENSE LicenseType = "unlicense"
	UNKNOWN   LicenseType = "unknown"
)

type Config struct {
	// The owner of the copyright
	CopyrightOwner string
	// License type
	LicenseType LicenseType
	// Invidiviual file or folder as well as glob patterns are recognized
	Files []string
	// Style of end-of-line comment that will be used to insert the license.
	// Ex: //, #, --, !, ', ;
	EOLCommentStyle string
	Replace         bool
}

func globFiles(args []string) ([]string, error) {
	files := make([]string, 0, len(args)+1)
	for _, arg := range args {
		f, err := filepath.Glob(arg)
		if err != nil {
			return files, err
		}
		files = append(files, f...)
	}
	return files, nil
}

// Dumps license to stdout setting the owner and year in the copyright notice
func Dump(ltype LicenseType, owner string) (string, error) {
	replacer := strings.NewReplacer(
		"@@owner@@", owner,
		"@@year@@", strconv.Itoa(time.Now().Year()),
	)
	data, err := Asset(filepath.Join("licenses", string(ltype)))
	if err != nil {
		return "", err
	}

	lcopyright, _ := Asset(filepath.Join("licenses", string(ltype)+".copyright"))
	data = append(lcopyright, data...)

	return replacer.Replace(string(data)), nil
}

// Sets license
func Set(config *Config) error {
	errors := new(Error)

	var wg sync.WaitGroup
	replacer := strings.NewReplacer(
		"@@owner@@", config.CopyrightOwner,
		"@@year@@", strconv.Itoa(time.Now().Year()),
	)

	removeConfig := *config

	for _, file := range config.Files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			if config.Replace {
				// Detect old license and remove before adding another one.
				old, err := detectLicense(file)
				//fmt.Fprintf(os.Stderr, "OLD:%s err=%v\n", old, err)
				if err == nil && old != UNKNOWN {
					removeConfig.LicenseType = old
					removeConfig.Files = []string{file}
					if err = removeLicense(file, &removeConfig); err != nil {
						errors.Append(fmt.Errorf("remove %q license from %q: %v", old, file, err))
					}
				}
			}

			if err := insertLicense(file, replacer, config); err != nil {
				errors.Append(err)
			}
		}(file)
	}
	wg.Wait()

	if errors.IsEmpty() {
		return nil
	}

	return errors
}

// Removes license
func Unset(config *Config) error {
	errors := new(Error)

	var wg sync.WaitGroup
	for _, file := range config.Files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			if err := removeLicense(file, config); err != nil {
				errors.Append(err)
			}
		}(file)
	}
	wg.Wait()

	if errors.IsEmpty() {
		return nil
	}

	return errors
}

// Removes license header from file represented by filename
func removeLicense(filename string, config *Config) error {
	lbuffer := bytes.NewBuffer(nil)
	lheader, err := Asset(filepath.Join("licenses", string(config.LicenseType)+".header"))
	if err != nil {
		// This license does require a license header in the source file.
		// Do not remove anything
		return nil
	}

	err = prependEOLComment(lbuffer, config.EOLCommentStyle, lheader)
	if err != nil {
		return err
	}

	license := lbuffer.String()

	licensedFile, err := ioutil.ReadFile(filename)
	buf := bytes.NewBuffer(licensedFile)
	unlicensedFile := bytes.NewBuffer(nil)

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		if bytes.HasPrefix(scanner.Bytes(), []byte(config.EOLCommentStyle+" Copyright")) {
			continue
		}
		line := scanner.Text()
		_, err := unlicensedFile.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Scanner error: %v", err)
	}

	unlicensedData := unlicensedFile.String()

	//fmt.Fprintf(os.Stderr, "unl=%q\nlic=%q\n", unlicensedData, license)
	unlicensedData = strings.Replace(unlicensedData, license, "", -1)

	// Strip multiple empty lines from before package.
	if i := strings.Index(unlicensedData, "\npackage"); i >= 3 {
		unlicensedData = strings.TrimRight(unlicensedData[:i], "\n") + unlicensedData[i:]
	}

	mode := os.FileMode(0640)
	fi, err := os.Stat(filename)
	if err == nil {
		mode = fi.Mode()
	}
	return ioutil.WriteFile(filename, []byte(unlicensedData), mode)
}

// Inserts license header to file represented by filename
func insertLicense(filename string, replacer *strings.Replacer, config *Config) error {
	licensedFile := bytes.NewBuffer(nil)

	lcopyright, err := Asset(filepath.Join("licenses", string(config.LicenseType)+".copyright"))

	cr := false
	if err == nil {
		err = prependEOLComment(licensedFile, config.EOLCommentStyle,
			[]byte(replacer.Replace(string(lcopyright))))
		if err != nil {
			return err
		}
		cr = true
	}

	lheader, err := Asset(filepath.Join("licenses", string(config.LicenseType)+".header"))
	if err == nil {
		plus := ""
		if cr {
			plus = "\n"
		}
		err := prependEOLComment(licensedFile, config.EOLCommentStyle,
			[]byte(replacer.Replace(plus+string(lheader))))
		if err != nil {
			return err
		}
	}
	// Extra newline for separating license code from package docs.
	licensedFile.WriteByte('\n')

	// Only use the replacer for the license, not the whole file.

	fh, err := os.Open(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(licensedFile, fh)
	fh.Close()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, licensedFile.Bytes(), 0640)
}

// Prepends end-of-line comment to newdata and returns it in licensedFile
func prependEOLComment(licensedFile *bytes.Buffer, eol string, newdata []byte) error {
	if len(newdata) == 0 {
		return nil
	}

	buffer := bytes.NewBuffer(newdata)
	scanner := bufio.NewScanner(buffer)

	for scanner.Scan() {
		line := scanner.Text()
		eol := strings.TrimSpace(eol + " " + line)
		_, err := licensedFile.WriteString(eol)
		if err != nil {
			return err
		}
		if err = licensedFile.WriteByte('\n'); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Scanner error: %v", err)
	}
	return nil
}

// List supported license types
func List() ([]string, error) {
	licenses, err := AssetDir("licenses")
	if err != nil {
		return nil, err
	}

	types := make([]string, 0, len(licenses))
	for _, l := range licenses {
		if strings.HasSuffix(l, "header") || strings.HasSuffix(l, "copyright") {
			continue
		}
		types = append(types, l)
	}
	return types, nil
}

type fileLicense struct {
	file    string
	license LicenseType
}

// Detect the licenses.
func Detect(config *Config) ([]fileLicense, error) {
	var typesMtx sync.Mutex
	types := make([]fileLicense, 0, len(config.Files))
	errors := new(Error)

	var wg sync.WaitGroup
	for _, file := range config.Files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			lic, err := detectLicense(file)
			typesMtx.Lock()
			types = append(types, fileLicense{file: file, license: lic})
			typesMtx.Unlock()
			if err != nil {
				errors.Append(err)
			}
		}(file)
	}
	wg.Wait()

	if errors.IsEmpty() {
		return types, nil
	}

	return types, errors
}

func detectLicense(filepath string) (LicenseType, error) {
	fh, err := os.Open(filepath)
	if err != nil {
		return UNKNOWN, err
	}
	defer fh.Close()

	var buf bytes.Buffer
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		if bytes.HasPrefix(scanner.Bytes(), []byte("package ")) {
			break
		}
		line := bytes.TrimSuffix(bytes.TrimPrefix(bytes.TrimPrefix(scanner.Bytes(),
			[]byte("//")), []byte("/*")), []byte("*/"))
		if len(line) > 0 && (line[0] == '+' || bytes.HasPrefix(bytes.TrimSpace(line), []byte("Copyright"))) {
			continue
		}
		buf.Write(bytes.TrimSpace(line))
		buf.WriteByte('\n')
	}
	//fmt.Fprintf(os.Stderr, "DETECT %q\n", strings.TrimSpace(buf.String()))
	l := license.New("", strings.TrimSpace(buf.String()))
	l.File = filepath
	if err = l.GuessType(); err != nil {
		if err == license.ErrUnrecognizedLicense {
			return UNKNOWN, scanner.Err()
		}
		return UNKNOWN, err
	}

	err = scanner.Err()
	switch l.Type {
	case license.LicenseMIT:
		return MIT, err
	case license.LicenseNewBSD:
		return NewBSD, err
	case license.LicenseFreeBSD:
		return Freebsd, err
	case license.LicenseApache20:
		return Apache2, err
	case license.LicenseMPL20:
		return MPL2, err
	case license.LicenseGPL20:
		return GPL2, err
	case license.LicenseGPL30:
		return GPL3, err
	case license.LicenseLGPL21:
		return LGPL2, err
	case license.LicenseLGPL30:
		return LGPL2, err
	case license.LicenseCDDL10:
		return CDDL, err
	case license.LicenseEPL10:
		return EPL, err
	}
	return UNKNOWN, err
}

func assetPath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.TrimPrefix(path, "/licenses")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
func Asset(path string) ([]byte, error) {
	fh, err := statikFS.Open(assetPath(path))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return ioutil.ReadAll(fh)
}

func AssetDir(path string) ([]string, error) {
	dh, err := statikFS.Open(assetPath(path))
	if err != nil {
		return nil, err
	}
	defer dh.Close()
	fis, err := dh.Readdir(-1)
	names := make([]string, len(fis))
	for i, fi := range fis {
		names[i] = fi.Name()
	}
	return names, err
}
