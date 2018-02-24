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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	lhlp "github.com/mipimipi/go-lhlp"
	worker "github.com/mipimipi/go-worker"
	log "github.com/sirupsen/logrus"
)

// deleteObsoleteFile deletes directories and files that are available in the
// destination directory tree but not in the source directory tree. It is called
// for all source directories that have been changes since the last sync
func deleteObsoleteFiles(cfg *config, srcDirPath string) {
	// assemble destination directory path
	dstDirPath, err := lhlp.PathRelCopy(cfg.srcDirPath, srcDirPath, cfg.dstDirPath)
	if err != nil {
		log.Errorf("%v", err)
		return
	}

	// open destination directory
	dstDir, err := os.Open(dstDirPath)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	// close destiantion directory (deferred)
	defer func() {
		if err = dstDir.Close(); err != nil {
			log.Errorf("%s can't be closed: %v", dstDirPath, err)
		}
	}()

	// read entries of destination directory
	dstEntrs, err := dstDir.Readdir(0)
	if err != nil {
		log.Errorf("%v", err)
		return
	}

	// loop over all entries of destination directory
	for _, dstEntr := range dstEntrs {
		if dstEntr.IsDir() {
			// if entry is a directory ...
			b, _ := lhlp.FileExists(filepath.Join(srcDirPath, dstEntr.Name()))
			if err != nil {
				panic(err.Error())
			}
			// ... and the counterpart on source side doesn't exists: ...
			if !b {
				// ... delete entry
				if err = os.Remove(filepath.Join(dstDirPath, dstEntr.Name())); err != nil {
					log.Errorf("%v", err)
					return
				}
			}
		} else // if entry is a file
		{
			// if entry is not regular: do nothing and continue loop
			if !dstEntr.Mode().IsRegular() {
				continue
			}
			// if entry is a smsync file (smsync.log or SMSYNC_CONF)
			if strings.Contains(dstEntr.Name(), logFileName) || strings.Contains(dstEntr.Name(), cfgFileName) {
				continue
			}
			// check if counterpart file on source side exists
			tr := lhlp.PathTrunk(dstEntr.Name())
			fs, err := filepath.Glob(lhlp.EscapePattern(filepath.Join(srcDirPath, tr)) + ".*")
			if err != nil {
				log.Errorf("%v", err)
				log.Debugf(filepath.Join(srcDirPath, tr+".*"))
				return
			}
			// if counterpart does not exist: ...
			if fs == nil {
				// ... delete entry
				if err = os.Remove(filepath.Join(dstDirPath, dstEntr.Name())); err != nil {
					log.Errorf("%v", err)
					return
				}
			}
		}
	}
}

// getSyncFiles determines which directory and files need to be synched
func getSyncFiles(cfg *config) (*[]*string, *[]*string) {

	// filter function needed for FindFiles
	filter := func(srcFile string) bool {
		fi, err := os.Stat(srcFile)
		if err != nil {
			return false
		}

		// check if file is relevant for smsync (i.e. its suffix is contained
		// in the symsync config). If not: Return false
		if !fi.IsDir() {
			_, ok := cfg.getTf(srcFile)
			if !ok {
				return false
			}
		}

		// check if the file has been changed since last sync. If not:
		// Return false
		if fi.ModTime().Before(cfg.lastSync) {
			if !fi.IsDir() {
				return false
			}
			// in case, srcFile is a file (and no directory), another check
			// is necessary since the modification time of downloaded music
			// files is sometimes earlier then the download time (i.e. the
			// modification time is not updated during download). That's the
			// case if an entire album is downloaded as zip file. Therefore,
			// in addition, it is checked whether the modification time of
			// directory of the file has changed since last sync. If that's
			// the case, the file is relevant for the synchronization
			fiDir, err := os.Stat(filepath.Dir(srcFile))
			if err != nil {
				return false
			}
			if fiDir.ModTime().Before(cfg.lastSync) {
				return false
			}
		}

		// if smsync has been called in add only mode, files on source side
		// are only relevant for sync, if no counterpart is existing on
		// destination side. That's check in the next if statement
		if cli.addOnly {
			// assemble destination file path
			dstFile, err := lhlp.PathRelCopy(cfg.srcDirPath, srcFile, cfg.dstDirPath)
			if err != nil {
				panic(err.Error())
			}

			// if source file is a directory, check it the counterpart on
			// destination side exists
			if fi.IsDir() {
				exists, err := lhlp.FileExists(dstFile)
				if err != nil {
					panic(err.Error())
				}
				return !exists
			}

			// otherwise (if it's a file): check if counterpart exists on
			// destination side as well
			fs, err := filepath.Glob(lhlp.EscapePattern(lhlp.PathTrunk(dstFile)) + ".*")
			if err != nil {
				log.Errorf("%v", err)
				return false
			}
			return (fs == nil)
		}

		return true
	}

	// call FindFiles with the smsync filter function to get the directories and files
	return lhlp.FindFiles([]string{cfg.srcDirPath}, filter, 20)
}

// processDirs creates new and deletes obsolets directories. processDirs
// displays the progress on the command line and returns the overall time that
// has been needed
func processDirs(cfg *config, dirs *[]*string) time.Duration {
	// nothing to do in case of empty directory array
	if len(*dirs) == 0 {
		return 0
	}

	// variables needed to display progress
	var (
		numDone int           // number of processed files
		start   = time.Now()  // start time of transformation
		elapsed time.Duration // elapsed time of transformation
	)

	// print headline
	fmt.Println("\n\033[1m\033[34m# Process directories\033[22m\033[39m")

	// print initial progress table
	progressTable(0, len(*dirs), 0, 0, true, progModeDirs)

	for _, d := range *dirs {
		// assemble full path of new directory (source & destination)
		dstDirPath, err := lhlp.PathRelCopy(cfg.srcDirPath, *d, cfg.dstDirPath)
		if err != nil {
			log.Errorf("%v", err)
			continue
		}

		// determine if directory exists
		exists, err := lhlp.FileExists(dstDirPath)
		if err != nil {
			log.Errorf("%v", err)
			continue
		}

		if exists {
			// if it exists: check if there are obsolete files and delete them
			deleteObsoleteFiles(cfg, *d)
		} else {
			// if it doesn't exist: create it
			if err = os.MkdirAll(dstDirPath, os.ModeDir|0755); err != nil {
				log.Errorf("%v", err)
			}
		}

		// increase counter
		numDone++
		// determine elapsed time
		elapsed = time.Since(start)

		// update progress table on command line
		progressTable(numDone, len(*dirs), elapsed, 0, false, progModeDirs)
	}

	return elapsed
}

// processFiles calls the transformation for all new or changes files. Files
// are processes in parallel using the package github.com/mipimipi/go-worker.
// processFiles displays the progress on the command line and returns the
// overall time that has been needed
func processFiles(cfg *config, files *[]*string) time.Duration {
	// nothing to do in case of empty files array
	if len(*files) == 0 {
		return 0
	}

	// print headline
	fmt.Println("\n\033[1m\033[34m# Transform files\033[22m\033[39m")

	// setup worker Go routine and get worklist and result channels
	wl, res := worker.Setup(func(i interface{}) interface{} { return transform(i.(tfInput)) }, cfg.numWrkrs)

	// fill worklist with files and close worklist channel
	go func() {
		for _, f := range *files {
			wl <- tfInput{cfg, *f}
		}
		close(wl)
	}()

	// print initial progress table
	progressTable(0, len(*files), 0, 0, true, progModeFiles)

	// variables needed to measure progress
	var (
		numErr  int                           // number of errors
		numDone int                           // number of transformed files
		start   = time.Now()                  // start time of transformation
		elapsed time.Duration                 // elapsed time of transformation
		ticker  = time.NewTicker(time.Second) // ticker to update progress on screen every second
		done    bool                          // indicator to leave the for loop
	)

	// retrieve worker results and ticks
	for !done {
		select {
		case <-ticker.C:
			// determine elapsed time
			elapsed = time.Since(start)
			// update progress table on command line
			progressTable(numDone, len(*files), elapsed, numErr, false, progModeFiles)
		case r, ok := <-res:
			// if all files have been transformed ...
			if !ok {
				// stop trigger
				ticker.Stop()
				// leave for loop
				done = true
				continue
			}
			// increase number of transformed files
			numDone++
			// increase number of errors
			if r.(tfOutput).err != nil {
				numErr++
			}
			// determine elapsed time
			elapsed = time.Since(start)
			// update progress table on command line
			progressTable(numDone, len(*files), elapsed, numErr, false, progModeFiles)
		}
	}

	return elapsed
}

// synchronize is the main function of smsync. It triggers the entire sync
// process:
// (1) read configuration
// (2) determine directories and files to be synched
// (3) start processing of these directories and files
func synchronize() error {
	// read configuration
	cfg, err := getCfg()
	if err != nil {
		return err
	}

	// print summary and ask user for OK
	cfg.summary()
	if !lhlp.UserOK("Start synchronization") {
		log.Infof("Synchronization not started due to user input")
		return nil
	}

	// set number of cpus to be used by smsync
	_ = runtime.GOMAXPROCS(cfg.numCpus)

	// start automatic progress string which increments every second
	stop := progressStr("Find differences (this can take a few minutes)", 1000)

	// get list of directories and files for sync
	dirs, files := getSyncFiles(cfg)

	// stop progress string
	stop <- struct{}{}
	close(stop)

	// if no directories and no files need to be synchec: exit
	if len(*dirs) == 0 && len(*files) == 0 {
		fmt.Println("Nothing to synchronize. Leaving smsync ...")
		log.Info("Nothing to synchronize")
		return nil
	}

	// print summary and ask user for OK to continue
	if !lhlp.UserOK(fmt.Sprintf("%d directories and %d files to synchronize. Continue", len(*dirs), len(*files))) {
		log.Infof("Synchronization not started due to user input")
		return nil
	}

	// process directories
	durDirs := processDirs(cfg, dirs)

	// process files
	durFiles := processFiles(cfg, files)

	// print headline
	fmt.Println("\n\033[1m\033[34m# Done :)\033[22m\033[39m")
	fmt.Printf("Processed %d directories and %d files in %s\n", len(*dirs), len(*files), lhlp.DurToHms(durDirs+durFiles, "%dh %02dmin %02ds"))

	// update last sync time in config file
	return cfg.updateLastSync()
}
