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
	"os"
	"path/filepath"
	"time"

	"github.com/mipimipi/go-lhlp/file"
	wp "github.com/mipimipi/workerpool"
	"github.com/ricochet2200/go-disk-usage/du"
	log "github.com/sirupsen/logrus"
)

type (
	// output structure of processing
	procOut struct {
		srcFile file.Info     // source file
		trgFile file.Info     // target file
		dur     time.Duration // duration of conversion
		err     error         // error (that occurred during the conversion)
	}
	// ProcInfo contains information about the conversion of a single file
	ProcInfo struct {
		SrcFile file.Info     // source file or directory
		TrgFile file.Info     // target file or directory
		Dur     time.Duration // duration of a conversion
		Err     error         // error (that occurred during processing)
	}
)

// Process contains the data to control the sync process
type Process struct {
	pl      *wp.Pool      // worker pool
	Trck    *Tracking     // progress tracking
	cfg     *Config       // smsync config
	files   *[]*file.Info // list of files that need to be synched
	init    bool          // called in init mode?
	cleanup chan struct{} // start cleanup
	done    chan struct{} // report processing to be done
	stopped bool          // processing has been stopped?
}

// constants for task names, needed for workerpool
const (
	taskNameDir  = "process directory"
	taskNameFile = "convert file"
)

// NewProcess create a new process object
func NewProcess(cfg *Config, files *[]*file.Info, init bool) *Process {
	log.Debug("smsync.NewProcess: BEGIN")
	defer log.Debug("smsync.NewProcess: END")

	proc := new(Process)

	// set up worker pool
	proc.pl = wp.NewPool(cfg.NumWrkrs)

	// set up progress tracking
	proc.Trck = newTrck(files, du.NewDiskUsage(cfg.TrgDir).Available()) // tracking

	// make channels
	proc.cleanup = make(chan struct{})
	proc.done = make(chan struct{})

	// store sync parameters
	proc.cfg = cfg
	proc.files = files
	proc.init = init

	return proc
}

// cleanUp removes temporary files and directories and updates the config file
func (proc *Process) cleanUp() {
	log.Debug("smsync.Process.cleanUp: BEGIN")
	defer log.Debug("smsync.Process.cleanUp: END")

	// wait until processing is finished
	<-proc.cleanup

	// remove log file if it's empty
	file.RemoveEmpty(filepath.Join(proc.cfg.TrgDir, LogFile))
	log.Debug("Removed log files (at least tried to do that)")

	// update config file
	if !proc.stopped {
		proc.cfg.setProcEnd()
	}

	// processing finished
	close(proc.done)
}

// Run executes the sync process and cleans up after the sync has finished
func (proc *Process) Run() {
	log.Debug("smsync.Process.Run: BEGIN")
	defer log.Debug("smsync.Process.Run: END")

	// if no files need to be synched: exit
	if len(*proc.files) == 0 {
		log.Info("Nothing to process")
		return
	}

	// remove potentially existing error directory from last run
	if err := os.RemoveAll(errDir); err != nil {
		log.Errorf("Process: %v", err)
		return
	}

	// delete all entries of the target directory if requested per cli option
	if proc.init {
		log.Info("Delete all entries of the target directory per cli option")
		deleteTrg(proc.cfg)
	}

	go func() {
		// trigger cleanup
		defer close(proc.cleanup)

		// start progress tracking and register tracking stop
		proc.Trck.start()

		// fill worklist with files and close worklist channel
		go func() {
			for _, f := range *proc.files {
				// send task to the worker pool, distinguishing between
				// directories and files. Files need to be converted, for
				// directories, obsolete files might need to be deleted
				if (*f).IsDir() {
					proc.pl.In <- wp.Task{
						Name: taskNameDir,
						F: func(i interface{}) interface{} {
							deleteObsoleteFiles(proc.cfg, i.(file.Info))
							return i.(file.Info)
						},
						In: *f}
				} else {
					proc.pl.In <- wp.Task{
						Name: taskNameFile,
						F: func(i interface{}) interface{} {
							cvOut := convert(proc.cfg, i.(file.Info))
							return procOut{srcFile: i.(file.Info),
								trgFile: cvOut.trgFile,
								dur:     cvOut.dur,
								err:     cvOut.err}
						},
						In: *f}
				}
			}
			close(proc.pl.In)
		}()

		// retrieve worker results and update tracking
		for res := range proc.pl.Out {
			switch res.Name {
			case taskNameDir:
				proc.Trck.in <- ProcInfo{SrcFile: res.Out.(file.Info),
					TrgFile: nil,
					Dur:     0,
					Err:     nil}
			case taskNameFile:
				proc.Trck.in <- ProcInfo{SrcFile: res.Out.(procOut).srcFile,
					TrgFile: res.Out.(procOut).trgFile,
					Dur:     res.Out.(procOut).dur,
					Err:     res.Out.(procOut).err}
			default:
				log.Warningf("Task name '%s' received", res.Name)
			}
		}
		close(proc.Trck.in)
	}()

	// cleaning up
	go proc.cleanUp()
}

// Stop stops the sync process
func (proc *Process) Stop() {
	proc.pl.Stop()
	proc.stopped = true
}

// Wait waits for the sync process to be finished
func (proc *Process) Wait() {
	<-proc.done
}
