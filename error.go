// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package main

import "fmt"

type Error struct {
	errors []error
}

func (e *Error) Error() string {
	var str string
	if len(e.errors) > 0 {
		for _, err := range e.errors {
			str += fmt.Sprintf("! %s\n", err)
		}
	}
	return str
}

func (e *Error) Append(err ...error) {
	e.errors = append(e.errors, err...)
}

func (e *Error) IsEmpty() bool {
	if len(e.errors) > 0 {
		return false
	}

	return true
}
