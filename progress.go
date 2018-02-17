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
	"strconv"
	"time"

	lhlp "github.com/mipimipi/go-lhlp"
)

// mode constans for progressTable
const (
	progModeDirs  = "d" // processing of directories
	progModeFiles = "f" // processing of files
)

// constants for colors etc. for command line output
const (
	cBold      = "\033[1m"
	cNormal    = "\033[0m"
	cFGDefault = "\033[39m"
	cFGRed     = "\033[31m"
)

// progressStr shows and moves a bar '...' during processing. It's used
// during the determination of the directories and files that need to be
// synched
func progressStr(act string, interval time.Duration) chan<- struct{} {
	// create stop channel for progress string
	stop := make(chan struct{})

	go func() {
		var (
			ticker = time.NewTicker(interval * time.Millisecond)
			bar    = "   ...  "
			i      = 5
		)

		fmt.Println()

		for {
			select {
			case <-ticker.C:
				fmt.Printf("\r%s %s ", act, bar[i:i+3])
				if i--; i < 0 {
					i = 5
				}
			case <-stop:
				// stop ticker ...
				ticker.Stop()
				// and return
				return
			}
		}
	}()

	return stop
}

// progressTable prints diverse data about the progress on stdout
func progressTable(done int, total int, elapsed time.Duration, numErr int, init bool, mode string) {
	var (
		formatStr string
		objStr    string
	)

	formatStrMeta := "\r%%6s %s%%8s %%9s %s%%6s%s\n"
	formatStrHead := fmt.Sprintf(formatStrMeta, "", "", "")
	formatStrLine := fmt.Sprintf(formatStrMeta, cBold, cNormal, "")
	formatStrLineErr := fmt.Sprintf(formatStrMeta, cBold, cFGRed, cFGDefault+cNormal)

	if mode == progModeDirs {
		objStr = "#DIRS"
	} else {
		objStr = "#FILES"
	}

	if init {
		fmt.Printf(formatStrHead, "", "DONE", "REMAINING", "ERRORS")
		fmt.Printf(formatStrLine, objStr, "0", "0", "0")
		fmt.Printf(formatStrLine, "TIME", "00:00:00", "00:00:00", "")
		return
	}

	var remaining time.Duration
	if done > 0 {
		remaining = time.Duration(int64(elapsed) / int64(done) * int64(total-done))
	}

	fmt.Printf("\u001B[2A")
	if numErr == 0 {
		formatStr = formatStrLine
	} else {
		formatStr = formatStrLineErr
	}
	fmt.Printf(formatStr, objStr, strconv.Itoa(done), strconv.Itoa(total-done), strconv.Itoa(numErr))
	fmt.Printf(formatStrLine, "TIME", lhlp.DurToHms(elapsed, "%02d:%02d:%02d"), lhlp.DurToHms(remaining, "%02d:%02d:%02d"), "")
}
