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

package smsync

// log.go implements some wrapper functionality for logging

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lhlp "github.com/mipimipi/go-lhlp"
	log "github.com/sirupsen/logrus"
)

const logFileName = "smsync.log" // smsync always logs into ./smsync.log

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
	if _, err := b.WriteString(fmt.Sprintf("[%-7s]:", strings.ToUpper(entry.Level.String()))); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		return nil, err
	}

	// write custom data fields
	for _, value := range entry.Data {
		if b.Len() > 0 {
			if err := b.WriteByte(' '); err != nil {
				if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
					panic(e.Error())
				}
				return nil, err
			}
		}
		stringVal, ok := value.(string)
		if !ok {
			stringVal = fmt.Sprint(value)
		}
		if _, err := b.WriteString("[" + stringVal + "]"); err != nil {
			if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
				panic(e.Error())
			}
			return nil, err
		}
	}

	// write log message
	if err := b.WriteByte(' '); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		return nil, err
	}
	if _, err := b.WriteString(entry.Message); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		return nil, err
	}

	// new line
	if err := b.WriteByte('\n'); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		return nil, err
	}

	return b.Bytes(), nil
}

// CreateLogger creates and initializes the logger for smsync
func CreateLogger(level log.Level) error {
	// set log file
	fp, err := filepath.Abs(filepath.Join(".", logFileName))
	if err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		return err
	}

	// delete log file if it already exists
	exists, err := lhlp.FileExists(fp)
	if err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		return err
	}
	if exists {
		if err = os.Remove(fp); err != nil {
			if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
				panic(e.Error())
			}
			return err
		}
	}

	// create log file
	f, err := os.Create(fp)
	if err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		return err
	}

	// set log file
	log.SetOutput(f)

	// set log level
	log.SetLevel(level)

	// set custom formatter
	log.SetFormatter(new(smsyncTextFormatter))

	return nil
}
