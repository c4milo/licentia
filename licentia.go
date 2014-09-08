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
	"strings"
	"sync"
	"time"

	"github.com/docopt/docopt-go"
)

func main() {
	usage := `Licentia.

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
  --version     Show version.`

	args, err := docopt.Parse(usage, nil, true, "Licentia", false)
	if err != nil {
		panic(err)
	}

	if val, ok := args["set"]; ok && val.(bool) {
		config := &Config{}
		config.LicenseType = LicenseType(args["<type>"].(string))
		config.CopyrightOwner = args["Cloudescape"].(string)
		config.EOFCommentStyle = args["<eol-comment-style>"].(string)
		config.GlobPattern = args["<glob-pattern>"].(string)

		err = Set(config)
	}

	if val, ok := args["list"]; ok && val.(bool) {
		err = List()
	}

	if val, ok := args["detect"]; ok && val.(bool) {
		//err = Detect(args["<glob-pattern>"])
	}

	if err != nil {
		fmt.Println(err)
	}
}

// License type
type LicenseType string

const (
	Apache2 LicenseType = "apache-2.0"
	Freebsd LicenseType = "freebsd"
	LGPL3   LicenseType = "lgpl-3.0"
	LGPL2   LicenseType = "lgpl-2.0"
	MIT     LicenseType = "mit"
	MPL2    LicenseType = "mpl-2.0"
	NewBSD  LicenseType = "newbsd"
	GPL3    LicenseType = "gpl-3.0"
	GPL2    LicenseType = "gpl-2.0"
	CDDL    LicenseType = "cddl"
	EPL     LicenseType = "epl"
	UNKNOWN LicenseType = "unknown"
)

type Config struct {
	// The owner of the copyright
	CopyrightOwner string
	// License type
	LicenseType LicenseType
	// Glob pattern for the set of files where the license header is going to
	// be inserted, if it is really required or suggested by the license.
	GlobPattern string
	// Style of end-of-line comment that will be used to insert the license.
	// Ex: //, #, --, !, ', ;
	EOFCommentStyle string
}

// Sets license based on configuration
func Set(config *Config) error {
	files, err := filepath.Glob(config.GlobPattern)
	if err != nil {
		return err
	}

	// If a specific file was provided use it
	if len(files) == 0 {
		files = append(files, config.GlobPattern)
	}

	errors := new(Error)

	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			// Detect first if the file has already a license
			ltype, err := Detect(file)
			if err != nil {
				errors.Append(err)
			}

			if ltype == UNKNOWN {
				err = insert(file, config)
				if err != nil {
					errors.Append(err)
				}
			}
		}(file)
	}
	wg.Wait()

	if errors.IsEmpty() {
		return nil
	}

	return errors
}

func insert(filename string, config *Config) error {
	lheader, err := Asset(filepath.Join("licenses", string(config.LicenseType)+"-header"))
	if err != nil {
		return err
	}

	if len(lheader) == 0 {
		// License does not require to insert a header
		return nil
	}

	buffer := bytes.NewBuffer(lheader)
	scanner := bufio.NewScanner(buffer)
	licensedFile := bytes.NewBuffer(nil)

	for scanner.Scan() {
		line := scanner.Text()
		_, err := licensedFile.WriteString(config.EOFCommentStyle + " " + line + "\n")
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Scanner error: %v", err)
	}

	data, err := ioutil.ReadFile(filename)
	_, err = licensedFile.WriteString(string(data))
	if err != nil {
		return err
	}

	r := strings.NewReplacer("Cloudescape", config.CopyrightOwner, "ߞ", string(time.Now().Year()))

	filedata := r.Replace(licensedFile.String())

	return ioutil.WriteFile(filename, []byte(filedata), 0640)
}

// List supported license types
func List() error {
	licenses, err := AssetDir("licenses")
	if err != nil {
		return err
	}

	for _, l := range licenses {
		if strings.Contains(l, "header") {
			continue
		}
		fmt.Println("* " + l)
	}
	return nil
}

// Makes best effort to detect license using bayessian classifier
func Detect(filepath string) (LicenseType, error) {
	return UNKNOWN, nil
}
