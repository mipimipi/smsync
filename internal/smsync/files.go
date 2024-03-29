package smsync

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/go-utilities/file"
	fp "gitlab.com/go-utilities/filepath"
	"gitlab.com/go-utilities/reflect"
)

// errDir is the directory that stores error logs from conversion
const errDir = "smsync.cv.errs"

// assembleTrgFile creates the target file path from the source file path
func assembleTrgFile(cfg *Config, srcFile string) string {
	var trgSuffix string

	// get conversion rule from config
	cvm, exists := cfg.getCv(srcFile)
	if !exists {
		log.Errorf("No conversion rule for '%s'", srcFile)
		return ""
	}

	// if corresponding conversion rule is for '*' ...
	if cvm.TrgSuffix == suffixStar {
		// ... target suffix is same as source suffix
		trgSuffix = fp.Suffix(srcFile)
	} else {
		// ... otherwise take target suffix from conversion rule
		trgSuffix = cvm.TrgSuffix
	}

	trgFile, err := fp.PathRelCopy(cfg.SrcDir.Path(),
		fp.PathTrunk(srcFile)+"."+trgSuffix,
		cfg.TrgDir.Path())
	if err != nil {
		log.Errorf("Target path cannot be assembled: %v", err)
		return ""
	}
	return trgFile
}

// CleanUp remove temporary files
func CleanUp(cfg *Config) {
	// remove log file if it's empty
	_ = file.RemoveEmpty(filepath.Join(cfg.TrgDir.Path(), LogFile))
	log.Debug("Removed log files (at least tried to do that)")
}

// deleteObsoleteFiles deletes directories and files that are available in the
// target directory tree but not in the source directory tree. It is called
// for all source directories that have been changed since the last sync.
// Typically, this is relevant if directories or files have been renamed or
// deleted. In this case, the parent directory has been touched. That's used to
// identify the relevant directories and files.
func deleteObsoleteFiles(cfg *Config, srcDir file.Info) {
	log.Debugf("smsync.deleteObsoleteFiles(%s): BEGIN", srcDir.Path())
	defer log.Debugf("smsync.deleteObsoleteFiles(%s): END", srcDir.Path())

	var (
		trgDir string
		exists bool
		err    error
	)

	// assemble target directory path
	trgDir, err = fp.PathRelCopy(cfg.SrcDir.Path(),
		srcDir.Path(),
		cfg.TrgDir.Path())
	if err != nil {
		log.Errorf("deleteObsoleteFiles: %v", err)
		return
	}

	// nothing to do if target directory doesn't exist
	if exists, err = file.Exists(trgDir); err != nil {
		log.Errorf("deleteObsoleteFiles: %v", err)
		return
	}
	if !exists {
		return
	}

	// read entries of target directory
	trgEntrs, err := os.ReadDir(trgDir)
	if err != nil {
		log.Errorf("deleteObsoleteFiles: %v", err)
		return
	}

	// loop over all entries of target directory
	for _, trgEntr := range trgEntrs {
		if trgEntr.IsDir() {
			// if entry is a directory ...
			b, err := file.Exists(filepath.Join(srcDir.Path(), trgEntr.Name()))
			if err != nil {
				log.Errorf("deleteObsoleteFiles: %v", err)
				return
			}
			// ... and the counterpart on source side doesn't exists: ...
			if !b {
				// ... delete entry
				if err = os.RemoveAll(filepath.Join(trgDir, trgEntr.Name())); err != nil {
					log.Errorf("deleteObsoleteFiles: %v", err)
					return
				}
			}
		} else {
			// if entry is a file ...

			// if entry is not regular: do nothing and continue loop
			if !trgEntr.Type().IsRegular() {
				continue
			}
			// exclude smsync files (smsync.log or smsync.yaml) from deletion logic
			if strings.Contains(trgEntr.Name(), LogFile) || strings.Contains(trgEntr.Name(), cfgFile) {
				continue
			}
			// check if counterpart file on source side exists
			tr := fp.PathTrunk(trgEntr.Name())
			fs, err := filepath.Glob(fp.EscapePattern(filepath.Join(srcDir.Path(), tr)) + ".*")
			if err != nil {
				log.Errorf("deleteObsoleteFiles: %v", err)
				return
			}
			// if counterpart does not exist: delete entry
			if fs == nil {
				if err = os.Remove(filepath.Join(trgDir, trgEntr.Name())); err != nil {
					log.Errorf("deleteObsoleteFiles: %v", err)
				}
			}
		}
	}
}

// DeleteTrg deletes all entries of the target directory
func deleteTrg(dir string) {
	log.Debug("smsync.deleteTrg: BEGIN")
	defer log.Debug("smsync.deleteTrg: END")

	// read entries of target directory
	trgEntrs, err := os.ReadDir(dir)
	if err != nil {
		log.Errorf("deleteTrg: %v", err)
		return
	}

	// loop over all entries of target directory
	for _, trgEntr := range trgEntrs {
		// don't delete smsync files (smsync.log or SMSYNC.yaml)
		if !trgEntr.IsDir() && (strings.Contains(trgEntr.Name(), LogFile) || strings.Contains(trgEntr.Name(), cfgFile)) {
			continue
		}
		// delete entry
		if err = os.RemoveAll(filepath.Join(dir, trgEntr.Name())); err != nil {
			log.Errorf("deleteTrg: %v", err)
			return
		}
	}
}

// GetSyncFiles determines which files need to be synched.
// Files are only relevant if they need to be converted. Directories are only
// relevant if they already exist on target side and might contain obsolete
// files.
// Note, that the entire directory path of a file is created (if this path
// doesn't exist yet) if this file is created on target side
// Note, that it's important to keep the number of existence checks on target
// side as small as possible because target devices aer often external devices
// (USB etc.). Thus existence checks on target side are time consuming.
// Therefore, because relevant directories lead to existence checks on target
// side in deleteObsoleteFiles(), the number of relevant directories has to be
// as small as possible.
func GetSyncFiles(cfg *Config, init bool) (files *[]*file.Info) {
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
		log.Debugf("smsync.GetSyncFiles.filter(%s): BEGIN", srcFile.Path())
		defer log.Debugf("smsync.GetSyncFiles(%s): END", srcFile.Path())

		// distinguish between directories and files
		if srcFile.IsDir() {
			log.Debug("Directory")

			// if a directory is excluded, itself and all sub directories and
			// files are not relevant
			if reflect.Contains(cfg.Excludes, srcFile.Path()) {
				return false, file.InvalidFromSuper
			}
			// if relevance is propagated from the parent, this directory is not
			// relevant, but all sub directories and files are relevant
			if vp == file.ValidFromSuper {
				log.Debug("VALID FROM SUPER -> FALSE, VALID FROM SUPER")

				return false, file.ValidFromSuper
			}
			// if the target side shall be initialized, this directory is not
			// relevant but relevance is propagated downwards
			if init {
				log.Debug("init -> FALSE, VALID FROM SUPER")
				return false, file.ValidFromSuper
			}
			// if the directory has been changed since the last sync, it's
			// relevant. This is because, it could be an indicator, that either
			// sub directories or files in that directory have been renamed.
			// Relevance is not propagated since the filter logic needs to be
			// applied to them
			if srcFile.ModTime().After(cfg.LastSync) && !cfg.LastSync.IsZero() {
				log.Debug("Modtime > lastsync -> TRUE, NONE FROM SUPER")
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
				log.Debug("Parent dir changed")

				// assemble target directory
				trgDir, _ := fp.PathRelCopy(cfg.SrcDir.Path(), srcFile.Path(), cfg.TrgDir.Path())
				// check if target directory exists
				if exists, err := file.Exists(trgDir); err == nil && !exists {
					log.Debug("Trg doesn't exist -> FALSE, NONE FROM SUPER")

					return false, file.ValidFromSuper
				}
			}
			// if none of the above rules applied, this directory is not
			// relevant and the downward content needs to be filtered
			log.Debug("Nothing applied -> FALSE, NONE FROM SUPER")
			return false, file.NoneFromSuper
		}
		// Here, srcFile is a file (no directory). For files, downward
		// propagation of relevance makes no sense. Thus, in this branch,
		// 'NoneFromSuper' is always returned
		log.Debug("File")

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
			log.Debug("VALID FROM SUPER -> TRUE, VALID FROM SUPER")

			return true, file.NoneFromSuper
		}
		// assemble target file name and check if file exists
		trgFile := assembleTrgFile(cfg, srcFile.Path())
		exists, inf, err := file.ExistsInfo(trgFile)
		// if this file has been changed since last sync and if the counterpart
		// on target side does either not exist or exists but is older than the
		// last sync time, then this file is relevant.
		// Note, that if the last run has been interrupted, it could be that the
		// counterpart on target side has already been updated before the
		// interrupt (and thus its mod time is after the last sync). In this
		// case this file is not relevant.
		if srcFile.ModTime().After(cfg.LastSync) {
			log.Debug("Modtime > lastsync")

			if err == nil && (!exists || inf.ModTime().Before(cfg.LastSync) && !cfg.LastSync.IsZero()) {
				log.Debug("Trg doesn't exist or modtime < lastsync or lastsync==0 -> TRUE, NONE FROM SUPER")

				return true, file.NoneFromSuper
			}
			log.Debug("Trg exist and (modtime > lastsync or lastsync!=0) -> FALSE, NONE FROM SUPER")
			return false, file.NoneFromSuper
		}
		// if parent was changed it could be that this files was renamed. Thus, it
		// needs to be checked if it's counterpart exists on target side. If the
		// counterpart exists, then this file is not relevant. Otherwise it is
		// relevant.
		if parentDirChgd(filepath.Dir(srcFile.Path()), cfg.LastSync) && err == nil && !exists {
			log.Debug("Parent dir changed and trg doesn't exist -> TRUE, NONE FROM SUPER")
			return true, file.NoneFromSuper
		}
		log.Debug("Nothing applied -> FALSE, NONE FROM SUPER")
		return false, file.NoneFromSuper
	}

	// call FindFiles with the smsync filter function to get the directories and files
	files = file.Find([]file.Info{cfg.SrcDir}, filter, 1)

	return files
}

// parentDirChgd returns true if the file speicified by path has changed after
// time t
func parentDirChgd(path string, t time.Time) bool {
	fi, err := os.Stat(path)
	if err != nil {
		log.Errorf("parentDirChgd: %v", err)
		return false
	}
	if fi.IsDir() && fi.ModTime().After(t) {
		return true
	}
	return false
}
