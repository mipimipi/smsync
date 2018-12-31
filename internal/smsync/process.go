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

// cleanUp removes temporary files and directories
func cleanUp(cfg *Config, wg *sync.WaitGroup, done chan<- struct{}) {
	// wait for processing of dirs and files to be done
	wg.Wait()

	log.Debug("smsync.cleanUp: BEGIN")
	defer log.Debug("smsync.cleanUp: END")

	defer func() { done <- struct{}{} }()

	// remove log file if it's empty
	file.RemoveEmpty(filepath.Join(cfg.TrgDir, LogFile))
	log.Debug("Removed log files (at least tried to do that)")

	// update config file
	cfg.setProcEnd()
}

// process calls f for all files. Files are processed in parallel using the
// worker pool.
func process(cfg *Config, f func(file.Info) procOut, files *[]*file.Info, wg *sync.WaitGroup) *Tracking {
	log.Debug("smsync.process: BEGIN")
	defer log.Debug("smsync.process: END")

	// nothing to do in case of empty files array
	if len(*files) == 0 {
		return nil
	}

	trck := newTrck(files, du.NewDiskUsage(cfg.TrgDir).Available()) // tracking

	wp := wp.New(func(i interface{}) interface{} { return f(i.(file.Info)) }, cfg.NumWrkrs)

	wg.Add(1)
	go func() {
		defer wg.Done()

		// start progress tracking and register tracking stop
		trck.start()
		defer trck.stop()

		// fill worklist with files and close worklist channel
		go func() {
			for _, f := range *files {
				wp.In <- *f
			}
			close(wp.In)
		}()

		// retrieve worker results and update tracking
		for r := range wp.Out {
			trck.update(
				ProcInfo{SrcFile: r.(procOut).srcFile,
					TrgFile: r.(procOut).trgFile,
					Dur:     r.(procOut).dur,
					Err:     r.(procOut).err})
		}
	}()

	return trck
}

// Process is the main "backend" function to control the conversion.
// Essentially, it gets the list of directories and files to be processed and
// returns a Tracking instances, an error channel and a done channel
func Process(cfg *Config, dirs, files *[]*file.Info, init bool) (*Tracking, <-chan struct{}) {
	log.Debug("smsync.Process: BEGIN")
	defer log.Debug("smsync.Process: END")

	var (
		done = make(chan struct{}) // done channel
		wg   sync.WaitGroup
	)

	// if no directories and no files need to be synchec: exit
	if len(*dirs) == 0 && len(*files) == 0 {
		log.Info("Nothing to process")
		return nil, nil
	}

	// remove potentially existing error directory from last run
	if err := os.RemoveAll(errDir); err != nil {
		log.Errorf("Process: %v", err)
		return nil, nil
	}

	// delete all entries of the target directory if requested per cli option
	if init {
		log.Info("Delete all entries of the target directory per cli option")
		deleteTrg(cfg)
	}

	dof := func(srcDir file.Info) procOut {
		deleteObsoleteFiles(cfg, srcDir)
		return procOut{srcFile: srcDir,
			trgFile: nil,
			dur:     0,
			err:     nil}
	}

	cv := func(srcFile file.Info) procOut {
		return convert(cfg, srcFile)
	}

	// fork processing of directories
	process(cfg, dof, dirs, &wg)

	// fork processing of and files
	trck := process(cfg, cv, files, &wg)

	// cleaning up. cleanUp waits for processDirs and processFiles to finish
	go cleanUp(cfg, &wg, done)

	return trck, done
}
