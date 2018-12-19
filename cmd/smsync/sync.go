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
	"github.com/ricochet2200/go-disk-usage/du"
	log "github.com/sirupsen/logrus"
)

// size returns the size of a file
func size(f string) uint64 {
	fi, err := os.Stat(f)
	if err != nil {
		log.Errorf("%v", err)
		return 0
	}
	if fi.IsDir() {
		return 0
	}
	return uint64(fi.Size())
}

// totalSize returns the aggregated size of a list of files
func totalSize(fs []*string) uint64 {
	var sz uint64
	for _, f := range fs {
		sz += size(*f)
	}
	return sz
}

// prog contains attributes that are used to communicate the progress of the
// conversion
type prog struct {
	done      int           // number of files / dirs that have been processed
	totalNum  int           // total number of files / dirs
	totalSize uint64        // total aggregated size of source files
	srcSize   uint64        // cumulated size of source files
	trgSize   uint64        // cumulated size of target files
	diskspace uint64        // available space on target device
	errors    int           // number of errors
	start     time.Time     // start time of processing
	elapsed   time.Duration // elapsed time
}

// format string for progress display
var format = "%8s %10s %10s %17s"

// printHeader prints the headline for progress display
func (prog *prog) printHeader() {
	var (
		line    = "------------------------------------------------" // lenght=48
		durNull = "--:--:--"                                         // "null" string for display of durations
	)

	fmt.Printf(format+"\n", "", "Elapsed", "Remaining", "Estimated")   // nolint, headline 1
	fmt.Printf(format+"\n", "#TODO", "Time", "Time", "Free Diskspace") // nolint, headline 2
	fmt.Println(line)                                                  // separator
	fmt.Printf(format, "0", durNull, durNull, "0 MB")                  // nolint
}

// print display the progress of the conversion. It takesthe attributes of the
// structure prog as basis and calculates additional data, such as elapsed and
// remaining time and the estimated free diskspace
func (prog *prog) print() {
	var (
		remaining time.Duration         // remaining time
		mb        = uint64(1024 * 1024) // one megabyte
		avail     uint64                // estimated free diskspace
	)

	// calculate the elapsed time
	prog.elapsed = time.Since(prog.start)

	// calculate the remaining time
	if prog.done > 0 {
		remaining = time.Duration(int64(prog.elapsed) / int64(prog.done) * int64(prog.totalNum-prog.done))
	}

	// local function to print durations as formatted string (HH:MM:SS)
	split := func(d time.Duration) string {
		sp := lhlp.SplitDuration(d)
		return fmt.Sprintf("%02d:%02d:%02d", sp[time.Hour], sp[time.Minute], sp[time.Second])
	}

	// calculates estimated available disk space
	if prog.srcSize > 0 {
		avail = uint64((float64(prog.diskspace) - float64(prog.trgSize)/float64(prog.srcSize)*float64(prog.totalSize)) / float64(mb))
	} else {
		avail = prog.diskspace / mb
	}

	// print progress (updates the same screen row)
	fmt.Printf("\r"+format, strconv.Itoa(prog.totalNum-prog.done), split(prog.elapsed), split(remaining), fmt.Sprintf("%d MB", avail)) //nolint
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
		fmt.Printf(fmGen, "Exclude", "") // nolint
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

// printCurrentFile displays a file name relative to the source directory (from
// the configuration). This function is used if the user called smsync with the
// option --verbose / -v
func printCurrentFile(cfg *smsync.Config, f string) {
	s, err := filepath.Rel(cfg.SrcDirPath, f)
	if err != nil {
		log.Error(err)
	}
	fmt.Printf("%s DONE\n", s)
}

// process is a wrapper around the specific functions for processing dirs or files.
// These functions are passed to process in the function parameter.
func process(cfg *smsync.Config, wl *[]*string, f func(*smsync.Config, *[]*string) <-chan smsync.ProcRes, verbose bool) (time.Duration, error) {
	var (
		p = prog{totalNum: len(*wl),
			totalSize: totalSize(*wl),
			diskspace: du.NewDiskUsage(cfg.SrcDirPath).Available(),
			start:     time.Now()} // progress structure
		pRes    smsync.ProcRes                // structure to return the processing result
		procRes = f(cfg, wl)                  // call of conversion
		ticker  = time.NewTicker(time.Second) // ticker to update progress on screen every second
		ticked  = false
		ok      = true
	)

	// print progress header (if the user doesn't want smsync to be verbose)
	if !verbose {
		p.printHeader()
	}

	// retrieve results and ticks
	for ok {
		select {
		case <-ticker.C:
			ticked = true
			// print progress (if the user doesn't want smsync to be verbose)
			if !verbose {
				p.print()
			}
		case pRes, ok = <-procRes:
			if ok {
				// if ticker hasn't ticked so far: print progress (if the user
				// doesn't want smsync to be verbose)
				if !ticked && !verbose {
					p.print()
				}
				// if the user wants smsync to be verbose, display file (that
				// has been processed) ...
				if verbose {
					printCurrentFile(cfg, pRes.SrcFile)
				} else {
					// ... otherwise update values in progress structure
					p.done++                        // increase number of processed files
					p.srcSize += size(pRes.SrcFile) // aggregate sizes of source files
					p.trgSize += size(pRes.TrgFile) // aggregate sizes of target files
				}
			} else {
				// if there is no more file to process, the final progress data
				// is displayed (if the user desn't want smsync to be verbose)
				if !verbose {
					p.print()
					fmt.Println()
				}

				// if all files have been transformed: stop trigger
				ticker.Stop()
			}
		}
	}

	// return elapsed time (needed to display final success message)
	return p.elapsed, nil
}

// synchronize is the main function of smsync. It triggers the entire sync
// process:
// (1) read configuration
// (2) determine directories and files to be synched
// (3) start processing of these directories and files
func synchronize(level log.Level, verbose bool) error {
	var (
		cfg      smsync.Config
		durDirs  time.Duration
		durFiles time.Duration
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

	// set processing status to "work in progress" in smsync.yaml
	if err := cfg.SetProcStatWIP(); err != nil {
		return err
	}

	// delete all entries of the target directory if requested per cli option
	if cli.init {
		log.Info("Delete all entries of the target directory per cli option")
		if err := smsync.DeleteTrg(&cfg); err != nil {
			return err
		}
	}

	// process directories
	if len(*dirs) > 0 {
		fmt.Println("\n:: Process directories")
		if durDirs, err = process(&cfg, dirs, smsync.ProcessDirs, verbose); err != nil {
			return err
		}
	}

	// process files
	if len(*files) > 0 {
		fmt.Println("\n:: Process files")
		if durFiles, err = process(&cfg, files, smsync.ProcessFiles, verbose); err != nil {
			return err
		}
	}

	// print headline
	fmt.Println("\n:: Done :)")
	// print total duration into a string
	split := lhlp.SplitDuration(durDirs + durFiles)
	totalStr := fmt.Sprintf("%dh %02dmin %02ds", split[time.Hour], split[time.Minute], split[time.Second])
	fmt.Printf("   Processed %d directories and %d files in %s\n", len(*dirs), len(*files), totalStr)

	// update config file
	return cfg.SetProcEnd()
}
