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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/ryanuber/go-license"
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
		config := &Config{
			Files: args["<files>"].(string),
		}
		var types []fileLicense
		types, err = Detect(config)
		for _, elt := range types {
			fmt.Printf("%s:\t%s\n", elt.file, elt.license)
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
	Files string
	// Style of end-of-line comment that will be used to insert the license.
	// Ex: //, #, --, !, ', ;
	EOLCommentStyle string
}

// Dumps license to stdout setting the owner and year in the copyright notice
func Dump(ltype LicenseType, owner string) (string, error) {
	replacer := strings.NewReplacer("The licentia Authors", owner, "2014", strconv.Itoa(time.Now().Year()))
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
	files, err := filepath.Glob(config.Files)
	if err != nil {
		return err
	}

	errors := new(Error)

	var wg sync.WaitGroup
	replacer := strings.NewReplacer("The licentia Authors", config.CopyrightOwner, "2014", strconv.Itoa(time.Now().Year()))

	removeConfig := *config

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			// Detect old license and remove before adding another one.
			old, err := detectLicense(file)
			if err == nil && old != UNKNOWN {
				removeConfig.Files = file
				if err = removeLicense(file, &removeConfig); err != nil {
					errors.Append(fmt.Errorf("remove %q license from %q: %v", old, file, err))
				}
			}

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
	unlicensedFile := bytes.NewBuffer(nil)

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		if bytes.HasPrefix(scanner.Bytes(), []byte("// Copyright")) {
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

	unlicensedData = strings.Replace(unlicensedData, license, "", -1)

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

	fh, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fh.Close()
	if _, err = io.Copy(licensedFile, fh); err != nil {
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
	// Extra newline for separating license code from package docs.
	licensedFile.WriteByte('\n')

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
	files, err := filepath.Glob(config.Files)
	if err != nil {
		return nil, err
	}

	var typesMtx sync.Mutex
	types := make([]fileLicense, 0, len(files))
	errors := new(Error)

	var wg sync.WaitGroup
	for _, file := range files {
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
		buf.Write(bytes.TrimSpace(line))
		buf.WriteByte('\n')
	}
	l := license.New("", buf.String())
	l.File = filepath
	if err = l.GuessType(); err != nil {
		if err.Error() == license.ErrUnrecognizedLicense {
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
