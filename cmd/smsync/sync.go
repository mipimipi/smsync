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
	"github.com/mipimipi/go-lhlp/file"
	"github.com/mipimipi/smsync/internal/smsync"
	log "github.com/sirupsen/logrus"
)

// printCfgSummary display a summary of the configuration. The content of the
// configuration files is taken as basis, and it's enriched by additional
//information
func printCfgSummary(cfg *smsync.Config) {
	const fmGen = "   %-12s: %s\n" // format string for general config values
	var (
		fmRl    string // format string for conversion rules
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
	fmt.Printf(fmGen, "Source", cfg.SrcDir) // nolint

	// directories to exclude
	if len(cfg.Excludes) > 0 {
		fmt.Printf(fmGen, "Exclude (expanded)", "") // nolint
		for _, s := range cfg.Excludes {
			fmt.Printf("       %s\n", s)
		}
	}

	// target directory
	fmt.Printf(fmGen, "Destination", cfg.TrgDir) // nolint

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

// printProgress displays the progress of the file conversion
func printProgress(prog *smsync.Progress, first bool) {
	const (
		format = "%6s %8s %8s %5s %6s %6s %12s %12s %7s" // format string for progress display
		mb     = uint64(1024 * 1024)                     // one megabyte
	)
	var (
		size  string
		avail string
	)

	// print headlines for progress display
	if first {
		func() {
			const (
				line    = "------------------------------------------------------------------------------" // length=77
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
		size = fmt.Sprintf("%d MB", prog.Size/mb)
		avail = fmt.Sprintf("%d MB", prog.Avail/int64(mb))
	}

	// print progress (updates the same screen row)
	fmt.Printf("\r"+format,
		strconv.Itoa(prog.TotalNum-prog.Done),
		split(prog.Elapsed()),
		split(prog.Remaining()),
		fmt.Sprintf("%2.1f", prog.Throughput()),
		fmt.Sprintf("%2.2fs", prog.AvgDur.Seconds()),
		fmt.Sprintf("%3.1f%%", prog.Comp*100),
		size,
		avail,
		strconv.Itoa(prog.Errors)) //nolint
}

// printVerbose displays a file name relative to the source directory (from
// the configuration). This function is used if the user called smsync with the
// option --verbose / -v
func printVerbose(cfg *smsync.Config, res smsync.ProcRes) {
	srcFile, err := filepath.Rel(cfg.SrcDir, res.SrcFile.Path())
	if err != nil {
		log.Error(err)

	} else {
		fmt.Println("----------")
		fmt.Printf("CONVERTED: %s\n", srcFile)
		fmt.Printf("DURATION : %2.2fs\n", res.Dur.Seconds())
	}
}

// process is a wrapper around the specific functions for processing dirs or files.
// These functions are passed to process in the function parameter.
func process(cfg *smsync.Config, dirs *file.InfoSlice, files *file.InfoSlice, init bool, verbose bool) (time.Duration, error) {
	log.Debug("cli.process: START")
	defer log.Debug("cli.process: END")

	var (
		res         smsync.ProcRes                // processing result
		ticker      = time.NewTicker(time.Second) // ticker to update progress on screen every second
		ticked      = false
		prog        *smsync.Progress
		err         error
		ok          = true
		errOccurred = false
		errors      <-chan error
		done        <-chan struct{}
	)

	// start processing
	if prog, errors, done, err = smsync.Process(cfg, dirs, files, init); err != nil {
		errOccurred = true
		return 0, nil
	}

	// print header (if the user doesn't want smsync to be verbose)
	if !verbose {
		printProgress(prog, true)
	}

	// retrieve results and ticks
	for ok {
		select {
		case <-ticker.C:
			ticked = true
			// print progress (if the user doesn't want smsync to be verbose)
			if !verbose {
				printProgress(prog, false)
			}
		case res, ok = <-prog.Res:
			if ok {
				// if ticker hasn't ticked so far: print progress (if the user
				// doesn't want smsync to be verbose)
				if !ticked && !verbose {
					printProgress(prog, false)
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
					printProgress(prog, false)
					fmt.Println()
				}

				// if all files have been transformed: stop trigger
				ticker.Stop()
			}
		case err, ok = <-errors:
			if err != nil {
				errOccurred = true
			}
		case _ = <-done:
		}
	}

	if errOccurred {
		return prog.Elapsed(), fmt.Errorf("At least one error occurred during processing")
	}

	return prog.Elapsed(), nil
}

// synchronize is the main function of smsync. It triggers the entire sync
// process:
// (1) read configuration
// (2) determine directories and files to be synched
// (3) start processing of these directories and files
func synchronize(level log.Level, verbose bool) error {
	// logger needs to be created before the first log entry is generated!!!
	if err := smsync.CreateLogger(level); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			return e
		}
		return err
	}

	log.Debug("cli.synchronize: START")
	defer log.Debug("cli.synchronize: END")

	var (
		cfg         smsync.Config
		dirs        *file.InfoSlice
		files       *file.InfoSlice
		elapsed     time.Duration
		err         error
		errOccurred = false
	)

	defer func() {
		if errOccurred {
			fmt.Printf("At least one error occured. Check %s", smsync.LogFile)
		}
	}()

	// print copyright etc. on command line
	fmt.Println(preamble)

	// read configuration
	if err := cfg.Get(cli.init); err != nil {
		errOccurred = true
		return nil
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
	if dirs, files, err = smsync.GetSyncFiles(&cfg, cli.init); err != nil {
		errOccurred = true
		return nil
	}

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

	// process files
	if len(*files) > 0 {
		fmt.Println("\n:: Process files")
		if elapsed, err = process(&cfg, dirs, files, cli.init, cli.verbose); err != nil {
			errOccurred = true
			return nil
		}
	}

	// print final success message
	fmt.Println("\n:: Done :)")
	split := lhlp.SplitDuration(elapsed)
	fmt.Printf("   Processed %d directories and %d files in %s\n",
		len(*dirs),
		len(*files),
		fmt.Sprintf("%dh %02dmin %02ds",
			split[time.Hour],
			split[time.Minute],
			split[time.Second]))

	// everything's fine
	return nil
}
