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

// cli.go implements the command line interface for smsync.

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
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
	Args:                  cobra.NoArgs,
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

		// call synchronization (which contains the main logic of smsync)
		return synchronize(level)
	},
}

// variables to store command line flags
var cli struct {
	doLog     bool // do logging
	init      bool // initialize
	addOnly   bool // only add files and directories
	noConfirm bool // don't ask for confirmation
}

func init() {
	// set custom help template
	rootCmd.SetHelpTemplate(helpTemplate)

	// define flag ...
	// - logging
	rootCmd.Flags().BoolVarP(&cli.doLog, "log", "l", false, "switch on logging")
	// - initialize
	rootCmd.Flags().BoolVarP(&cli.init, "initialize", "i", false, "delete content of target directory and do initial sync")
	// - add only
	rootCmd.Flags().BoolVarP(&cli.addOnly, "add-only", "a", false, "only add files")
	// - no confirmation
	rootCmd.Flags().BoolVarP(&cli.noConfirm, "yes", "y", false, "don't ask for confirmation")
}

// Execute executes the root command
func execute() error {
	return rootCmd.Execute()
}
