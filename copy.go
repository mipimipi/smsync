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
	lhlp "github.com/mipimipi/go-lhlp"
)

type tfCopy struct{}

// isValid checks is s represents the copy command
func (tfCopy) isValid(s string) bool {
	return s == tfCopyStr
}

// exec executes a file copy
func (tfCopy) exec(cfg *config, f string) error {
	dstFile, err := assembleDstFile(cfg, f)
	if err != nil {
		return err
	}
	return lhlp.CopyFile(f, dstFile)
}
