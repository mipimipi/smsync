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
	"sync"
	"time"

	"github.com/mipimipi/go-lhlp/file"
	log "github.com/sirupsen/logrus"
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

	PInfo chan ProcInfo // channel to report intermediate results

	mu sync.Mutex // sync updates
}

// newTrck create a Tracking instance
func newTrck(wl *[]*file.Info, space uint64) *Tracking {
	var trck Tracking

	trck.TotalNum = len(*wl)
	trck.Diskspace = space
	trck.PInfo = make(chan ProcInfo)

	for _, inf := range *wl {
		trck.TotalSize += uint64((*inf).Size())
	}

	return &trck
}

// start begins progress tracking
func (trck *Tracking) start() {
	trck.Started = time.Now()
}

// stop ends progress tracking
func (trck *Tracking) stop() {
	log.Debug("smsync.Tracking.stop: BEGIN")
	defer log.Debug("smsync.Tracking.stop: END")

	close(trck.PInfo)

	trck.UpdElapsed()
}

// update receives information about a finished conversion and updates
// tracking accordingly
func (trck *Tracking) update(pInfo ProcInfo) {
	// send conversion information to whoever is interested
	trck.PInfo <- pInfo

	trck.mu.Lock()

	trck.Done++
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

	trck.mu.Unlock()
}

// UpdElapsed updates the elapsed time and calculated depending data
func (trck *Tracking) UpdElapsed() {
	trck.mu.Lock()

	trck.Elapsed = time.Since(trck.Started)

	if trck.Done > 0 {
		trck.Remaining = time.Duration(int64(trck.Elapsed) / int64(trck.Done) * int64(trck.TotalNum-trck.Done))
	}

	if trck.Elapsed > 0 {
		trck.Throughput = float64(trck.Done) / trck.Elapsed.Minutes()
	}

	trck.mu.Unlock()
}
