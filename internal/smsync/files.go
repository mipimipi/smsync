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
	"sort"
	"strings"

	lhlp "github.com/mipimipi/go-lhlp"
	"github.com/mipimipi/go-lhlp/file"
	log "github.com/sirupsen/logrus"
)

// errDir is the directory that stores error logs from conversion
const errDir = "smsync.cv.errs"

// deleteObsoleteFiles deletes directories and files that are available in the
// target directory tree but not in the source directory tree. It is called
// for all source directories that have been changed since the last sync
func deleteObsoleteFiles(cfg *Config, srcDir file.Info) error {
	log.Debugf("smsync.deleteObsoleteFiles(%s): BEGIN", srcDir.Path())
	defer log.Debugf("smsync.deleteObsoleteFiles(%s): END", srcDir.Path())

	var (
		trgDir string
		exists bool
		err    error
	)

	// assemble target directory path
	trgDir, err = file.PathRelCopy(cfg.SrcDir, srcDir.Path(), cfg.TrgDir)
	if err != nil {
		log.Errorf("Target path cannot be assembled: %v", err)
		return fmt.Errorf("Target path cannot be assembled: %v", err)
	}

	// nothing to do if target directory doesn't exist
	if exists, err = file.Exists(trgDir); err != nil {
		log.Errorf("Cannot determine if directory '%s' exists: %v", trgDir, err)
		return fmt.Errorf("Cannot determine if directory '%s' exists: %v", trgDir, err)
	}
	if !exists {
		log.Debug("Target directory doesn't exist: DO NOTHING")
		return nil
	}

	// read entries of target directory
	trgEntrs, err := ioutil.ReadDir(trgDir)
	if err != nil {
		log.Errorf("Cannot read directory '%s': %v", trgDir, err)
		return fmt.Errorf("Cannot read directory '%s': %v", trgDir, err)
	}

	// loop over all entries of target directory
	for _, trgEntr := range trgEntrs {
		log.Debugf("processing %s", trgEntr.Name())

		if trgEntr.IsDir() {
			// if entry is a directory ...
			b, err := file.Exists(filepath.Join(srcDir.Path(), trgEntr.Name()))
			if err != nil {
				log.Errorf("Cannot determine if file '%s' exists: %v", filepath.Join(srcDir.Path(), trgEntr.Name()), err)
				return fmt.Errorf("Cannot determine if file '%s' exists: %v", filepath.Join(srcDir.Path(), trgEntr.Name()), err)
			}
			// ... and the counterpart on source side doesn't exists: ...
			if !b {
				log.Debug("is directory and src counterpart doesn't exist: DELETE")
				// ... delete entry
				if err = os.RemoveAll(filepath.Join(trgDir, trgEntr.Name())); err != nil {
					log.Errorf("Cannot remove '%s': %v", filepath.Join(trgDir, trgEntr.Name()), err)
					return fmt.Errorf("Cannot remove '%s': %v", filepath.Join(trgDir, trgEntr.Name()), err)
				}
			}
		} else {
			// if entry is a file

			// if entry is not regular: do nothing and continue loop
			if !trgEntr.Mode().IsRegular() {
				log.Debug("is file but not regular: DON'T DELETE")
				continue
			}
			// exclude smsync files (smsync.log or smsync.yaml) from deletion logic
			if strings.Contains(trgEntr.Name(), LogFile) || strings.Contains(trgEntr.Name(), cfgFile) {
				log.Debug("is smsync.log or smsync.yaml: DON'T DELETE")
				continue
			}
			// check if counterpart file on source side exists
			tr := file.PathTrunk(trgEntr.Name())
			fs, err := filepath.Glob(file.EscapePattern(filepath.Join(srcDir.Path(), tr)) + ".*")
			if err != nil {
				log.Errorf("Error from Glob('%s'): %v", file.EscapePattern(filepath.Join(srcDir.Path(), tr))+".*", err)
				return fmt.Errorf("Error from Glob('%s'): %v", file.EscapePattern(filepath.Join(srcDir.Path(), tr))+".*", err)
			}
			// if counterpart does not exist: ...
			if fs == nil {
				log.Debug("is file and src counterpart doesn't exist: DELETE")
				// ... delete entry
				if err = os.Remove(filepath.Join(trgDir, trgEntr.Name())); err != nil {
					log.Errorf("Cannot remove '%s': %v", filepath.Join(trgDir, trgEntr.Name()), err)
					return fmt.Errorf("Cannot remove '%s': %v", filepath.Join(trgDir, trgEntr.Name()), err)
				}
			}
		}
		log.Debug("src counterpart exists: DON'T DELETE")
	}

	return nil
}

// DeleteTrg deletes all entries of the target directory
func deleteTrg(cfg *Config) error {
	log.Debug("smsync.deleteTrg: BEGIN")
	defer log.Debug("smsync.deleteTrg: END")

	// read entries of target directory
	trgEntrs, err := ioutil.ReadDir(cfg.TrgDir)
	if err != nil {
		log.Errorf("Cannot read directory '%s': %v", cfg.TrgDir, err)
		return fmt.Errorf("Cannot read directory '%s': %v", cfg.TrgDir, err)
	}

	// loop over all entries of target directory
	for _, trgEntr := range trgEntrs {
		// don't delete smsync files (smsync.log or SMSYNC.yaml)
		if !trgEntr.IsDir() && (strings.Contains(trgEntr.Name(), LogFile) || strings.Contains(trgEntr.Name(), cfgFile)) {
			continue
		}
		// delete entry
		if err = os.RemoveAll(filepath.Join(cfg.TrgDir, trgEntr.Name())); err != nil {
			log.Errorf("Cannot remove '%s': %v", filepath.Join(cfg.TrgDir, trgEntr.Name()), err)
			return fmt.Errorf("Cannot remove '%s': %v", filepath.Join(cfg.TrgDir, trgEntr.Name()), err)
		}
	}

	// everything's fine
	return nil
}

// GetSyncFiles determines which directories and files need to be synched
func GetSyncFiles(cfg *Config, init bool) (*file.InfoSlice, *file.InfoSlice, error) {
	log.Debug("smsync.GetSyncFiles: BEGIN")
	defer log.Debug("smsync.GetSyncFiles: END")

	// filter function needed for file.Find(...)
	filter := func(srcFile file.Info, propagated bool) (bool, bool) {
		log.Debugf("smsync.GetSyncFiles.filter(%s): BEGIN", srcFile.Path())
		defer log.Debugf("smsync.GetSyncFiles.filter(%s): END", srcFile.Path())

		if srcFile.IsDir() {
			// check if the directory shall be excluded
			if lhlp.Contains(cfg.Excludes, srcFile.Path()) {
				log.Debug("directory excluded: INVALID, PROPAGATE")
				return false, true
			}
			// assemble target directory
			trgDir, _ := file.PathRelCopy(cfg.SrcDir, srcFile.Path(), cfg.TrgDir)
			if exists, _ := file.Exists(trgDir); !exists {
				// if directory doesn't exist on target side: not relevant
				log.Debug("target counterpart of directory doesn't exist: INVALID, NO PROPAGATE")
				return false, false
			}

		} else {
			// if file is not regular: not relevant
			if !srcFile.Mode().IsRegular() {
				log.Debug("file is regular: INVALID, NO PROPAGATE")
				return false, false
			}
			// check if file is relevant for smsync (i.e. its suffix is
			// contained in the smsync config)
			_, ok := cfg.getCv(srcFile.Path())
			if !ok {
				log.Debug("suffix is not contained in smsync config: INVALID, NO PROPAGATE")
				return false, false
			}
			// if init: file is relevant
			if init {
				log.Debug("init: VALID, PROPAGATE")
				return true, true
			}
			// if file doesn't exists on target side: it's valid.
			// Note: The timestamp of a file is not changed if it's renamed.
			// Therefore this check is necessary
			trgFile, _ := assembleTrgFile(cfg, srcFile.Path())
			if exists, _ := file.Exists(trgFile); !exists {
				log.Debug("target file doesn't exist:: VALID, PROPAGATE")
				return true, true
			}
		}

		// check if the file/directory has been changed since last sync
		if srcFile.ModTime().After(cfg.LastSync) && !cfg.WIP {
			log.Debug("source file has been changed and not WIP: VALID, NO PROPAGATE")
			return true, false
		}

		log.Debug("nothing applied: INVALID, NO PROPAGATE")

		return false, false
	}

	// call FindFiles with the smsync filter function to get the directories and files
	dirs, files := file.Find([]string{cfg.SrcDir}, filter, 20)

	// sort dir array to allow more efficient processing later
	sort.Sort(*dirs)

	return dirs, files, nil
}
