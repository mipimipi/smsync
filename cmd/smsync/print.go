package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	t "gitlab.com/go-utilities/time"
	"gitlab.com/mipimipi/smsync/internal/smsync"
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
	fmt.Printf(fmGen, "Source", cfg.SrcDir.Path()) // nolint

	// directories to exclude
	if len(cfg.Excludes) > 0 {
		fmt.Printf(fmGen, "Exclude (expanded)", "") // nolint
		for _, s := range cfg.Excludes {
			fmt.Printf("       %s\n", s)
		}
	}

	// target directory
	fmt.Printf(fmGen, "Destination", cfg.TrgDir.Path()) // nolint

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

func printFinal(trck *smsync.Tracking, verbose bool) {
	if trck.TotalNum > trck.Done {
		fmt.Printf("\n:: STOPPED! %d files or directories left to process\n", trck.TotalNum-trck.Done)
	} else {
		fmt.Printf("\n:: Done :)\n")
	}
	split := t.SplitDuration(trck.Elapsed)
	fmt.Printf("   Processed %d files and directories in %s\n",
		trck.Done,
		fmt.Sprintf("%dh %02dmin %02ds",
			split[time.Hour],
			split[time.Minute],
			split[time.Second]))
	if verbose {
		fmt.Printf("   Conv errs: %d\n", trck.Errors)
		fmt.Printf("   #Conv/min: %2.1f\n", trck.Throughput)
		fmt.Printf("   Avg durat: %2.2fs\n", trck.AvgDur.Seconds())
		fmt.Printf("   Avg compr: %3.1f%%\n", 100*trck.Comp)
	}
}

// printProgress displays the progress of the file conversion
func printProgress(trck *smsync.Tracking, first, wantstop bool) {
	const (
		format = "%6s %8s %8s %7s %6s %6s %11s %11s %5s %4s" // format string for progress display
		mb     = uint64(1024 * 1024)                         // one megabyte
	)
	var (
		size  = "- MB"
		avail = "- MB"
		stop  = "    "
	)

	// print headlines for progress display
	if first {
		func() {
			const (
				line    = "----------------------------------------------------------------------------" // length=75
				durNull = "--:--:--"                                                                     // "null" string for display of durations
			)

			fmt.Printf(format+"\n", "", "Elapsed", "Remain", "#Conv", "Avg", "Avg", "Estimated", "Estimated", "", "")             // nolint, headline 1
			fmt.Printf(format+"\n", "#TODO", "Time", "Time", "/ min", "Durat", "Compr", "Target Size", "Free Space", "#Errs", "") // nolint, headline 2
			fmt.Println(line)                                                                                                     // separator
			fmt.Printf(format, "-", durNull, durNull, "-", "- s", "- %", "- MB", "- MB", "-", "")                                 // nolint
		}()

		return
	}

	// local function to print durations as formatted string (HH:MM:SS)
	split := func(d time.Duration) string {
		sp := t.SplitDuration(d)
		return fmt.Sprintf("%02d:%02d:%02d", sp[time.Hour], sp[time.Minute], sp[time.Second])
	}

	if trck.Size > 0 {
		size = fmt.Sprintf("%d MB", trck.Size/mb)
		avail = fmt.Sprintf("%d MB", trck.Avail/int64(mb))
	}

	if wantstop {
		stop = "STOP"
	}

	// print progress (updates the same screen row)
	fmt.Printf("\r"+format,
		strconv.Itoa(trck.TotalNum-trck.Done),
		split(time.Since(trck.Started)),
		split(trck.Remaining),
		fmt.Sprintf("%2.1f", trck.Throughput),
		fmt.Sprintf("%2.2fs", trck.AvgDur.Seconds()),
		fmt.Sprintf("%3.1f%%", trck.Comp*100),
		size,
		avail,
		strconv.Itoa(trck.Errors),
		stop) //nolint
}

// printVerbose displays detailed information after each conversion. The name
// of the converted file is displayed relative to the source directory.This
// function is used if the user called smsync with the option --verbose / -v
func printVerbose(cfg *smsync.Config, pInfo smsync.ProcInfo) {
	srcFile, err := filepath.Rel(cfg.SrcDir.Path(), pInfo.SrcFile.Path())
	if err != nil {
		log.Error(err)
	} else {
		fmt.Println("----------")
		fmt.Printf("CONVERTED: %s\n", srcFile)
		fmt.Printf("DURATION : %2.2fs\n", pInfo.Dur.Seconds())
		if pInfo.Err != nil {
			fmt.Println("STATUS   : ERROR")
		} else {
			fmt.Println("STATUS   : OK")
		}
	}
}
