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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mipimipi/go-lhlp/file"
	worker "github.com/mipimipi/go-worker"
	"github.com/ricochet2200/go-disk-usage/du"
	log "github.com/sirupsen/logrus"
)

// ProcRes is the result structure for directory or file processing
type ProcRes struct {
	SrcFile file.Info     // source file or directory
	TrgFile file.Info     // target file or directory
	Dur     time.Duration // duration of a conversion
	Err     error         // error (that occurred during processing)
}

// Progress contains attributes that are used to communicate the progress of the
// conversion
type Progress struct {
	Start      time.Time     // start time of processing
	Done       int           // number of files / dirs that have been processed
	TotalNum   int           // total number of files / dirs
	TotalSize  uint64        // total aggregated size of source files
	SrcSize    uint64        // cumulated size of source files
	TrgSize    uint64        // cumulated size of target files
	Diskspace  uint64        // available space on target device
	Avail      int64         // estimated free diskspace
	Size       uint64        // estimated target size
	Comp       float64       // average compression
	throughput float64       // throughput (= conversion per time)
	Errors     int           // number of errors
	Dur        time.Duration // cumulated duration
	AvgDur     time.Duration // average duration per minute
	elapsed    time.Duration // elapsed time
	Res        chan ProcRes  // channel to report intermediate results
}

func (prog *Progress) close() {
	close(prog.Res)
}

func (prog *Progress) kickOff() {
	log.Debug("smsync.Progress.kickOff: START")
	defer log.Debug("smsync.Progress.kickOff: END")

	prog.Start = time.Now()
}

func newProg(wl *file.InfoSlice, space uint64) *Progress {
	log.Debug("smsync.newProg: START")
	defer log.Debug("smsync.newProg: END")

	var prog Progress

	prog.TotalNum = len(*wl)
	prog.Diskspace = space
	prog.Res = make(chan ProcRes)

	for _, fi := range *wl {
		prog.TotalSize += uint64(fi.Size())
	}

	return &prog
}

// Elapsed calculates the elapsed time since prog.Start
func (prog *Progress) Elapsed() time.Duration {
	prog.elapsed = time.Since(prog.Start)
	return prog.elapsed
}

// Remaining calculates the estimated remaining processing time
func (prog *Progress) Remaining() time.Duration {
	if prog.Done == 0 {
		return 0
	}
	return time.Duration(int64(prog.elapsed) / int64(prog.Done) * int64(prog.TotalNum-prog.Done))
}

// Throughput average number of conversions per minute
func (prog *Progress) Throughput() float64 {
	if prog.elapsed == 0 {
		return 0
	}
	return float64(prog.Done) / prog.elapsed.Minutes()
}

func (prog *Progress) update(srcFile, trgFile file.Info, dur time.Duration, err error) {
	prog.Done++
	if srcFile != nil {
		prog.SrcSize += uint64(srcFile.Size())
	}
	if trgFile != nil {
		prog.TrgSize += uint64(trgFile.Size())
	}
	prog.Comp = float64(prog.TrgSize) / float64(prog.SrcSize)
	prog.Size = uint64(prog.Comp * float64(prog.TotalSize))
	prog.Avail = int64(prog.Diskspace) - int64(prog.Size)
	prog.Dur += dur
	prog.AvgDur = time.Duration(int(prog.Dur) / prog.Done)
	prog.elapsed = time.Since(prog.Start)
	if err != nil {
		prog.Errors++
	}

	// send conversion information to whoever is interested
	prog.Res <- ProcRes{SrcFile: srcFile,
		TrgFile: trgFile,
		Dur:     dur,
		Err:     err}
}

// Process is the main "backend" function to control the conversion.
// Essentially, it gets the list of directories and files to be processed and
// returns corresponding handles to Progress instances. Via these instances,
// the calling UI (be it a cli or some other UI) can retrieve progress
// information
func Process(cfg *Config, dirs *file.InfoSlice, files *file.InfoSlice, init bool) (*Progress, <-chan error, <-chan struct{}, error) {
	log.Debug("smsync.Process: START")
	defer log.Debug("smsync.Process: END")

	var (
		prog   = newProg(files, du.NewDiskUsage(cfg.TrgDir).Available()) // progress structure for files
		cvDone = make(chan struct{}, 2)                                  // channel for processing go routine to report that it's done
		errors = make(chan error)                                        // error channel
		done   = make(chan struct{})                                     // done channel
		wg     sync.WaitGroup
	)

	// if no directories and no files need to be synchec: exit
	if len(*dirs) == 0 && len(*files) == 0 {
		log.Info("Nothing to process")
		return nil, nil, nil, nil
	}

	// remove potentially existing error directory from last run
	if err := os.RemoveAll(errDir); err != nil {
		log.Errorf("Couldn't delete error directory: %v", err)
		return nil, nil, nil, fmt.Errorf("Couldn't delete error directory: %v", err)
	}

	// set processing status to "work in progress" in smsync.yaml
	if err := cfg.setProcStatWIP(); err != nil {
		return nil, nil, nil, err
	}

	// delete all entries of the target directory if requested per cli option
	if init {
		log.Info("Delete all entries of the target directory per cli option")
		if err := deleteTrg(cfg); err != nil {
			return nil, nil, nil, err
		}
	}

	// fork processing of directories
	go processDirs(cfg, dirs, cvDone, errors)

	// fork processing of and files
	go processFiles(cfg, prog, files, cvDone, errors)

	// clean up
	wg.Add(1)
	go cleanUp(cfg, cvDone, &wg, errors)

	// wait cleanUp is finisheduntil parallel sub processes are finished
	wg.Wait()

	return prog, errors, done, nil
}

// processDirs creates new and deletes obsolete directories. processDirs
// returns a channel that it uses to return the processing status/result
// continuously after a directory has been processed.
func processDirs(cfg *Config, dirs *file.InfoSlice, done chan<- struct{}, errors chan<- error) {
	log.Debug("smsync.processDirs: START")
	defer log.Debug("smsync.processDirs: END")

	defer func() { done <- struct{}{} }()

	for _, d := range *dirs {

		if err := deleteObsoleteFiles(cfg, d); err != nil {
			errors <- err
		}
	}
}

// ProcessFiles calls the conversion for all new or changes files. Files
// are processed in parallel using the package github.com/mipimipi/go-worker.
// It returns a channel that it uses to return the processing status/result
// continuously after a file has been processed.
func processFiles(cfg *Config, prog *Progress, files *file.InfoSlice, done chan<- struct{}, errors chan<- error) {
	log.Debug("smsync.processFiles: START")
	defer log.Debug("smsync.processFiles: END")

	defer func() { done <- struct{}{} }()

	// nothing to do in case of empty files array
	if len(*files) == 0 {
		return
	}

	// start  progress tracking
	prog.kickOff()
	defer prog.close()

	// setup worker Go routine and get worklist and result channels
	wl, res := worker.Setup(func(i interface{}) interface{} { return convert(i.(cvInput)) }, cfg.NumWrkrs)

	// fill worklist with files and close worklist channel
	go func() {
		for _, f := range *files {
			wl <- cvInput{cfg: cfg, srcFile: f}
		}
		close(wl)
	}()

	// retrieve worker results
	for r := range res {
		// update progress
		prog.update(r.(cvOutput).srcFile, r.(cvOutput).trgFile, r.(cvOutput).dur, r.(cvOutput).err)

		if r.(cvOutput).err != nil {
			errors <- r.(cvOutput).err
		}
	}
}

// cleanUp removes temporary files and directories from smsync that are
// obsolete
func cleanUp(cfg *Config, cvDone <-chan struct{}, wg *sync.WaitGroup, errors chan<- error) {
	log.Debug("smsync.cleanUp: START")
	defer log.Debug("smsync.cleanUp: END")

	defer wg.Done()

	// wait for conversion to be done
	_ = <-cvDone
	_ = <-cvDone

	// remove log file if it's empty
	file.RemoveEmpty(filepath.Join(cfg.TrgDir, LogFile))

	// update config file
	if err := cfg.setProcEnd(); err != nil {
		errors <- err
		return
	}
}
