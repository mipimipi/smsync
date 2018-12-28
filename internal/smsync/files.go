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
	"time"

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

// GetSyncFiles determines which directories and files need to be synched.
// Files are only relevant if they need to be converted. Directories are only
// relevant if they already exist on target side and might contain obsolete
// files (which will be handled by deleteObsoleteFiles).
// Note, that the entire directory path of a file is created (if this path
// doesn't exist yet) if this file is created on target side
// Note, that it's important to keep the number of existence checks on target
// side as small as possible because target devices often external devices (USB
// etc.). Thus existence checks on target side are time consuming. Therefore,
// because relevant directories lead to existence checks on target side in
// deleteObsoleteFiles(), the number of relevant directories has to be as small
// as possible.
func GetSyncFiles(cfg *Config, init bool) (*file.InfoSlice, *file.InfoSlice, error) {
	log.Debug("smsync.GetSyncFiles: BEGIN")
	defer log.Debug("smsync.GetSyncFiles: END")

	// filter function needed as input for file.Find(). This function contains
	// the filter logic for files and directories.
	// Note, what propagation means in this context:
	// - NoneFromSuper: Filter logic is applied to sub directories and files
	// - ValidFromSuper: Downward files are relevant, downward directories
	//                   are not relevant without applying filter logic
	// - InvalidFromSuper: file.Find() will not descend into the
	//                     corresponding directory. I.e. all downward content
	//                     will be ignored
	filter := func(srcFile file.Info, vp file.ValidPropagate) (bool, file.ValidPropagate) {

		// distinguish between directories and files
		if srcFile.IsDir() {
			// if a directory is excluded, itself and all sub directories and
			// files are not relevant
			if lhlp.Contains(cfg.Excludes, srcFile.Path()) {
				return false, file.InvalidFromSuper
			}
			// if relevance is propagated from the parent, this directory is not
			// relevant, but all sub directries and files are relevant
			if vp == file.ValidFromSuper {
				return false, file.ValidFromSuper
			}
			// if the target side shall be initialized or it's the first sync
			// run for that target (in which case no counterparts can be
			// there), this directory is not relevant but relevance is
			// propagated downwards
			if init || cfg.LastSync.IsZero() {
				return false, file.ValidFromSuper
			}
			// if the directory has been changed since the last sync, it's
			// relevant. This is because, it could be an indicator, that either
			// sub directories oer files in that directory have been renamed.
			// Relevance is not propagated since the filter logic needs to be
			// applied to them
			if srcFile.ModTime().After(cfg.LastSync) {
				return true, file.NoneFromSuper
			}
			// if parent has been changed ...
			if parentDirChgd(filepath.Dir(srcFile.Path()), cfg.LastSync) {
				// ... it could be that this directory has been renamed. Thus,
				// it needs to be checked if it's counterpart exists on target
				// side. If that's the case, this directory is not relevant but
				// downwards content needs to be filtered. Otherwise, it's
				// clear that the downward content is relevant, filter logic
				// doesn't have to be applied.

				// assemble target directory
				trgDir, _ := file.PathRelCopy(cfg.SrcDir, srcFile.Path(), cfg.TrgDir)
				// check if target directory exists
				if exists, err := file.Exists(trgDir); err == nil && exists {
					return false, file.NoneFromSuper
				}
				return false, file.ValidFromSuper
			}
			// if none of the above rules applied, the directory is not
			// relevant and the downward content is filtered
			return false, file.NoneFromSuper
		}
		// srcFile is a file (no directory). For files, downward propagation of
		// relevance makes no sense. Thus, in this branch, 'NoneFromSuper' is
		// always returned

		// if file is not regular, it's not relevant
		if !srcFile.Mode().IsRegular() {
			return false, file.NoneFromSuper
		}
		// if file type is not relevant per the smsync configuration, this file
		// is not relevant
		if _, ok := cfg.getCv(srcFile.Path()); !ok {
			return false, file.NoneFromSuper
		}
		// if relevance is propagated from the parent, this file is relevant
		// without further checks
		if vp == file.ValidFromSuper {
			return true, file.NoneFromSuper
		}
		// if this file has been changed since last sync, it is relevant
		if srcFile.ModTime().After(cfg.LastSync) {
			return true, file.NoneFromSuper
		}
		// if parent has been changed ...
		if parentDirChgd(filepath.Dir(srcFile.Path()), cfg.LastSync) {
			// ... it could be that this files has been renamed. Thus,
			// it needs to be checked if it's counterpart exists on target
			// side. If the counterpart exists, then this file is not relevant.
			// Otherwise it is relevant.

			// assemble target file
			trgFile, _ := assembleTrgFile(cfg, srcFile.Path())
			// check if target file exists
			if exists, err := file.Exists(trgFile); err == nil && !exists {
				return true, file.NoneFromSuper
			}
		}
		return false, file.NoneFromSuper
	}

	// call FindFiles with the smsync filter function to get the directories and files
	dirs, files := file.Find([]string{cfg.SrcDir}, filter, 20)

	// sort dir array to allow more efficient processing later
	sort.Sort(*dirs)

	return dirs, files, nil
}

// parentDirChgd returns true if the file speicified by path has changed after
// time t
func parentDirChgd(path string, t time.Time) bool {
	fi, _ := os.Stat(path)
	if fi.IsDir() && fi.ModTime().After(t) {
		return true
	}
	return false
}
