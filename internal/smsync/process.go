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

	"github.com/mipimipi/go-lhlp/file"
	worker "github.com/mipimipi/go-worker"
	"github.com/ricochet2200/go-disk-usage/du"
	log "github.com/sirupsen/logrus"
)

// cleanUp removes temporary files and directories
func cleanUp(cfg *Config, wg *sync.WaitGroup, done chan<- struct{}, errors chan<- error) {
	// wait for processing of dirs and files to be done
	wg.Wait()

	log.Debug("smsync.cleanUp: BEGIN")
	defer log.Debug("smsync.cleanUp: END")

	defer func() { done <- struct{}{} }()

	// remove log file if it's empty
	file.RemoveEmpty(filepath.Join(cfg.TrgDir, LogFile))
	log.Debug("Removed log files (at least tried to do that)")

	// update config file
	if err := cfg.setProcEnd(); err != nil {
		errors <- err
		return
	}
}

// Process is the main "backend" function to control the conversion.
// Essentially, it gets the list of directories and files to be processed and
// returns a Tracking instances, an error channel and a done channel
func Process(cfg *Config, dirs *file.InfoSlice, files *file.InfoSlice, init bool) (*Tracking, <-chan error, <-chan struct{}, error) {
	log.Debug("smsync.Process: BEGIN")
	defer log.Debug("smsync.Process: END")

	var (
		trck   = newTrck(files, du.NewDiskUsage(cfg.TrgDir).Available()) // tracking
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

	// delete all entries of the target directory if requested per cli option
	if init {
		log.Info("Delete all entries of the target directory per cli option")
		if err := deleteTrg(cfg); err != nil {
			return nil, nil, nil, err
		}
	}

	// fork processing of directories
	wg.Add(1)
	go processDirs(cfg, dirs, &wg, errors)

	// fork processing of and files
	wg.Add(1)
	go processFiles(cfg, trck, files, &wg, errors)

	// cleaning up. cleanUp waits for processDirs and processFiles to finish
	go cleanUp(cfg, &wg, done, errors)

	return trck, errors, done, nil
}

// processDirs creates new and deletes obsolete directories
func processDirs(cfg *Config, dirs *file.InfoSlice, wg *sync.WaitGroup, errors chan<- error) {
	log.Debug("smsync.processDirs: BEGIN")
	defer log.Debug("smsync.processDirs: END")

	defer wg.Done()

	for _, d := range *dirs {
		if err := deleteObsoleteFiles(cfg, d); err != nil {
			errors <- err
		}
	}
}

// processFiles calls the conversion for all new or changed files. Files
// are processed in parallel using the package github.com/mipimipi/go-worker.
func processFiles(cfg *Config, trck *Tracking, files *file.InfoSlice, wg *sync.WaitGroup, errors chan<- error) {
	log.Debug("smsync.processFiles: BEGIN")
	defer log.Debug("smsync.processFiles: END")

	defer wg.Done()

	// nothing to do in case of empty files array
	if len(*files) == 0 {
		return
	}

	// start progress tracking and register tracking stop
	trck.start()
	defer trck.stop()

	// setup worker Go routine and get worklist and result channels
	wl, res := worker.Setup(func(i interface{}) interface{} { return convert(i.(cvInput)) }, cfg.NumWrkrs)

	// fill worklist with files and close worklist channel
	go func() {
		for _, f := range *files {
			wl <- cvInput{cfg: cfg, srcFile: f}
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
}
