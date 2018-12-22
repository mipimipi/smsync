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
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	lhlp "github.com/mipimipi/go-lhlp"
	"github.com/mipimipi/smsync/internal/smsync"
	log "github.com/sirupsen/logrus"
)

// printDirProgress displays the progress of the directory processing
func printDirProgress(prog *smsync.Progress, first bool) {
	// format string for progress display
	var format = "  %6s  %9s  %9s"

	// print headlines for progress display
	if first {
		func() {
			var (
				line    = "------------------------------" // lenght=30
				durNull = "--:--:--"                       // "null" string for display of durations
			)

			fmt.Printf(format+"\n", "", "Elapsed", "Remaining") // nolint, headline 1
			fmt.Printf(format+"\n", "#TODO", "Time", "Time")    // nolint, headline 2
			fmt.Println(line)                                   // separator
			fmt.Printf(format, "0", durNull, durNull)           // nolint
		}()

		return
	}

	// local function to print durations as formatted string (HH:MM:SS)
	split := func(d time.Duration) string {
		sp := lhlp.SplitDuration(d)
		return fmt.Sprintf("%02d:%02d:%02d", sp[time.Hour], sp[time.Minute], sp[time.Second])
	}

	fmt.Printf("\r"+format,
		strconv.Itoa(prog.TotalNum-prog.Done),
		split(prog.Elapsed),
		split(prog.Remaining)) //nolint
}

// printFileProgress displays the progress of the file conversion
func printFileProgress(prog *smsync.Progress, first bool) {
	var (
		format = "%6s %8s %8s %5s %6s %6s %12s %12s %7s" // format string for progress display
		mb     = uint64(1024 * 1024)                     // one megabyte
		size   string
		avail  string
	)

	// print headlines for progress display
	if first {
		func() {
			var (
				line    = "------------------------------------------------------------------------------" // lenght=77
				durNull = "--:--:--"                                                                       // "null" string for display of durations
			)

			fmt.Printf(format+"\n", "", "Elapsed", "Remain", "#Conv", "Avg", "Avg", "Estimated", "Estimated", "")               // nolint, headline 1
			fmt.Printf(format+"\n", "#TODO", "Time", "Time", "/ min", "Durat", "Compr", "Target Size", "Free Space", "#Errors") // nolint, headline 2
			fmt.Println(line)                                                                                                   // separator
			fmt.Printf(format, "-", durNull, durNull, "-", "- s", "- %", "- MB", "- MB", "-")                                   // nolint
		}()

		return
	}

	// local function to print durations as formatted string (HH:MM:SS)
	split := func(d time.Duration) string {
		sp := lhlp.SplitDuration(d)
		return fmt.Sprintf("%02d:%02d:%02d", sp[time.Hour], sp[time.Minute], sp[time.Second])
	}

	if prog.Size == 0 {
		size = "- MB"
		avail = "- MB"
	} else {
		size = fmt.Sprintf("%dMB", prog.Size/mb)
		avail = fmt.Sprintf("%dMB", prog.Avail/int64(mb))
	}

	// print progress (updates the same screen row)
	fmt.Printf("\r"+format,
		strconv.Itoa(prog.TotalNum-prog.Done),
		split(prog.Elapsed),
		split(prog.Remaining),
		fmt.Sprintf("%2.1f", prog.Throughput),
		fmt.Sprintf("%2.2fs", prog.AvgDur.Seconds()),
		fmt.Sprintf("%3.1f%%", prog.Comp*100),
		size,
		avail,
		strconv.Itoa(prog.Errors)) //nolint
}

// printCfgSummary display a summary of the configuration. The content of the
// configuration files is taken as basis, and it's enriched by additional
//information
func printCfgSummary(cfg *smsync.Config) {
	var (
		fmGen   = "   %-12s: %s\n" // format string for general config values
		fmRl    string             // format string for conversion rules
		hasStar bool
	)

	// assemble format string for conversion rules
	{
		var (
			lenTrg int
			lenSrc int
		)
		for srcSuffix, cv := range cfg.Cvs {
			if len(srcSuffix) > lenSrc {
				lenSrc = len(srcSuffix)
			}
			if len(cv.TrgSuffix) > lenTrg {
				lenTrg = len(cv.TrgSuffix)
			}
		}
		fmRl = "       %-" + strconv.Itoa(lenSrc) + "s -> %-" + strconv.Itoa(lenTrg) + "s = %s\n"
	}

	// configuration headline
	fmt.Println("\n:: Configuration")

	// source directory
	fmt.Printf(fmGen, "Source", cfg.SrcDirPath) // nolint

	// directories to exclude
	if len(cfg.Excludes) > 0 {
		fmt.Printf(fmGen, "Exclude (expanded)", "") // nolint
		for _, s := range cfg.Excludes {
			fmt.Printf("       %s\n", s)
		}
	}

	// target directory
	fmt.Printf(fmGen, "Destination", cfg.TrgDirPath) // nolint

	// last sync time
	if cfg.LastSync.IsZero() {
		fmt.Printf(fmGen, "Last Sync", "Not set, initial sync") // nolint
	} else {
		if cli.init {
			fmt.Printf(fmGen, "Last Sync", "Set, but initial sync will be done per cli option") // nolint
		} else {
			fmt.Printf(fmGen, "Last Sync", cfg.LastSync.Local()) // nolint
		}
	}

	// number of CPU's & workers
	fmt.Printf(fmGen, "#CPUs", strconv.Itoa(int(cfg.NumCpus)))     // nolint
	fmt.Printf(fmGen, "#Workers", strconv.Itoa(int(cfg.NumWrkrs))) // nolint

	// conversions
	fmt.Printf(fmGen, "Conversions", "") // nolint
	for srcSuffix, cv := range cfg.Cvs {
		if srcSuffix == "*" {
			hasStar = true
			continue
		}
		fmt.Printf(fmRl, srcSuffix, cv.TrgSuffix, cv.NormCvStr) // nolint
	}
	if hasStar {
		fmt.Printf(fmRl, "*", cfg.Cvs["*"].TrgSuffix, cfg.Cvs["*"].NormCvStr) // nolint
	}
}

// printVerbose displays a file name relative to the source directory (from
// the configuration). This function is used if the user called smsync with the
// option --verbose / -v
func printVerbose(cfg *smsync.Config, pRes smsync.ProcRes) {
	srcFile, err := filepath.Rel(cfg.SrcDirPath, pRes.SrcFile)
	if err != nil {
		log.Error(err)

	} else {
		fmt.Printf("%s -> DONE\n", srcFile)
	}
}

// process is a wrapper around the specific functions for processing dirs or files.
// These functions are passed to process in the function parameter.
func process(cfg *smsync.Config, prog *smsync.Progress, wl *[]lhlp.FileInfo, print func(*smsync.Progress, bool), verbose bool) error {
	var (
		procRes = prog.Res                    // channel to receive processing results
		res     smsync.ProcRes                // processing result
		ticker  = time.NewTicker(time.Second) // ticker to update progress on screen every second
		ticked  = false
		ok      = true
	)

	// print header (if the user doesn't want smsync to be verbose)
	if !verbose {
		print(prog, true)
	}

	// retrieve results and ticks
	for ok {
		select {
		case <-ticker.C:
			ticked = true
			// print progress (if the user doesn't want smsync to be verbose)
			if !verbose {
				print(prog, false)
			}
		case res, ok = <-procRes:
			if ok {
				// if ticker hasn't ticked so far: print progress (if the user
				// doesn't want smsync to be verbose)
				if !ticked && !verbose {
					print(prog, false)
				}
				// if the user wants smsync to be verbose, display file (that
				// has been processed) ...
				if verbose {
					printVerbose(cfg, res)
				}
			} else {
				// if there is no more file to process, the final progress data
				// is displayed (if the user desn't want smsync to be verbose)
				if !verbose {
					print(prog, false)
					fmt.Println()
				}

				// if all files have been transformed: stop trigger
				ticker.Stop()
			}
		}
	}

	// everything's fine
	return nil
}

// synchronize is the main function of smsync. It triggers the entire sync
// process:
// (1) read configuration
// (2) determine directories and files to be synched
// (3) start processing of these directories and files
func synchronize(level log.Level, verbose bool) error {
	var (
		cfg      smsync.Config
		dirProg  *smsync.Progress
		fileProg *smsync.Progress
		errors   <-chan error
		err      error
	)

	if err := smsync.CreateLogger(level); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			return e
		}
		return err
	}

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

	// get list of directories and files for sync
	dirs, files := smsync.GetSyncFiles(&cfg, cli.init)

	// stop progress string and receive stop confirmation. The confirmation is necessary to not
	// scramble the command line output
	stop <- struct{}{}
	close(stop)
	_ = <-confirm

	// if no directories and no files need to be synchec: exit
	if len(*dirs) == 0 && len(*files) == 0 {
		fmt.Println("   Nothing to synchronize. Leaving smsync ...")
		log.Info("Nothing to synchronize")
		return nil
	}

	// print summary and ask user for OK to continue
	if !cli.noConfirm {
		if !lhlp.UserOK(fmt.Sprintf("\n:: %d directories and %d files to synchronize. Continue", len(*dirs), len(*files))) {
			log.Infof("Synchronization not started due to user input")
			return nil
		}
	}

	// start processing
	if dirProg, fileProg, errors, err = smsync.Process(&cfg, dirs, files, cli.init); err != nil {
		return err
	}

	// process directories
	if len(*dirs) > 0 {
		fmt.Println("\n:: Process directories")
		if err = process(&cfg, dirProg, dirs, printDirProgress, verbose); err != nil {
			return err
		}
	}

	// process files
	if len(*files) > 0 {
		fmt.Println("\n:: Process files")
		if err = process(&cfg, fileProg, files, printFileProgress, verbose); err != nil {
			return err
		}
	}

	// print final success message
	fmt.Println("\n:: Done :)")
	split := lhlp.SplitDuration(dirProg.Elapsed + fileProg.Elapsed)
	fmt.Printf("   Processed %d directories and %d files in %s\n",
		len(*dirs),
		len(*files),
		fmt.Sprintf("%dh %02dmin %02ds",
			split[time.Hour],
			split[time.Minute],
			split[time.Second]))

	// receive potential error from smsync.Process
	err = <-errors
	if err != nil {
		return err
	}

	// everything's fine
	return nil
}
