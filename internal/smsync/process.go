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

	lhlp "github.com/mipimipi/go-lhlp"
	worker "github.com/mipimipi/go-worker"
	"github.com/ricochet2200/go-disk-usage/du"
	log "github.com/sirupsen/logrus"
)

// ProcRes is the result structure for directory or file processing
type ProcRes struct {
	SrcFile string        // source file or directory
	TrgFile string        // target file or directory
	dur     time.Duration // duration of a conversion
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
	Throughput float64       // throughput (= conversion per time)
	Errors     int           // number of errors
	Dur        time.Duration // cumulated duration
	AvgDur     time.Duration // average duration per minute
	Elapsed    time.Duration // elapsed time
	Remaining  time.Duration // remaining time
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

func newProg(wl *[]lhlp.FileInfo, space uint64) *Progress {
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

func (prog *Progress) update(srcFile, trgFile lhlp.FileInfo, dur time.Duration, err error) {
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
	prog.Elapsed = time.Since(prog.Start)
	prog.Remaining = time.Duration(int64(prog.Elapsed) / int64(prog.Done) * int64(prog.TotalNum-prog.Done))
	prog.Dur += dur
	prog.AvgDur = time.Duration(int(prog.Dur) / prog.Done)
	if prog.Elapsed > 0 {
		prog.Throughput = float64(prog.Done) / prog.Elapsed.Minutes()
	}
	if err != nil {
		prog.Errors++
	}
}

// Process is the main "backend" function to control the conversion.
// Essentially, it gets the list of directories and files to be processed and
// returns corresponding handles to Progress instances. Via these instances,
// the calling UI (be it a cli or some other UI) can retrieve progress
// information
func Process(cfg *Config, dirs *[]lhlp.FileInfo, files *[]lhlp.FileInfo, init bool) (*Progress, *Progress, <-chan error, error) {
	log.Debug("smsync.Process: START")
	defer log.Debug("smsync.Process: END")

	var (
		dirProg  = newProg(dirs, 0)                                            // progress structure for directories
		fileProg = newProg(files, du.NewDiskUsage(cfg.TrgDirPath).Available()) // progress structure for files
		done     = make(chan struct{})                                         // channel processing go routine to report that it's done
		errors   = make(chan error)                                            // error channel
	)

	// if no directories and no files need to be synchec: exit
	if len(*dirs) == 0 && len(*files) == 0 {
		log.Info("Nothing to process")
		return nil, nil, nil, nil
	}

	// remove potentially existing error directory from last run
	removeErrDir()

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

	// the actual processing of directories and files
	go func() {
		// register closure of done channel
		defer close(done)

		// process directories. This is only necessary, if ...
		// - at least one directory has been changed and
		// - smsync hasn't been called in initialize mode and
		// - there was at least one sync before
		if len(*dirs) > 0 && !init && !cfg.LastSync.IsZero() {
			dirProg.kickOff()
			processDirs(cfg, dirProg, dirs)
		}

		// process files
		if len(*files) > 0 {
			fileProg.kickOff()
			processFiles(cfg, fileProg, files)
		}

		// done
		done <- struct{}{}
	}()

	// clean up
	go func() {
		// register closure of error channel
		defer close(errors)

		// wait for processing to be done
		_ = <-done

		// remove obsolete stuff
		if err := cleanUp(cfg); err != nil {
			errors <- err
			return
		}

		// update config file
		if err := cfg.setProcEnd(); err != nil {
			errors <- err
			return
		}

		errors <- nil
	}()

	return dirProg, fileProg, errors, nil
}

// processDirs creates new and deletes obsolete directories. processDirs
// returns a channel that it uses to return the processing status/result
// continuously after a directory has been processed.
func processDirs(cfg *Config, prog *Progress, dirs *[]lhlp.FileInfo) {
	log.Debug("smsync.processDirs: START")
	defer log.Debug("smsync.processDirs: END")

	defer prog.close()

	// nothing to do in case of empty directory array
	if len(*dirs) == 0 {
		return
	}

	var (
		trgDirPath string
		exists     bool
		err        error
	)

	for _, d := range *dirs {
		// assemble full path of new directory (source & target)
		trgDirPath, err = lhlp.PathRelCopy(cfg.SrcDirPath, d.Path(), cfg.TrgDirPath)
		if err != nil {
			log.Errorf("Target path cannot be assembled: %v", err)
			return
		}

		// determine if directory exists
		exists, err = lhlp.FileExists(trgDirPath)
		if err != nil {
			return
		}

		if exists {
			// if it exists: check if there are obsolete files and delete them
			if err = deleteObsoleteFiles(cfg, d); err != nil {
				return
			}
		}

		// update progress
		prog.update(d, nil, 0, err)
	}
}

// ProcessFiles calls the conversion for all new or changes files. Files
// are processed in parallel using the package github.com/mipimipi/go-worker.
// It returns a channel that it uses to return the processing status/result
// continuously after a file has been processed.
func processFiles(cfg *Config, prog *Progress, files *[]lhlp.FileInfo) {
	log.Debug("smsync.processFiles: START")
	defer log.Debug("smsync.processFiles: END")

	// nothing to do in case of empty files array
	if len(*files) == 0 {
		return
	}

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
	}

	prog.close()
}
