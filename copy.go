// Copyright (C) 2018 Michael Picht
//
// This file is part of smsync (Smart Music Sync).
//
// smsync is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// smsync is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with smsync. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"strings"

	lhlp "github.com/mipimipi/go-lhlp"
)

// implementation of interface "conversion" for simple file copy
type cvCopy struct{}

// exec executes simple file copy
func (cvCopy) exec(srcFile string, trgFile string, cvStr string) error {
	return lhlp.CopyFile(srcFile, trgFile)
}

// normCvStr checks if the parameters string from config file is either empty
// or equals "copy". If that's the case, "copy" is returned. Otherwise an error
// is returned.
func (cvCopy) normCvStr(s string) (string, error) {
	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	if s != cvCopyStr && s != "" {
		return "", fmt.Errorf("'%s' is not a valid copy conversion", s)
	}
	return cvCopyStr, nil
}
