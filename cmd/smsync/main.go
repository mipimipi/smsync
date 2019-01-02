// Copyright (C) 2018-2019 Michael Picht
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
	"os"

	log "github.com/sirupsen/logrus"
)

// Version stores version information. It's filled by make (see Makefile)
var Version string

func main() {
	log.Debug("cli.main: START")
	defer log.Debug("cli.main: END")

	if err := execute(); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		os.Exit(1)
	}
}
