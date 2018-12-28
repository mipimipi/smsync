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
	log "github.com/sirupsen/logrus"
)

// CvInfo contains information about the conversion of a single file
type CvInfo struct {
	SrcFile file.Info     // source file or directory
	TrgFile file.Info     // target file or directory
	Dur     time.Duration // duration of a conversion
	Err     error         // error (that occurred during processing)
}

// Tracking contains attributes that are used to keep track of the progress of
// the processing
type Tracking struct {
	started   time.Time     // start time of processing
	totalNum  int           // total number of files / dirs
	totalSize uint64        // total aggregated size of source files
	diskspace uint64        // available space on target device
	done      int           // number of files / dirs that have been processed
	srcSize   uint64        // cumulated size of source files
	trgSize   uint64        // cumulated size of target files
	errors    int           // number of errors
	dur       time.Duration // cumulated duration
	CvInfo    chan CvInfo   // channel to report intermediate results
}

// Status contains attributes that are used to communicate the progress of the
// processing
type Status struct {
	Todo       int           // number of files that still have to be processed
	Elapsed    time.Duration // elapsed time
	Remaining  time.Duration // remaining time
	Throughput float64       // average number of conversions per minute
	Size       uint64        // estimated total target size
	Avail      int64         // estimated free diskspace
	Comp       float64       // average compression rate
	AvgDur     time.Duration // average duration of a conversion
	Errors     int           // number of errors
}

// newTrck create a Tracking instance
func newTrck(wl *[]*file.Info, space uint64) *Tracking {
	var trck Tracking

	trck.totalNum = len(*wl)
	trck.diskspace = space
	trck.CvInfo = make(chan CvInfo)

	for _, inf := range *wl {
		trck.totalSize += uint64((*inf).Size())
	}

	return &trck
}

// start begins progress tracking
func (trck *Tracking) start() {
	trck.started = time.Now()
}

// stop ends progress tracking
func (trck *Tracking) stop() {
	log.Debug("smsync.Tracking.stop: BEGIN")
	defer log.Debug("smsync.Tracking.stop: END")
	close(trck.CvInfo)
}

// Status calculates the current processing status based on the attributes of
// Tracking
func (trck *Tracking) Status() *Status {
	var status Status

	status.Todo = trck.totalNum - trck.done
	status.Elapsed = time.Since(trck.started)
	if trck.done > 0 {
		status.Remaining = time.Duration(int64(status.Elapsed) / int64(trck.done) * int64(trck.totalNum-trck.done))
		status.AvgDur = time.Duration(int(trck.dur) / trck.done)
	}
	if status.Elapsed > 0 {
		status.Throughput = float64(trck.done) / status.Elapsed.Minutes()
	}
	if trck.srcSize > 0 {
		status.Comp = float64(trck.trgSize) / float64(trck.srcSize)
	}
	status.Size = uint64(status.Comp * float64(trck.totalSize))
	status.Avail = int64(trck.diskspace) - int64(status.Size)
	status.Errors = trck.errors

	return &status
}

// update receives information about a finished conversion and updates
// tracking accordingly
func (trck *Tracking) update(cvInfo CvInfo) {
	trck.done++
	if cvInfo.SrcFile != nil {
		trck.srcSize += uint64(cvInfo.SrcFile.Size())
	}
	if cvInfo.TrgFile != nil {
		trck.trgSize += uint64(cvInfo.TrgFile.Size())
	}
	trck.dur += cvInfo.Dur

	// send conversion information to whoever is interested
	trck.CvInfo <- cvInfo
}
