// Copyright (C) 2018 Michael Picht
//
// This file is part of smsync (Smart Music Sync).
//
// smsync is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// gool is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with smsync. If not, see <http://www.gnu.org/licenses/>.

package main

// log.go implements some wrapper functionality for logging

import (
	"path/filepath"

	log "github.com/mipimipi/go-lazylog"
)

// smsync always logs into ./smsync.log
const logFileName = "smsync.log"

// createLogger creates and initializes the logger for smsync
func createLogger(level log.Level) {
	// set log file
	fp, err := filepath.Abs(filepath.Join(".", logFileName))
	if err != nil {
		panic(err.Error())
	}
	log.SetLogFilePath(fp)

	// set log level
	log.SetLevel(level)
}
