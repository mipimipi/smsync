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
	"sync"
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

type Process struct {
	pl    *wp.Pool
	Trck  *Tracking
	wg    sync.WaitGroup
	cfg   *Config
	files *[]*file.Info
	init  bool
}

const (
	taskNameDir  = "process directory"
	taskNameFile = "convert file"
)

// Process is the main "backend" function to control the conversion.
// Essentially, it gets the list of directories and files to be processed and
// returns a Tracking instances, an error channel and a done channel
func NewProcess(cfg *Config, files *[]*file.Info, init bool) *Process {
	log.Debug("smsync.NewProcess: BEGIN")
	defer log.Debug("smsync.NewProcess: END")

	proc := new(Process)

	proc.pl = wp.NewPool(cfg.NumWrkrs)

	proc.Trck = newTrck(files, du.NewDiskUsage(cfg.TrgDir).Available()) // tracking

	proc.cfg = cfg
	proc.files = files
	proc.init = init

	return proc
}

// cleanUp removes temporary files and directories
func (proc *Process) cleanUp(wg *sync.WaitGroup) {
	defer proc.wg.Done()

	proc.pl.Wait()

	log.Debug("smsync.Process.cleanUp: BEGIN")
	defer log.Debug("smsync.Process.cleanUp: END")

	// remove log file if it's empty
	file.RemoveEmpty(filepath.Join(proc.cfg.TrgDir, LogFile))
	log.Debug("Removed log files (at least tried to do that)")

	// update config file
	proc.cfg.setProcEnd()
}

func (proc *Process) Run() {
	log.Debug("smsync.Process.Run: BEGIN")
	defer log.Debug("smsync.Process.Run: END")

	var wg sync.WaitGroup

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

	wg.Add(1)
	go func() {
		defer wg.Done()

		// start progress tracking and register tracking stop
		proc.Trck.start()
		defer proc.Trck.stop()

		// fill worklist with files and close worklist channel
		go func() {
			for _, f := range *proc.files {
				// assemble task
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
				proc.Trck.update(
					ProcInfo{SrcFile: res.Out.(file.Info),
						TrgFile: nil,
						Dur:     0,
						Err:     nil})
			case taskNameFile:
				proc.Trck.update(
					ProcInfo{SrcFile: res.Out.(procOut).srcFile,
						TrgFile: res.Out.(procOut).trgFile,
						Dur:     res.Out.(procOut).dur,
						Err:     res.Out.(procOut).err})
			default:
				log.Warningf("Task name '%s' received", res.Name)
			}
		}
	}()

	// cleaning up
	proc.wg.Add(1)
	go proc.cleanUp(&wg)
}

func (proc *Process) Stop() {
	proc.pl.Stop()
}

func (proc *Process) Wait() {
	proc.wg.Wait()
}
