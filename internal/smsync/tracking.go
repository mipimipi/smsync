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

package smsync

import (
	"time"

	"github.com/mipimipi/go-lhlp/file"
)

// Tracking contains attributes that are used to keep track of the progress of
// the processing
type Tracking struct {
	// number of files
	TotalNum int // total number of files
	Done     int // number of files / dirs that have been processed

	// time
	Started   time.Time     // start time of processing
	AvgDur    time.Duration // average duration of a conversion
	Dur       time.Duration // cumulated duration
	Elapsed   time.Duration // elapsed time
	Remaining time.Duration // remaining time

	// size
	TotalSize uint64 // total aggregated size of source files
	Diskspace uint64 // available space on target device
	SrcSize   uint64 // cumulated size of source files
	TrgSize   uint64 // cumulated size of target files
	Size      uint64 // estimated total target size
	Avail     int64  // estimated free diskspace

	// efficiency
	Throughput float64 // average number of conversions per minute
	Comp       float64 // average compression rate

	Errors int // number of errors

	in  chan ProcInfo // channel to update tracking
	Out chan ProcInfo // channel to send intermediate results
}

// newTrck create a Tracking instance
func newTrck(wl *[]*file.Info, space uint64) *Tracking {
	var trck Tracking

	trck.TotalNum = len(*wl)
	trck.Diskspace = space
	trck.in = make(chan ProcInfo)
	trck.Out = make(chan ProcInfo)

	for _, inf := range *wl {
		trck.TotalSize += uint64((*inf).Size())
	}

	// receive updates and forward them
	go func() {
		defer close(trck.Out)
		for pInfo := range trck.in {
			if pInfo.SrcFile != nil {
				trck.Out <- pInfo
			}
			trck.update(pInfo)
		}
	}()

	return &trck
}

// start begins progress tracking
func (trck *Tracking) start() {
	trck.Started = time.Now()
}

// update receives information about a finished conversion and updates
// tracking accordingly
func (trck *Tracking) update(pInfo ProcInfo) {
	trck.Elapsed = time.Since(trck.Started)

	if trck.Done > 0 {
		trck.Remaining = time.Duration(int64(trck.Elapsed) / int64(trck.Done) * int64(trck.TotalNum-trck.Done))
	}

	if trck.Elapsed > 0 {
		trck.Throughput = float64(trck.Done) / trck.Elapsed.Minutes()
	}

	if pInfo.SrcFile != nil && pInfo.TrgFile != nil {
		trck.Done++
	}
	if pInfo.SrcFile != nil {
		trck.SrcSize += uint64(pInfo.SrcFile.Size())
	}
	if pInfo.TrgFile != nil {
		trck.TrgSize += uint64(pInfo.TrgFile.Size())
	}
	trck.Dur += pInfo.Dur

	if trck.Done > 0 {
		trck.AvgDur = time.Duration(int(trck.Dur) / trck.Done)
	}
	if trck.SrcSize > 0 {
		trck.Comp = float64(trck.TrgSize) / float64(trck.SrcSize)
	}
	trck.Size = uint64(trck.Comp * float64(trck.TotalSize))
	trck.Avail = int64(trck.Diskspace) - int64(trck.Size)
}

// Tick updates the elapsed time
func (trck *Tracking) Tick() {
	trck.in <- ProcInfo{SrcFile: nil,
		TrgFile: nil,
		Dur:     0,
		Err:     nil}
}
