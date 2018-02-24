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

/*
// text formatting structure for gool
type smsyncTextFormatter struct{}

// Format prints one log line in smsync specific format
func (f *smsyncTextFormatter) Format(entry *log.Entry) ([]byte, error) {
	var b *bytes.Buffer

	// initialize buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// write log level
	if _, err := b.WriteString(fmt.Sprintf("[%-7s]:", entry.Level.String())); err != nil {
		panic(err.Error())
	}

	// write custom data fields
	for _, value := range entry.Data {
		if b.Len() > 0 {
			if err := b.WriteByte(' '); err != nil {
				panic(err.Error())
			}
		}
		stringVal, ok := value.(string)
		if !ok {
			stringVal = fmt.Sprint(value)
		}
		if _, err := b.WriteString("[" + stringVal + "]"); err != nil {
			panic(err.Error())
		}
	}

	// write log message
	if err := b.WriteByte(' '); err != nil {
		panic(err.Error())
	}
	if _, err := b.WriteString(entry.Message); err != nil {
		panic(err.Error())
	}

	// new line
	if err := b.WriteByte('\n'); err != nil {
		panic(err.Error())
	}

	return b.Bytes(), nil
}
*/

// createLogger creates and initializes the logger for smsync
func createLogger(level log.Level) {
	/*
		// get absolute filepath for log file
		fp, err := filepath.Abs(filepath.Join(".", logFileName))
		if err != nil {
			panic(err.Error())
		}

		// delete log file if it already exists
		exists, err := lhlp.FileExists(fp)
		if err != nil {
			panic(err.Error())
		}
		if exists {
			if err = os.Remove(fp); err != nil {
				panic(err.Error())
			}
		}

		// create log file
		f, err := os.Create(fp)
		if err != nil {
			fmt.Printf("Log file could not be created/opened: %v", err)
			return
		}

		// set log file as output for logging
		log.SetOutput(f)
	*/
	fp, err := filepath.Abs(filepath.Join(".", logFileName))
	if err != nil {
		panic(err.Error())
	}
	log.SetLogFilePath(fp)

	// log all messages
	log.SetLevel(level)

	/*
		// set custom formatter
		log.SetFormatter(new(smsyncTextFormatter))
	*/
}
