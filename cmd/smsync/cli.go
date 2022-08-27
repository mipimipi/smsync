package main

// cli.go implements the command line interface for smsync.

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var preamble = `smsync (Smart Music Sync) ` + Version + `
Copyright (C) 2018-2022 Michael Picht <https://github.com/mipimipi/smsync>`

var helpTemplate = preamble + `
{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// root command
var rootCmd = &cobra.Command{
	Use:                   "smsync [options]",
	Version:               Version,
	DisableFlagsInUseLine: true,
	SilenceErrors:         true,
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
		if cli.log {
			level = log.DebugLevel
		} else {
			level = log.ErrorLevel
		}

		// call synchronization (which contains the main logic of smsync)
		return synchronize(level, cli.verbose)
	},
}

// variables to store command line flags
var cli struct {
	log       bool // switch on logging
	init      bool // initialize
	noConfirm bool // don't ask for confirmation
	verbose   bool // print detailed progress
}

func init() {
	// set custom help template
	rootCmd.SetHelpTemplate(helpTemplate)

	// define flag ...
	// - initialize
	rootCmd.Flags().BoolVarP(&cli.init, "init", "i", false, "delete content of target directory and do initial sync ignoring the change times on source side")
	// - logging
	rootCmd.Flags().BoolVarP(&cli.log, "log", "l", false, "switch on logging")
	// - print detailed progress
	rootCmd.Flags().BoolVarP(&cli.verbose, "verbose", "v", false, "print detailed progress")
	// - no confirmation
	rootCmd.Flags().BoolVarP(&cli.noConfirm, "yes", "y", false, "don't ask for confirmation")
}

// Execute executes the root command
func execute() error {
	return rootCmd.Execute()
}
