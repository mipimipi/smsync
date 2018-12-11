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
	"strings"
	"time"

	lhlp "github.com/mipimipi/go-lhlp"
	worker "github.com/mipimipi/go-worker"
	log "github.com/sirupsen/logrus"
)

// Prog contains attributes that are used to communicate the progress of the
// conversion between (Go) routins
type Prog struct {
	Done    int           // number of files / dirs that have been processed
	Total   int           // total number of files / dirs
	Errors  int           // number of errors
	Elapsed time.Duration // elapsed time
}

// Remaining calculates the remaining time of a conversion
func (prog *Prog) Remaining() time.Duration {
	if prog.Done > 0 {
		return time.Duration(int64(prog.Elapsed) / int64(prog.Done) * int64(prog.Total-prog.Done))
	}
	return 0
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

	// open target directory
	trgDir, err := os.Open(trgDirPath)
	if err != nil {
		log.Errorf("Cannot open '%s': %v", trgDirPath, err)
		return err
	}
	// close target directory (deferred)
	defer func() {
		if err = trgDir.Close(); err != nil {
			log.Errorf("%s can't be closed: %v", trgDirPath, err)
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
		if trgEntr.IsDir() {
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
			// if entry is a smsync file (smsync.log or SMSYNC.CONF): do nothing and continue loop
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

// GetSyncFiles determines which directories and files need to be synched
func GetSyncFiles(cfg *Config) (*[]*string, *[]*string) {

	// filter function needed for FindFiles
	filter := func(srcFile string) bool {
		fi, err := os.Stat(srcFile)
		if err != nil {
			log.Errorf("Error from os.Stat('%s'): %v", srcFile, err)
			return false
		}

		// check if file is relevant for smsync (i.e. its suffix is contained
		// in the symsync config). If not: Return false
		if !fi.IsDir() {
			_, ok := cfg.getCv(srcFile)
			if !ok {
				return false
			}
		}

		// check if the file/directory has been changed since last sync.
		// If not: Return false
		if fi.ModTime().Before(cfg.LastSync) {
			if fi.IsDir() {
				return false
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
				return false
			}
			if fiDir.ModTime().Before(cfg.LastSync) {
				return false
			}
		}
		/*
			// if smsync has been called in add-only mode, files on source side
			// are only relevant for sync, if no counterpart is existing on
			// target side. That's checked in the next if statement
			if cli.addOnly {
				// assemble target file path
				trgFile, err := lhlp.PathRelCopy(cfg.srcDirPath, srcFile, cfg.trgDirPath)
				if err != nil {
					log.Errorf("Target path cannot be assembled: %v", err)
					return false
				}

				// if source file is a directory, check it the counterpart on
				// target side exists
				if fi.IsDir() {
					var exists bool
					exists, err = lhlp.FileExists(trgFile)
					if err != nil {
						log.Errorf("%v", err)
						return false
					}
					return !exists
				}

				// otherwise (if it's a file): check if counterpart exists on
				// target side as well
				fs, err := filepath.Glob(lhlp.EscapePattern(lhlp.PathTrunk(trgFile)) + ".*")
				if err != nil {
					log.Errorf("Error from Glob('%s'): %v", lhlp.EscapePattern(lhlp.PathTrunk(trgFile))+".*", err)
					return false
				}
				return (fs == nil)
			}
		*/

		return true
	}

	// call FindFiles with the smsync filter function to get the directories and files
	return lhlp.FindFiles([]string{cfg.SrcDirPath}, filter, 20)
}

// ProcessDirs creates new and deletes obsolete directories. processDirs
// displays the progress on the command line and returns the overall time that
// has been needed
func ProcessDirs(cfg *Config, dirs *[]*string) (<-chan Prog, error) {
	var (
		start    = time.Now() // start time of conversion
		prog     = Prog{Done: 0, Total: len(*dirs), Errors: 0, Elapsed: 0}
		progress = make(chan Prog)
		err      error
	)

	// nothing to do in case of empty directory array
	if len(*dirs) == 0 {
		return nil, nil
	}

	go func() {
		var (
			trgDirPath string
			exists     bool
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
			} else {
				// if it doesn't exist: create it
				if err = os.MkdirAll(trgDirPath, os.ModeDir|0755); (err != nil) && (err != os.ErrExist) {
					log.Errorf("Error from MkdirAll('%s'): %v", trgDirPath, err)
					return
				}
			}

			// increase counter
			prog.Done++
			// determine elapsed time
			prog.Elapsed = time.Since(start)

			progress <- prog
		}

		close(progress)
	}()

	return progress, nil
}

// ProcessFiles calls the conversion for all new or changes files. Files
// are processed in parallel using the package github.com/mipimipi/go-worker.
// It displays the progress on the command line and returns the overall time
// that was needed
func ProcessFiles(cfg *Config, files *[]*string) (<-chan Prog, error) {
	// variables needed to measure progress
	var (
		start    = time.Now() // start time of conversion
		prog     = Prog{Done: 0, Total: len(*files), Elapsed: 0}
		progress = make(chan Prog)
	)

	// nothing to do in case of empty files array
	if len(*files) == 0 {
		return nil, nil
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
			// increase number of transformed files
			prog.Done++
			// increase number of errors
			if r.(cvOutput).err != nil {
				prog.Errors++
			}
			// determine elapsed time
			prog.Elapsed = time.Since(start)
			// send current progress
			progress <- prog
		}

		close(progress)
	}()

	return progress, nil
}
