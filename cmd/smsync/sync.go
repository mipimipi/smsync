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
	"os"
	"runtime"
	"time"

	lhlp "github.com/mipimipi/go-lhlp"
	"github.com/mipimipi/go-lhlp/file"
	"github.com/mipimipi/smsync/internal/smsync"
	log "github.com/sirupsen/logrus"
)

// process starts the processing of directories and file conversions. It also
// calls the print functions to display the required information onthe command
// line
func process(cfg *smsync.Config, files *[]*file.Info, init bool, verbose bool) *smsync.Tracking {
	log.Debug("cli.process: BEGIN")
	defer log.Debug("cli.process: END")

	var (
		ticker = time.NewTicker(time.Second) // ticker to update progress on screen every second
		ticked = false
	)

	// start processing
	proc := smsync.NewProcess(cfg, files, init)
	proc.Run()

	// print header (if the user doesn't want smsync to be verbose)
	if !verbose {
		printProgress(proc.Trck, true)
	}

loop:
	// retrieve results and ticks
	for {
		select {
		case <-ticker.C:
			ticked = true
			// print progress (if the user doesn't want smsync to be verbose)
			if !verbose {
				proc.Trck.UpdElapsed()
				printProgress(proc.Trck, false)
			}
		case pInfo, ok := <-proc.Trck.PInfo:
			if !ok {
				// if there is no more file to process, the final progress data
				// is displayed (if the user desn't want smsync to be verbose)
				if !verbose {
					printProgress(proc.Trck, false)
					fmt.Println()
				}
				break loop
			}
			// if ticker hasn't ticked so far: print progress (if the user
			// doesn't want smsync to be verbose)
			if !ticked && !verbose {
				printProgress(proc.Trck, false)
			}

			// if the user wants smsync to be verbose, display file (that
			// has been processed) ...
			if verbose {
				printVerbose(cfg, pInfo)
			}
		}
	}

	// if processing has finished: stop ticker
	ticker.Stop()

	// wait for processing
	proc.Wait()

	// return elapsed time
	return proc.Trck
}

// synchronize is the main function of smsync. It triggers the entire sync
// process:
// (1) read configuration
// (2) determine directories and files to be synched
// (3) start processing of these directories and files
func synchronize(level log.Level, verbose bool) error {
	// logger needs to be created before the first log entry is generated!!!
	if err := smsync.CreateLogger(level); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	log.Debug("cli.synchronize: BEGIN")
	defer log.Debug("cli.synchronize: END")

	var (
		cfg   smsync.Config
		files *[]*file.Info
	)

	// print copyright etc. on command line
	fmt.Println(preamble)

	// read configuration
	if err := cfg.Get(cli.init); err != nil {
		return err
	}

	// print summary and ask user for OK
	printCfgSummary(&cfg)
	if !cli.noConfirm {
		if !lhlp.UserOK("\n:: Start synchronization") {
			log.Infof("Synchronization not started due to user input")
			return nil
		}
	}

	// set number of cpus to be used by smsync
	_ = runtime.GOMAXPROCS(int(cfg.NumCpus))

	// start automatic progress string which increments every second
	stop, confirm := lhlp.ProgressStr(":: Find differences (this can take a few minutes)", 1000)

	// get files and directories that need to be synched
	files = smsync.GetSyncFiles(&cfg, cli.init)

	// stop progress string and receive stop confirmation. The confirmation is necessary to not
	// scramble the command line output
	stop <- struct{}{}
	close(stop)
	_ = <-confirm

	// if no files need to be synchec: exit
	if len(*files) == 0 {
		fmt.Println("   Nothing to synchronize. Leaving smsync ...")
		log.Info("Nothing to synchronize")
		return nil
	}

	// print summary and ask user for OK to continue
	if !cli.noConfirm {
		if !lhlp.UserOK(fmt.Sprintf("\n:: %d files and directories to be synchronized. Continue", len(*files))) {
			log.Infof("Synchronization not started due to user input")
			return nil
		}
	}

	// do synchronization / conversion
	fmt.Println("\n:: Synchronization / conversion")
	trck := process(&cfg, files, cli.init, cli.verbose)

	// print final success message
	if trck.TotalNum > trck.Done {
		fmt.Println("\n:: Stopped")
	} else {
		fmt.Println("\n:: Done :)")
	}
	split := lhlp.SplitDuration(trck.Elapsed)
	fmt.Printf("   Processed %d files and directories in %s\n",
		len(*files),
		fmt.Sprintf("%dh %02dmin %02ds",
			split[time.Hour],
			split[time.Minute],
			split[time.Second]))
	if trck.Errors > 0 {
		fmt.Printf("   %d errors during conversion\n", trck.Errors)
	}

	// everything's fine
	return nil
}
