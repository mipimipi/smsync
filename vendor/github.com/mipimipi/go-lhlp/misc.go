// Copyright (C) 2018 Michael Picht
//
// This file is part of go-lhlp (Go's little helper).
//
// go-lhlp is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-lhlp is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-lhlp. If not, see <http://www.gnu.org/licenses/>.

// Package lhlp contains practical and handy functions that are useful in many
// Go projects, but which are not part of the standards Go libraries.
package lhlp

import (
	"fmt"
	"reflect"
	"time"
)

// Contains checks if the array a contains the element e.
// inspired by: https://stackoverflow.com/questions/10485743/contains-method-for-a-slice
func Contains(a interface{}, e interface{}) bool {
	arr := reflect.ValueOf(a)

	if arr.Kind() == reflect.Slice {
		for i := 0; i < arr.Len(); i++ {
			// XXX - panics if slice element points to an unexported struct field
			// see https://golang.org/pkg/reflect/#Value.Interface
			if arr.Index(i).Interface() == e {
				return true
			}
		}
	}

	return false
}

// ProgressStr shows and moves a bar '...' on the command line. It can be used
// to show that an activity is ongoing. The parameter 'interval' steers the
// refresh rate (in milli seconds). The text in 'msg' is displayed in form of
// '...'. The progress bar is stopped by sending an empty struct to the
// returned channel:
//	 chan <- struct{}{}
//	 close(chan)
func ProgressStr(msg string, interval time.Duration) (chan<- struct{}, <-chan struct{}) {
	// create channel to receive stop signal
	stop := make(chan struct{})

	// create channel to send stop confirmation
	confirm := make(chan struct{})

	go func() {
		var (
			ticker  = time.NewTicker(interval * time.Millisecond)
			bar     = "   ...  "
			i       = 5
			isFirst = true
			ticked  = false
		)

		for {
			select {
			case <-ticker.C:
				// at the very first tick, the output switches to the next row.
				// At all subsequent ticks, the output is printed into that
				// same row.
				if isFirst {
					fmt.Println()
					isFirst = false
				}
				// print message and progress indicator
				fmt.Printf("\r%s %s ", msg, bar[i:i+3])
				// increase progress indicator counter for next tick
				if i--; i < 0 {
					i = 5
				}
				// ticker has ticked: set flag accordingly
				ticked = true
			case <-stop:
				// stop ticker ...
				ticker.Stop()
				// if the ticker had displayed at least once, move to next row
				if ticked {
					fmt.Println()
				}
				// send stop confirmation
				confirm <- struct{}{}
				close(confirm)
				// and return
				return
			}
		}
	}()

	return stop, confirm
}

// UserOK print the message s followed by " (Y/n)?" on stdout and askes the
// user to press either Y (to continue) or n (to stop). Y is treated as
// default. I.e. if the user only presses return, that's interpreted as if
// he has pressed Y.
func UserOK(s string) bool {
	var input string

	for {
		fmt.Printf("\r%s (Y/n)? ", s)
		if _, err := fmt.Scanln(&input); err != nil {
			if err.Error() != "unexpected newline" {
				return false
			}
			input = "Y"
		}
		switch {
		case input == "Y":
			return true
		case input == "n":
			return false
		}
		fmt.Println()
	}
}
