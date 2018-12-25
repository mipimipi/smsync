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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	lhlp "github.com/mipimipi/go-lhlp"
	"github.com/mipimipi/go-lhlp/file"
	log "github.com/sirupsen/logrus"
)

// errDir is the directory that stores error logs from conversion
const errDir = "smsync.err"

// cleanUp removes temporary files and directories from smsync that are
// obsolete
func cleanUp(cfg *Config) error {
	log.Debug("smsync.cleanUp: START")
	defer log.Debug("smsync.cleanUp: END")

	var (
		err     error
		logFile = filepath.Join(cfg.TrgDirPath, logFileName)
	)

	// remove log file if it's empty
	if err = file.RemoveEmpty(logFile); err != nil {
		return err
	}

	return nil
}

// deleteObsoleteFiles deletes directories and files that are available in the
// target directory tree but not in the source directory tree. It is called
// for all source directories that have been changes since the last sync
func deleteObsoleteFiles(cfg *Config, srcDir file.Info) error {
	log.Debugf("smsync.deleteObsoleteFiles(%s): START", srcDir.Path())
	defer log.Debugf("smsync.deleteObsoleteFiles(%s): END", srcDir.Path())

	// assemble target directory path
	trgDirPath, err := file.PathRelCopy(cfg.SrcDirPath, srcDir.Path(), cfg.TrgDirPath)
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
		log.Debugf("processing %s", trgEntr.Name())

		if trgEntr.IsDir() {
			// if entry is a directory ...
			b, err := file.Exists(filepath.Join(srcDir.Path(), trgEntr.Name()))
			if err != nil {
				log.Errorf("%v", err)
			}
			// ... and the counterpart on source side doesn't exists: ...
			if !b {
				log.Debug("is directory and src counterpart doesn't exist: DELETE")
				// ... delete entry
				if err = os.RemoveAll(filepath.Join(trgDirPath, trgEntr.Name())); err != nil {
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
			tr := file.PathTrunk(trgEntr.Name())
			fs, err := filepath.Glob(file.EscapePattern(filepath.Join(srcDir.Path(), tr)) + ".*")
			if err != nil {
				log.Errorf("Error from Glob('%s'): %v", file.EscapePattern(filepath.Join(srcDir.Path(), tr))+".*", err)
				return err
			}
			// if counterpart does not exist: ...
			if fs == nil {
				log.Debug("is file and src counterpart doesn't exist: DELETE")
				// ... delete entry
				if err = os.Remove(filepath.Join(trgDirPath, trgEntr.Name())); err != nil {
					log.Errorf("Cannot remove '%s': %v", filepath.Join(trgDirPath, trgEntr.Name()), err)
					return err
				}
			}
		}
		log.Debug("src counterpart exists: DON'T DELETE")
	}

	return nil
}

// DeleteTrg deletes all entries of the target directory
func deleteTrg(cfg *Config) error {
	log.Debug("smsync.deleteTrg: START")
	defer log.Debug("smsync.deleteTrg: END")

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
func GetSyncFiles(cfg *Config, init bool) (*[]file.Info, *[]file.Info) {
	log.Debug("smsync.GetSyncFiles: START")
	defer log.Debug("smsync.GetSyncFiles: END")

	// filter function needed for FindFiles
	filter := func(srcFile file.Info, propagated bool) (bool, bool) {
		log.Debugf("smsync.GetSyncFiles.filter(%s): START", srcFile.Path())
		defer log.Debugf("smsync.GetSyncFiles.filter(%s): END", srcFile.Path())

		var (
			trgFile string
			err     error
		)

		// check if file is relevant for smsync (i.e. its suffix is contained
		// in the smsync config). If not: Return false
		if !srcFile.IsDir() {
			_, ok := cfg.getCv(srcFile.Path())
			if !ok {
				log.Debug("suffix is not contained smsync config: INVALID, NO PROPAGATE")
				return false, false
			}
			if propagated {
				log.Debug("suffix is contained smsync config and propagated: VALID, NO PROPAGATE")
				return true, false
			}
			log.Debug("suffix is contained smsync config, but not propagated: GO AHEAD")
		}

		// check if the directory needs to be excluded
		if srcFile.IsDir() && lhlp.Contains(cfg.Excludes, srcFile.Path()) {
			log.Debug("directory excluded: INVALID, NO PROPAGATE")
			return false, false
		}

		// assemble target file/directory path
		if srcFile.IsDir() {
			trgFile, err = file.PathRelCopy(cfg.SrcDirPath, srcFile.Path(), cfg.TrgDirPath)
		} else {
			trgFile, _ = assembleTrgFile(cfg, srcFile.Path())
		}
		if err != nil {
			log.Errorf("Target path cannot be assembled: %v", err)
			return false, false
		}
		// if file/directory doesn't exists: return true
		if exists, _ := file.Exists(trgFile); !exists {
			log.Debug("target file doesn't exist:: VALID, PROPAGATE")
			return true, true
		}

		// check if the file/directory has been changed since last sync.
		if srcFile.ModTime().After(cfg.LastSync) && !cfg.WIP {
			log.Debug("source file has been changed and not WIP: VALID, NO PROPAGATE")
			return true, false
		}

		// if init: file/directory is relevant
		if init {
			log.Debug("init: VALID, PROPAGATE")
			return true, true
		}

		log.Debug("nothing applied: INVALID, NO PROPAGATE")

		return false, false
	}

	// call FindFiles with the smsync filter function to get the directories and files
	return file.Find([]string{cfg.SrcDirPath}, filter, 20)
}
