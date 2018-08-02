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

// cli.go implements the command line interface for smsync.

import (
	"fmt"
	"os"

	log "github.com/mipimipi/logrus"
	"github.com/spf13/cobra"
)

var preamble = `smsync (Smart Music Sync) ` + Version + `
Copyright (C) 2018 Michael Picht <https://github.com/mipimipi/smsync>`

var helpTemplate = preamble + `
{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// root command
var rootCmd = &cobra.Command{
	Use:                   "smsync [options]",
	Version:               Version,
	DisableFlagsInUseLine: true,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// retrieve flags
		if err := cmd.ParseFlags(args); err != nil {
			if _, e := fmt.Fprintf(os.Stderr, "Error during parsing of flags: %v", err); e != nil {
				return e
			}
			return err
		}

		// set up logging
		var level log.Level
		if cli.doLog {
			level = log.DebugLevel
		} else {
			level = log.ErrorLevel
		}
		if err := createLogger(level); err != nil {
			if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
				return e
			}
			return err
		}

		// print copyright etc. on command line
		fmt.Println(preamble)

		// call synchronization (which contains the main logic of smsync)
		return synchronize()
	},
}

// variables to store command line flags
var cli struct {
	doLog   bool // do logging
	addOnly bool // only add files and directories
}

func init() {
	// set custom help template
	rootCmd.SetHelpTemplate(helpTemplate)

	// define flag for logging
	rootCmd.Flags().BoolVarP(&cli.doLog, "log", "l", false, "switch on logging")

	// define flag for add only
	rootCmd.Flags().BoolVarP(&cli.addOnly, "add-only", "a", false, "only add files")
}

// Execute executes the root command
func execute() error {
	return rootCmd.Execute()
}
