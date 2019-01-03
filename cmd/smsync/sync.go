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
	"runtime"
	"time"

	"github.com/eiannone/keyboard"
	lhlp "github.com/mipimipi/go-lhlp"
	"github.com/mipimipi/go-lhlp/file"
	"github.com/mipimipi/smsync/internal/smsync"
	log "github.com/sirupsen/logrus"
)

// listenStop waits for <ESC> pressed on keyboard as stop signal
func listenStop() (stop chan struct{}) {
	stop = make(chan struct{})

	go func() {
		if _, key, _ := keyboard.GetSingleKey(); key == keyboard.KeyEsc {
			stop <- struct{}{}
			close(stop)
		}
	}()

	return stop
}

// process starts the processing of directories and file conversions. It also
// calls the print functions to display the required information onthe command
// line
func process(cfg *smsync.Config, files *[]*file.Info, init bool, verbose bool) {
	log.Debug("cli.process: BEGIN")
	defer log.Debug("cli.process: END")

	var (
		ticker   = time.NewTicker(time.Second) // ticker to update progress on screen every second
		ticked   = false                       // has ticker ticked?
		wantstop = false                       // stop wanted?
	)

	// start processing
	proc := smsync.NewProcess(cfg, files, init)
	proc.Run()

	// channel for stop from keyboard. deferred close is necessary since if
	// processing hasn't been stopped, listenStop is still waiting for a key
	// to be pressed
	defer keyboard.Close()
	stop := listenStop()

	// print header (if the user doesn't want smsync to be verbose)
	if !verbose {
		printProgress(proc.Trck, true, false)
	}

loop:
	// retrieve results and ticks
	for {
		select {
		case <-ticker.C:
			ticked = true
			// print progress (if the user doesn't want smsync to be verbose)
			if !verbose {
				printProgress(proc.Trck, false, wantstop)
			}
		case pInfo, ok := <-proc.Trck.Out:
			if !ok {
				// if there is no more file to process, the final progress data
				// is displayed (if the user doesn't want smsync to be verbose)
				if !verbose {
					printProgress(proc.Trck, false, false)
					fmt.Println()
				}
				break loop
			}
			// if the user wants smsync to be verbose, display detailed info
			if verbose {
				printVerbose(cfg, pInfo)
				continue
			}
			// if ticker hasn't ticked so far: print progress
			if !ticked {
				printProgress(proc.Trck, false, wantstop)
			}
		case _, ok := <-stop:
			if ok {
				wantstop = true
				proc.Stop()
			}
		}
	}

	ticker.Stop()

	// wait for processing to be finished
	proc.Wait()

	// print final success message
	printFinal(proc.Trck, verbose)
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

	// print copyright etc. on command line
	fmt.Println(preamble)

	// read configuration
	cfg := new(smsync.Config)
	if err := cfg.Get(cli.init); err != nil {
		return err
	}

	// print summary and ask user for OK
	printCfgSummary(cfg)
	if !cli.noConfirm {
		if !lhlp.UserOK("\n:: Start synchronization") {
			log.Infof("Synchronization not started due to user input")
			return nil
		}
	}

	// set number of cpus to be used by smsync
	runtime.GOMAXPROCS(int(cfg.NumCpus))

	// start automatic progress string which increments every second
	stop, confirm := lhlp.ProgressStr(":: Find differences (this can take a few minutes)", 1000)

	// get files and directories that need to be synched
	files := smsync.GetSyncFiles(cfg, cli.init)

	// stop progress string and receive stop confirmation. The confirmation is necessary to not
	// scramble the command line output
	close(stop)
	<-confirm

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
	fmt.Println("\n:: Synchronization / conversion (PRESS <ESC> TO STOP)")
	process(cfg, files, cli.init, cli.verbose)

	// everything's fine
	return nil
}
