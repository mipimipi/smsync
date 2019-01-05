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
	"path/filepath"
	"strconv"
	"time"

	lhlp "github.com/mipimipi/go-lhlp"
	"github.com/mipimipi/smsync/internal/smsync"
	log "github.com/sirupsen/logrus"
)

// printCfgSummary display a summary of the configuration. The content of the
// configuration files is taken as basis. It's enriched by additional
// information
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
func printProgress(trck *smsync.Tracking, first bool) {
	const (
		format = "%6s %8s %8s %5s %6s %6s %12s %12s %7s" // format string for progress display
		mb     = uint64(1024 * 1024)                     // one megabyte
	)
	var (
		size   string
		avail  string
		status = trck.Status()
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

	if status.Size == 0 {
		size = "- MB"
		avail = "- MB"
	} else {
		size = fmt.Sprintf("%d MB", status.Size/mb)
		avail = fmt.Sprintf("%d MB", status.Avail/int64(mb))
	}

	// print progress (updates the same screen row)
	fmt.Printf("\r"+format,
		strconv.Itoa(status.Todo),
		split(status.Elapsed),
		split(status.Remaining),
		fmt.Sprintf("%2.1f", status.Throughput),
		fmt.Sprintf("%2.2fs", status.AvgDur.Seconds()),
		fmt.Sprintf("%3.1f%%", status.Comp*100),
		size,
		avail,
		strconv.Itoa(status.Errors)) //nolint
}

// printVerbose displays detailed information after each conversion. The name
// of the converted file is displayed relative to the source directory.This
// function is used if the user called smsync with the option --verbose / -v
func printVerbose(cfg *smsync.Config, cvInfo smsync.CvInfo) {
	srcFile, err := filepath.Rel(cfg.SrcDir, cvInfo.SrcFile.Path())
	if err != nil {
		log.Error(err)

	} else {
		fmt.Println("----------")
		fmt.Printf("CONVERTED: %s\n", srcFile)
		fmt.Printf("DURATION : %2.2fs\n", cvInfo.Dur.Seconds())
	}
}
