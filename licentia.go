// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docopt/docopt-go"
)

func main() {
	usage := `Licentia.

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
* mpl2      * cddl
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
  --version     Show version.`

	args, err := docopt.Parse(usage, nil, true, "Licentia", false)
	if err != nil {
		panic(err)
	}

	if val, ok := args["set"]; ok && val.(bool) {
		config := &Config{
			LicenseType:     LicenseType(args["<type>"].(string)),
			CopyrightOwner:  args["<owner>"].(string),
			EOLCommentStyle: args["<eol-comment-style>"].(string),
			Files:           args["<files>"].(string),
		}
		err = Set(config)
	}

	if val, ok := args["unset"]; ok && val.(bool) {
		config := &Config{
			LicenseType:     LicenseType(args["<type>"].(string)),
			CopyrightOwner:  args["<owner>"].(string),
			EOLCommentStyle: args["<eol-comment-style>"].(string),
			Files:           args["<files>"].(string),
		}
		err = Unset(config)
	}

	if val, ok := args["list"]; ok && val.(bool) {
		err = List()
	}

	if val, ok := args["dump"]; ok && val.(bool) {
		err = Dump(LicenseType(args["<type>"].(string)), args["<owner>"].(string))
	}

	if val, ok := args["detect"]; ok && val.(bool) {
		//err = Detect(args["<files>"].(string))
	}

	if err != nil {
		fmt.Println(err)
	}
}

// License type
type LicenseType string

const (
	Apache2 LicenseType = "apache2"
	Freebsd LicenseType = "freebsd"
	LGPL3   LicenseType = "lgpl3"
	LGPL2   LicenseType = "lgpl2"
	MIT     LicenseType = "mit"
	MPL2    LicenseType = "mpl2"
	NewBSD  LicenseType = "newbsd"
	GPL3    LicenseType = "gpl3"
	GPL2    LicenseType = "gpl2"
	CDDL    LicenseType = "cddl"
	EPL     LicenseType = "epl"
	UNKNOWN LicenseType = "unknown"
)

type Config struct {
	// The owner of the copyright
	CopyrightOwner string
	// License type
	LicenseType LicenseType
	// Invidiviual file or folder as well as glob patterns are recognized
	Files string
	// Style of end-of-line comment that will be used to insert the license.
	// Ex: //, #, --, !, ', ;
	EOLCommentStyle string
}

// Dumps license to stdout setting the owner and year in the copyright notice
func Dump(ltype LicenseType, owner string) error {
	replacer := strings.NewReplacer("@@owner@@", owner, "@@year@@", strconv.Itoa(time.Now().Year()))
	data, err := Asset(filepath.Join("licenses", string(ltype)))
	if err != nil {
		return err
	}

	lcopyright, err := Asset(filepath.Join("licenses", string(ltype)+".copyright"))
	data = append(lcopyright, data...)

	license := replacer.Replace(string(data))
	fmt.Println(license)

	return nil
}

// Sets license
func Set(config *Config) error {
	files, err := filepath.Glob(config.Files)
	if err != nil {
		return err
	}

	// If a specific file was provided use it
	if len(files) == 0 {
		// if folder
		files = append(files, config.Files)
	}

	errors := new(Error)

	var wg sync.WaitGroup
	replacer := strings.NewReplacer("@@owner@@", config.CopyrightOwner, "@@year@@", strconv.Itoa(time.Now().Year()))

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			err = insertLicense(file, replacer, config)
			if err != nil {
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
	files, err := filepath.Glob(config.Files)
	if err != nil {
		return err
	}

	// If a specific file was provided use it
	if len(files) == 0 {
		// if folder
		files = append(files, config.Files)
	}

	errors := new(Error)

	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			err = removeLicense(file, config)
			if err != nil {
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

	err = prependEOLComment(lbuffer, config, lheader)
	if err != nil {
		return err
	}

	license := lbuffer.String()

	licensedFile, err := ioutil.ReadFile(filename)
	buf := bytes.NewBuffer(licensedFile)
	unlincesedFile := bytes.NewBuffer(nil)

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "// Copyright") {
			continue
		}
		_, err := unlincesedFile.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Scanner error: %v", err)
	}

	unlicensedData := unlincesedFile.String()

	unlicensedData = strings.Replace(unlicensedData, license, "", -1)

	return ioutil.WriteFile(filename, []byte(unlicensedData), 0640)
}

// Inserts license header to file represented by filename
func insertLicense(filename string, replacer *strings.Replacer, config *Config) error {
	licensedFile := bytes.NewBuffer(nil)

	lcopyright, err := Asset(filepath.Join("licenses", string(config.LicenseType)+".copyright"))

	if err == nil {
		err = prependEOLComment(licensedFile, config, lcopyright)
		if err != nil {
			return err
		}
	}

	lheader, err := Asset(filepath.Join("licenses", string(config.LicenseType)+".header"))
	if err == nil {
		err := prependEOLComment(licensedFile, config, lheader)
		if err != nil {
			return err
		}
	}

	data, err := ioutil.ReadFile(filename)
	_, err = licensedFile.WriteString(string(data))
	if err != nil {
		return err
	}

	filedata := replacer.Replace(licensedFile.String())

	return ioutil.WriteFile(filename, []byte(filedata), 0640)
}

// Prepends end-of-line comment to newdata and returns it in licensedFile
func prependEOLComment(licensedFile *bytes.Buffer, config *Config, newdata []byte) error {
	if len(newdata) == 0 {
		return nil
	}

	buffer := bytes.NewBuffer(newdata)
	scanner := bufio.NewScanner(buffer)

	for scanner.Scan() {
		line := scanner.Text()
		eol := strings.TrimSpace(config.EOLCommentStyle + " " + line)
		_, err := licensedFile.WriteString(eol + "\n")
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Scanner error: %v", err)
	}
	return nil
}

// List supported license types
func List() error {
	licenses, err := AssetDir("licenses")
	if err != nil {
		return err
	}

	for _, l := range licenses {
		if strings.HasSuffix(l, "header") || strings.HasSuffix(l, "copyright") {
			continue
		}
		fmt.Println("* " + l)
	}
	return nil
}

// TODO(c4milo): Use go-license
func Detect(filepath string) (LicenseType, error) {
	return UNKNOWN, nil
}
