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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	lhlp "github.com/mipimipi/go-lhlp"
	worker "github.com/mipimipi/go-worker"
	log "github.com/sirupsen/logrus"
)

// errDir is the directory that stores error logs from conversion
const errDir = "smsync.err"

// ProcRes is the result structure for directory or file processing
type ProcRes struct {
	SrcFile string // source file or directory
	TrgFile string // target file or directory
	Err     error  // error (that occurred during processing)
}

// CleanUp removes temporary files and directories from smsync that are
// obsolete
func CleanUp(cfg *Config) error {
	var (
		b       bool
		err     error
		logFile = filepath.Join(cfg.TrgDirPath, logFileName)
	)

	// remove log file if it's empty
	if b, err = lhlp.FileIsEmpty(logFile); err != nil {
		return err
	}
	if b {
		if err = os.Remove(logFile); err != nil {
			log.Errorf("Cannot remove '%s': %v", logFile, err)
			return err
		}
	}

	return nil
}

// deleteObsoleteFiles deletes directories and files that are available in the
// target directory tree but not in the source directory tree. It is called
// for all source directories that have been changes since the last sync
func deleteObsoleteFiles(cfg *Config, srcDirPath string) error {
	// assemble target directory path
	trgDirPath, err := lhlp.PathRelCopy(cfg.SrcDirPath, srcDirPath, cfg.TrgDirPath)
	if err != nil {
		log.Errorf("Target path cannot be assembled: %v", err)
		return err
	}

	// read entries of target directory
	trgEntrs, err := ioutil.ReadDir(trgDirPath)
	if err != nil {
		log.Errorf("Cannot read directory '%s': %v", trgDirPath, err)
		return err
	}

	// loop over all entries of target directory
	for _, trgEntr := range trgEntrs {
		if trgEntr.IsDir() {
			fmt.Println("deleteObsoleteFiles: ", trgEntr.Name())

			// if entry is a directory ...
			b, _ := lhlp.FileExists(filepath.Join(srcDirPath, trgEntr.Name()))
			if err != nil {
				log.Errorf("%v", err)
			}
			// ... and the counterpart on source side doesn't exists: ...
			if !b {
				// ... delete entry
				if err = os.Remove(filepath.Join(trgDirPath, trgEntr.Name())); err != nil {
					log.Errorf("Cannot remove '%s': %v", filepath.Join(trgDirPath, trgEntr.Name()), err)
					return err
				}
			}
		} else // if entry is a file
		{
			// if entry is not regular: do nothing and continue loop
			if !trgEntr.Mode().IsRegular() {
				continue
			}
			// exclude smsync files (smsync.log or smsync.yaml) from deletion logic
			if strings.Contains(trgEntr.Name(), logFileName) || strings.Contains(trgEntr.Name(), cfgFileName) {
				continue
			}
			// check if counterpart file on source side exists
			tr := lhlp.PathTrunk(trgEntr.Name())
			fs, err := filepath.Glob(lhlp.EscapePattern(filepath.Join(srcDirPath, tr)) + ".*")
			if err != nil {
				log.Errorf("Error from Glob('%s'): %v", lhlp.EscapePattern(filepath.Join(srcDirPath, tr))+".*", err)
				return err
			}
			// if counterpart does not exist: ...
			if fs == nil {
				// ... delete entry
				if err = os.Remove(filepath.Join(trgDirPath, trgEntr.Name())); err != nil {
					log.Errorf("Cannot remove '%s': %v", filepath.Join(trgDirPath, trgEntr.Name()), err)
					return err
				}
			}
		}
	}

	return nil
}

// DeleteTrg deletes all entries of the target directory
func DeleteTrg(cfg *Config) error {
	// open target directory
	trgDir, err := os.Open(cfg.TrgDirPath)
	if err != nil {
		log.Errorf("Cannot open '%s': %v", cfg.TrgDirPath, err)
		return err
	}
	// close target directory (deferred)
	defer func() {
		if err = trgDir.Close(); err != nil {
			log.Errorf("%s can't be closed: %v", cfg.TrgDirPath, err)
		}
	}()
	// read entries of target directory
	trgEntrs, err := trgDir.Readdir(0)
	if err != nil {
		log.Errorf("Cannot read directory '%s': %v", trgDir.Name(), err)
		return err
	}

	// loop over all entries of target directory
	for _, trgEntr := range trgEntrs {
		// don't delete smsync files (smsync.log or SMSYNC.yaml)
		if !trgEntr.IsDir() && (strings.Contains(trgEntr.Name(), logFileName) || strings.Contains(trgEntr.Name(), cfgFileName)) {
			continue
		}
		// delete entry
		if err = os.RemoveAll(filepath.Join(cfg.TrgDirPath, trgEntr.Name())); err != nil {
			log.Errorf("Cannot remove '%s': %v", filepath.Join(cfg.TrgDirPath, trgEntr.Name()), err)
			return err
		}
	}

	return nil
}

// GetSyncFiles determines which directories and files need to be synched
func GetSyncFiles(cfg *Config, init bool) (*[]*string, *[]*string) {
	// filter function needed for FindFiles
	filter := func(srcFile string) (bool, bool) {
		fi, err := os.Stat(srcFile)
		if err != nil {
			log.Errorf("Error from os.Stat('%s'): %v", srcFile, err)
			return false, true
		}

		// check if file is relevant for smsync (i.e. its suffix is contained
		// in the symsync config). If not: Return false
		if !fi.IsDir() {
			_, ok := cfg.getCv(srcFile)
			if !ok {
				return false, false
			}
		}

		// check if the directory needs to be excluded
		if fi.IsDir() && lhlp.Contains(cfg.Excludes, srcFile) {
			return false, false
		}

		// check if the file/directory has been changed since last sync.
		// If not: Return false
		if fi.ModTime().Before(cfg.LastSync) {
			if fi.IsDir() {
				return false, true
			}
			// in case, srcFile is a file (and no directory), another check
			// is necessary since the modification time of downloaded music
			// files is sometimes earlier then the download time (i.e. the
			// modification time is not updated during download). That's the
			// case if an entire album is downloaded as zip file, for instance.
			// Therefore, in addition, it is checked whether the modification
			// time of the directory of the file has changed since last sync.
			// If that's the case, the file is relevant for the synchronization.
			fiDir, err := os.Stat(filepath.Dir(srcFile))
			if err != nil {
				log.Errorf("Error from os.Stat('%s'): %v", filepath.Dir(srcFile), err)
				return false, false
			}
			if fiDir.ModTime().Before(cfg.LastSync) {
				return false, false
			}
		}

		if !fi.IsDir() {
			_, ok := cfg.getCv(srcFile)
			if !ok {
				return false, false
			}
		}

		// if the last call smsync has been interrupted ('work in progress',
		// WIP) and command line option 'initialize' hasn't been set, files on
		// source side are only relevant for sync, if no counterpart is
		// existing on target side. That's checked in the next if statement
		if cfg.WIP && !init {
			// assemble target file path
			trgFile, err := lhlp.PathRelCopy(cfg.SrcDirPath, srcFile, cfg.TrgDirPath)
			if err != nil {
				log.Errorf("Target path cannot be assembled: %v", err)
				return false, true
			}

			// if source file is a directory, check it the counterpart on
			// target side exists
			if fi.IsDir() {
				var exists bool
				exists, err = lhlp.FileExists(trgFile)
				if err != nil {
					log.Errorf("%v", err)
					return false, true
				}
				return !exists, true
			}

			// otherwise (if it's a file): check if counterpart exists on
			// target side as well
			fs, err := filepath.Glob(lhlp.EscapePattern(lhlp.PathTrunk(trgFile)) + ".*")
			if err != nil {
				log.Errorf("Error from Glob('%s'): %v", lhlp.EscapePattern(lhlp.PathTrunk(trgFile))+".*", err)
				return false, false
			}
			return false, (fs == nil)
		}

		return true, true
	}

	// call FindFiles with the smsync filter function to get the directories and files
	return lhlp.FindFiles([]string{cfg.SrcDirPath}, filter, 20)
}

// ProcessDirs creates new and deletes obsolete directories. processDirs
// returns a channel that it uses to return the processing status/result
// continuously after a directory has been processed.
func ProcessDirs(cfg *Config, dirs *[]*string) <-chan ProcRes {
	var procRes = make(chan ProcRes)

	// nothing to do in case of empty directory array
	if len(*dirs) == 0 {
		return nil
	}

	go func() {
		var (
			trgDirPath string
			exists     bool
			err        error
		)

		for _, d := range *dirs {
			// assemble full path of new directory (source & target)
			trgDirPath, err = lhlp.PathRelCopy(cfg.SrcDirPath, *d, cfg.TrgDirPath)
			if err != nil {
				log.Errorf("Target path cannot be assembled: %v", err)
				return
			}

			// determine if directory exists
			exists, err = lhlp.FileExists(trgDirPath)
			if err != nil {
				log.Errorf("%v", err)
				return
			}

			if exists {
				// if it exists: check if there are obsolete files and delete them
				if err = deleteObsoleteFiles(cfg, *d); err != nil {
					return
				}
			}

			procRes <- ProcRes{SrcFile: *d, TrgFile: trgDirPath, Err: err}
		}

		close(procRes)
	}()

	return procRes
}

// ProcessFiles calls the conversion for all new or changes files. Files
// are processed in parallel using the package github.com/mipimipi/go-worker.
// It returns a channel that it uses to return the processing status/result
// continuously after a file has been processed.
func ProcessFiles(cfg *Config, files *[]*string) <-chan ProcRes {
	// variables needed to measure progress
	var procRes = make(chan ProcRes)

	// nothing to do in case of empty files array
	if len(*files) == 0 {
		return nil
	}

	// setup worker Go routine and get worklist and result channels
	wl, res := worker.Setup(func(i interface{}) interface{} { return convert(i.(cvInput)) }, cfg.NumWrkrs)

	// fill worklist with files and close worklist channel
	go func() {
		for _, f := range *files {
			wl <- cvInput{cfg, *f}
		}
		close(wl)
	}()

	// retrieve worker results
	go func() {
		for r := range res {
			// send current progress
			procRes <- ProcRes{SrcFile: r.(cvOutput).srcFile, TrgFile: r.(cvOutput).trgFile, Err: r.(cvOutput).err}
		}

		close(procRes)
	}()

	return procRes
}

// RemoveErrDir deletes the error directory
func RemoveErrDir() error {
	return os.RemoveAll(filepath.Join(".", errDir))
}
