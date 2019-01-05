// Copyright (C) 2018-2019 Michael Picht
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

	"github.com/mipimipi/go-lhlp/file"
	worker "github.com/mipimipi/go-worker"
	"github.com/ricochet2200/go-disk-usage/du"
	log "github.com/sirupsen/logrus"
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

// Process is the main "backend" function to control the conversion.
// Essentially, it gets the list of directories and files to be processed and
// returns a Tracking instances, an error channel and a done channel
func Process(cfg *Config, dirs, files *[]*file.Info, init bool) (*Tracking, <-chan struct{}) {
	log.Debug("smsync.Process: BEGIN")
	defer log.Debug("smsync.Process: END")

	var (
		trck = newTrck(files, du.NewDiskUsage(cfg.TrgDir).Available()) // tracking
		done = make(chan struct{})                                     // done channel
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

	// fork processing of directories
	processDirs(cfg, dirs, &wg)

	// fork processing of and files
	processFiles(cfg, trck, files, &wg)

	// cleaning up. cleanUp waits for processDirs and processFiles to finish
	go cleanUp(cfg, &wg, done)

	return trck, done
}

// processDirs creates new and deletes obsolete directories
func processDirs(cfg *Config, dirs *[]*file.Info, wg *sync.WaitGroup) {
	log.Debug("smsync.processDirs: BEGIN")
	defer log.Debug("smsync.processDirs: END")

	// nothing to do in case of empty dirs array
	if len(*dirs) == 0 {
		return
	}

	// setup worker Go routine and get worklist and result channels
	wl, res, _ := worker.Setup(func(i interface{}) interface{} { return deleteObsoleteFiles(i.(obsInput)) }, cfg.NumWrkrs)

	wg.Add(1)
	go func() {
		defer wg.Done()

		// fill worklist with directories and close worklist channel
		go func() {
			for _, d := range *dirs {
				wl <- obsInput{cfg: cfg, srcDir: d}
			}
			close(wl)
		}()

		// empty results channel
		for range res {
		}
	}()
}

// processFiles calls the conversion for all new or changed files. Files
// are processed in parallel using the package github.com/mipimipi/go-worker.
func processFiles(cfg *Config, trck *Tracking, files *[]*file.Info, wg *sync.WaitGroup) {
	log.Debug("smsync.processFiles: BEGIN")
	defer log.Debug("smsync.processFiles: END")

	// nothing to do in case of empty files array
	if len(*files) == 0 {
		return
	}

	// setup worker Go routine and get worklist and result channels
	wl, res, _ := worker.Setup(func(i interface{}) interface{} { return convert(i.(cvInput)) }, cfg.NumWrkrs)

	wg.Add(1)
	go func() {
		defer wg.Done()

		// start progress tracking and register tracking stop
		trck.start()
		defer trck.stop()

		// fill worklist with files and close worklist channel
		go func() {
			for _, f := range *files {
				wl <- cvInput{cfg: cfg, srcFile: *f}
			}
			close(wl)
		}()

		// retrieve worker results and update tracking
		for r := range res {
			trck.update(
				CvInfo{SrcFile: r.(cvOutput).srcFile,
					TrgFile: r.(cvOutput).trgFile,
					Dur:     r.(cvOutput).dur,
					Err:     r.(cvOutput).err})
		}
	}()
}
