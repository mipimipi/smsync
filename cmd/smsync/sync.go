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

// prog contains attributes that are used to communicate the progress of the
// conversion
type prog struct {
	done    int           // number of files / dirs that have been processed
	total   int           // total number of files / dirs
	errors  int           // number of errors
	elapsed time.Duration // elapsed time
}

// print calculates the elapsed time (based on the start time and the current
// time), the remaining time and the expected end time. The results are printed
// in a specific format
func (prog *prog) print(start time.Time) {
	var remaining time.Duration // remaining time

	// calculate the elapsed time
	prog.elapsed = time.Since(start)

	// calculate the remaining time
	if prog.done > 0 {
		remaining = time.Duration(int64(prog.elapsed) / int64(prog.done) * int64(prog.total-prog.done))
	}

	// print progress
	split := lhlp.SplitDuration(remaining)
	fmt.Printf("\r   To do: %0"+strconv.Itoa(len(strconv.Itoa(prog.total)))+"d | Rem time: %02d:%02d:%02d | Est end: %s", prog.total-prog.done, split[time.Hour], split[time.Minute], split[time.Second], time.Now().Add(remaining).Local().Format("15:04:05"))
}

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
	fmt.Printf(fmGen, "Source", cfg.SrcDirPath)

	// target directory
	fmt.Printf(fmGen, "Destination", cfg.TrgDirPath)

	// last sync time
	if cfg.LastSync.IsZero() {
		fmt.Printf(fmGen, "Last Sync", "Not set, initial sync")
	} else {
		if cli.init {
			fmt.Printf(fmGen, "Last Sync", "Set, but initial sync will be done per cli option")
		} else {
			fmt.Printf(fmGen, "Last Sync", cfg.LastSync.Local())
		}
	}

	// number of CPU's & workers
	fmt.Printf(fmGen, "#CPUs", strconv.Itoa(int(cfg.NumCpus)))
	fmt.Printf(fmGen, "#Workers", strconv.Itoa(int(cfg.NumWrkrs)))

	// conversions
	fmt.Printf(fmGen, "Conversions", "")
	for srcSuffix, cv := range cfg.Cvs {
		if srcSuffix == "*" {
			hasStar = true
			continue
		}
		fmt.Printf(fmRl, srcSuffix, cv.TrgSuffix, cv.NormCvStr)
	}
	if hasStar {
		fmt.Printf(fmRl, "*", cfg.Cvs["*"].TrgSuffix, cfg.Cvs["*"].NormCvStr)
	}
}

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
		prog    prog
		pRes    smsync.ProcRes
		procRes <-chan smsync.ProcRes
		start   = time.Now()
		ticker  = time.NewTicker(time.Second) // ticker to update progress on screen every second
		ticked  = false
		ok      = true
	)

	procRes = f(cfg, wl)

	prog.total = len(*wl)

	// retrieve results and ticks
	for ok {
		select {
		case <-ticker.C:
			ticked = true
			if !verbose {
				prog.print(start)
			}
		case pRes, ok = <-procRes:
			if ok {
				// if ticker hasn't ticked so far: print progress
				if !ticked && !verbose {
					prog.print(start)
				}
				prog.done++

				if verbose {
					printCurrentFile(cfg, pRes.SrcFile)
				}
			} else {
				if !verbose {
					prog.print(start)
					fmt.Println()
				}

				// if all files have been transformed: stop trigger
				ticker.Stop()
			}
		}
	}

	return prog.elapsed, nil
}

// synchronize is the main function of smsync. It triggers the entire sync
// process:
// (1) read configuration
// (2) determine directories and files to be synched
// (3) start processing of these directories and files
func synchronize(level log.Level, verbose bool) error {
	if err := smsync.CreateLogger(level); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			return e
		}
		return err
	}

	// print copyright etc. on command line
	fmt.Println(preamble)

	// read configuration
	cfg, err := smsync.GetCfg(cli.init)
	if err != nil {
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
	_ = runtime.GOMAXPROCS(int(cfg.NumCpus))

	// start automatic progress string which increments every second
	stop := lhlp.ProgressStr(":: Find differences (this can take a few minutes)", 1000)

	// get list of directories and files for sync
	dirs, files := smsync.GetSyncFiles(cfg)

	// stop progress string
	stop <- struct{}{}
	close(stop)

	// if no directories and no files need to be synchec: exit
	if len(*dirs) == 0 && len(*files) == 0 {
		fmt.Println("   Nothing to synchronize. Leaving smsync ...")
		log.Info("Nothing to synchronize")
		return nil
	}

	// print summary and ask user for OK to continue
	if !cli.noConfirm {
		if !lhlp.UserOK(fmt.Sprintf(":: %d directories and %d files to synchronize. Continue", len(*dirs), len(*files))) {
			log.Infof("Synchronization not started due to user input")
			return nil
		}
	}

	// set processing status to "work in progress" in smsync.yaml
	if err := cfg.SetProcStatWIP(); err != nil {
		return err
	}

	// delete all entries of the target directory per cli option
	if cli.init {
		log.Info("Delete all entries of the target directory per cli option")
		if err := smsync.DeleteTrg(cfg); err != nil {
			return err
		}
	}

	// process directories
	fmt.Println("\n:: Process directories")
	durDirs, err := process(cfg, dirs, smsync.ProcessDirs, verbose)
	if err != nil {
		return err
	}

	// process files
	fmt.Println("\n:: Process files")
	durFiles, err := process(cfg, files, smsync.ProcessFiles, verbose)
	if err != nil {
		return err
	}

	// print headline
	fmt.Println("\n:: Done :)")
	// print total duration into a string
	split := lhlp.SplitDuration(durDirs + durFiles)
	totalStr := fmt.Sprintf("%dh %02dmin %02ds", split[time.Hour], split[time.Minute], split[time.Second])
	fmt.Printf("   Processed %d directories and %d files in %s\n", len(*dirs), len(*files), totalStr)

	// update last sync time in config file
	return cfg.UpdateLastSync()
}
