// Copyright (C) 2018 Michael Picht
//
// This file is part of go-lazylog.
//
// go-lazylog is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-lazylog is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-lhlp. If not, see <http://www.gnu.org/licenses/>.

// Package lazylog is a wrapper for logrus (github.com/sirupsen/logrus)
// TODO
package lazylog

import (
	"fmt"
	"io"
	"os"

	lhlp "github.com/mipimipi/go-lhlp"
	log "github.com/sirupsen/logrus"
)

type (
	Entry     log.Entry
	Formatter log.Formatter
	Level     log.Level
)

const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel Level = iota
	// FatalLevel level. Logs and then calls `os.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

var (
	logFile     *os.File
	logFilePath string
)

func setLogFile() {
	if logFile != nil || len(logFilePath) == 0 {
		return
	}

	fmt.Println("HALLO")

	// delete log file if it already exists
	exists, err := lhlp.FileExists(logFilePath)
	if err != nil {
		panic(err.Error())
	}
	if exists {
		if err = os.Remove(logFilePath); err != nil {
			panic(err.Error())
		}
	}

	// create log file
	logFile, err := os.Create(logFilePath)
	if err != nil {
		panic("LazyLog: Log file could not be created/opened: " + err.Error())
	}

	// set log file as output for logging
	SetOutput(logFile)
}

// Debug logs a message at level Debug on the standard logger
func Debug(args ...interface{}) {
	if Level(log.GetLevel()) < DebugLevel {
		return
	}
	setLogFile()
	log.Debug(args)
}

// Debugf logs a message at level Debug on the standard logger
func Debugf(format string, args ...interface{}) {
	if Level(log.GetLevel()) < DebugLevel {
		return
	}
	setLogFile()
	log.Debugf(format, args)
}

// Debugln logs a message at level Debug on the standard logger
func Debugln(args ...interface{}) {
	if Level(log.GetLevel()) < DebugLevel {
		return
	}
	setLogFile()
	log.Debug(args)
}

// Error logs a message at level Error on the standard logger
func Error(args ...interface{}) {
	if Level(log.GetLevel()) < ErrorLevel {
		return
	}
	setLogFile()
	log.Error(args)
}

// Errorf logs a message at level Error on the standard logger
func Errorf(format string, args ...interface{}) {
	if Level(log.GetLevel()) < ErrorLevel {
		return
	}
	setLogFile()
	log.Errorf(format, args)
}

// Errorln logs a message at level Error on the standard logger
func Errorln(args ...interface{}) {
	if Level(log.GetLevel()) < ErrorLevel {
		return
	}
	setLogFile()
	log.Errorln(args)
}

// Fatal logs a message at level Fatal on the standard logger
func Fatal(args ...interface{}) {
	if Level(log.GetLevel()) < FatalLevel {
		return
	}
	setLogFile()
	log.Fatal(args)
}

// Fatalf logs a message at level Fatal on the standard logger
func Fatalf(format string, args ...interface{}) {
	if Level(log.GetLevel()) < FatalLevel {
		return
	}
	setLogFile()
	log.Fatalf(format, args)
}

// Fatalln logs a message at level Fatal on the standard logger
func Fatalln(args ...interface{}) {
	if Level(log.GetLevel()) < FatalLevel {
		return
	}
	setLogFile()
	log.Fatalln(args)
}

// Info logs a message at level Info on the standard logger
func Info(args ...interface{}) {
	if Level(log.GetLevel()) < InfoLevel {
		return
	}
	setLogFile()
	log.Info(args)
}

// Infof logs a message at level Info on the standard logger
func Infof(format string, args ...interface{}) {
	if Level(log.GetLevel()) < InfoLevel {
		return
	}
	setLogFile()
	log.Infof(format, args)
}

// Infoln logs a message at level Info on the standard logger
func Infoln(args ...interface{}) {
	if Level(log.GetLevel()) < InfoLevel {
		return
	}
	setLogFile()
	log.Infoln(args)
}

// Panic logs a message at level Panic on the standard logger
func Panic(args ...interface{}) {
	if Level(log.GetLevel()) < PanicLevel {
		return
	}
	setLogFile()
	log.Panic(args)
}

// Panicf logs a message at level Panic on the standard logger
func Panicf(format string, args ...interface{}) {
	if Level(log.GetLevel()) < PanicLevel {
		return
	}
	setLogFile()
	log.Panicf(format, args)
}

// Panicln logs a message at level Panic on the standard logger
func Panicln(args ...interface{}) {
	if Level(log.GetLevel()) < PanicLevel {
		return
	}
	setLogFile()
	log.Panicln(args)
}

// Warn logs a message at level Warn on the standard logger
func Warn(args ...interface{}) {
	if Level(log.GetLevel()) < WarnLevel {
		return
	}
	setLogFile()
	log.Warn(args)
}

// Warnf logs a message at level Warn on the standard logger
func Warnf(format string, args ...interface{}) {
	if Level(log.GetLevel()) < WarnLevel {
		return
	}
	setLogFile()
	log.Warnf(format, args)
}

// Warnln logs a message at level Warn on the standard logger
func Warnln(args ...interface{}) {
	if Level(log.GetLevel()) < WarnLevel {
		return
	}
	setLogFile()
	log.Warnln(args)
}

// SetFormatter sets the standard logger formatter
func SetFormatter(formatter Formatter) {
	log.SetFormatter(log.Formatter(formatter))
}

// SetLevel sets the standard logger level
func SetLevel(level Level) {
	log.SetLevel(log.Level(level))
}

// SetLogFilePath sets the filepath of the log file
func SetLogFilePath(filePath string) {
	logFilePath = filePath
}

// SetOutput sets the standard logger output
func SetOutput(out io.Writer) {
	log.SetOutput(out)
}
