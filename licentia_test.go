// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

var mpl2 = `// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

`

func TestSetUnset(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "licentia-tests-")
	ok(t, err)

	filepath := file.Name()

	defer os.Remove(filepath)

	config := &Config{
		CopyrightOwner:  "Test",
		LicenseType:     MPL2,
		Files:           []string{filepath},
		EOLCommentStyle: "//",
	}

	err = Set(config)
	ok(t, err)

	data, err := ioutil.ReadFile(filepath)
	ok(t, err)

	equals(t, mpl2, string(data))

	err = Unset(config)
	ok(t, err)

	data, err = ioutil.ReadFile(filepath)
	ok(t, err)

	equals(t, "\n", string(data))
}

func TestList(t *testing.T) {
	types, err := List()
	ok(t, err)
	assert(t, len(types) == 12, "Number of supported licenses should be 12")
}

func TestDump(t *testing.T) {
	data, err := ioutil.ReadFile("licenses/mpl2")
	ok(t, err)

	license, err := Dump(MPL2, "Test")
	ok(t, err)

	equals(t, string(data), license)
}

func TestDetect(t *testing.T) {
	//TODO(c4milo)
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
