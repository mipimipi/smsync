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

type cvCopy struct{}

// normParams checks if the string contains a valid set of parameters and
// normalizes it (e.g. removes blanks and sets default values)
func (cvCopy) normParams(s *string) error {
	// set *s to lower case and remove blanks
	*s = strings.Trim(strings.ToLower(*s), " ")

	if *s != cvCopyStr {
		if *s == "" {
			*s = cvCopyStr
		} else {
			return fmt.Errorf("'%s' is not a valid copy conversion", *s)
		}
	}
	return nil
}

// exec executes a file copy
func (cvCopy) exec(cfg *config, f string) error {
	trgFile, err := assembleTrgFile(cfg, f)
	if err != nil {
		return err
	}
	return lhlp.CopyFile(f, trgFile)
}
