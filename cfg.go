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

// cfg.go implements the logic that is needed for the configuration
// of smsync.
// getCfg is the main function. It reads the configuration from the
// file SMSYNC_CONFIG (which is stored in the target directory).
// It is in INI format.

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/go-ini/ini"
	lhlp "github.com/mipimipi/go-lhlp"
	log "github.com/mipimipi/logrus"
)

// Constants for smsync configuration
const (
	cfgFileName       = "SMSYNC.CONF" // file name of config file
	cfgSectionGeneral = "general"     // id of general section
	cfgSectionRule    = "rule"        // base id of rule sections
	cfgKeyLastSync    = "last_sync"   // id of key for last sync time
	cfgKeySrcDir      = "source_dir"  // id of key for source directiory
	cfgKeyNumCPUs     = "num_cpus"    // id of key for #cpus to be used
	cfgKeyNumWrkrs    = "num_wrkrs"   // id of key for #workers to be created
	cfgKeySrc         = "source"      // id of key for source file suffix (rules)
	cfgKeyTrg         = "target"      // id of key for target file suffix (rules)
	cfgKeyTransform   = "transform"   // id of key for transformation to execute (rules), a. k. a. transformation string
)

const suffixStar = "*"

// config contains the content read from the smsync config file
type config struct {
	srcDirPath string          // source directory
	trgDirPath string          // target directory
	lastSync   time.Time       // timestamp when the last sync happend
	numCpus    int             // number of CPUs that gool is allowed to use
	numWrkrs   int             // number of worker Go routines to be created
	tfs        map[string]*tfm // transformation rules
}

// mapping of target suffix to transformation string (sometimes also
// called "transformation rule")
type tfm struct {
	trgSuffix string
	tfStr     string
}

// getTf retrieves a tfm structure for a given file path. In case it could be
// retrieved, a pointer to the tfm structure and true is returned, otherwise
// nil and false
func (cfg *config) getTf(f string) (*tfm, bool) {
	if _, ok := cfg.tfs[lhlp.FileSuffix(f)]; ok {
		return cfg.tfs[lhlp.FileSuffix(f)], true
	}
	if _, ok := cfg.tfs[suffixStar]; ok {
		return cfg.tfs[suffixStar], true
	}
	return nil, false
}

// getCfgFile opens configuration file and return handle
func getCfgFile() (*ini.File, error) {
	cfgFile, err := ini.InsensitiveLoad(filepath.Join(".", cfgFileName))
	if err != nil {
		// determine working directory for error message
		wd, err0 := os.Getwd()
		if err0 != nil {
			log.Errorf("Cannot determine working directory: %v", err0)
			return nil, fmt.Errorf("Cannot determine working directory: %v", err0)
		}
		log.Errorf("No configuration file found in '%s'", wd)
		return nil, fmt.Errorf("No configuration file found in '%s'", wd)
	}

	return cfgFile, nil
}

// getCfg reads the smsync configuration from the file ./SMSYNC.CONF and stores
// the configuration values in the attributes of instance of type config.
func getCfg() (*config, error) {
	// structure for transformation rule
	type rule struct {
		srcSuffix string // suffix of source file
		trgSuffix string // suffix of target file
		tfStr     string // transformation string
	}

	var cfg config

	log.Info("Read config file ...")

	// get handle for configuration file
	cfgFile, err := getCfgFile()
	if err != nil {
		return nil, err
	}

	// get section "GENERAL"
	sec, err := getGeneralSection(cfgFile)
	if err != nil {
		return nil, err
	}

	// get source directory and check if it exists and if it's a directory
	key, err := getKey(sec, cfgKeySrcDir, false)
	if err != nil {
		log.Errorf("Key '%s' does not exist", cfgKeySrcDir)
		return nil, err
	}
	fi, err := os.Stat(key.Value())
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Source directory '%s' doesn't exist", key.Value())
			return nil, fmt.Errorf("Source directory '%s' doesn't exist", key.Value())
		}
		log.Errorf("Error regarding source directory '%s': %v", key.Value(), err)
		return nil, fmt.Errorf("Error regarding source directory '%s': %v", key.Value(), err)
	}
	if !fi.IsDir() {
		log.Errorf("Source '%s' is no directory", key.Value())
		return nil, fmt.Errorf("Source '%s' is no directory", key.Value())
	}
	cfg.srcDirPath = key.Value()

	// get number of CPU's (optional). Default is to use all available cpus
	if key, err = getKey(sec, cfgKeyNumCPUs, false); err != nil {
		cfg.numCpus = runtime.NumCPU()
		log.Infof("num_cpus not configured. Use default: %d", cfg.numCpus)
	} else {
		if cfg.numCpus, err = key.Int(); err != nil {
			return nil, fmt.Errorf("Key '%s' has no invalid value: %v", cfgKeyNumCPUs, err)
		}
	}

	// get number of workers (optional). Per default it's set to the number of cpus
	if key, err = getKey(sec, cfgKeyNumWrkrs, false); err != nil {
		cfg.numWrkrs = cfg.numCpus
		log.Infof("num_wrkrs not configured. Use default: %d", cfg.numWrkrs)
	} else {
		if cfg.numWrkrs, err = key.Int(); err != nil {
			return nil, fmt.Errorf("Key '%s' has no invalid value: %v", cfgKeyNumWrkrs, err)
		}
	}

	// get last sync time (optional)
	if key, err = getKey(sec, cfgKeyLastSync, true); err != nil {
		log.Infof("No last sync time could be detected")
	} else {
		if cfg.lastSync, err = key.Time(); err != nil {
			log.Errorf("Last sync time couldn't be read: %v", err)
			return nil, fmt.Errorf("Last sync time couldn't be read: %v", err)
		}
	}

	// get rules
	var rls []rule
	for i := 0; ; i++ {
		// assemble section name for the i-th rule
		rlStr := cfgSectionRule + strconv.Itoa(i)

		// get section of i-th rule. If it's not existing: leave loop
		if sec, err = cfgFile.GetSection(rlStr); err != nil {
			break
		}

		var rl rule

		// get source suffix
		if key, err = getKey(sec, cfgKeySrc, false); err != nil {
			log.Errorf("No source suffix in rule #%d", i)
			return nil, fmt.Errorf("No source suffix in rule #%d", i)
		}
		rl.srcSuffix = key.Value()

		// get transformation
		if key, err = getKey(sec, cfgKeyTransform, false); err != nil {
			log.Infof("No transformation in rule #%d", i)
			rl.tfStr = ""
		} else {
			rl.tfStr = key.Value()
		}

		// check that transformation is copy or empty in case of suffix '*'.
		// if the transformation is empty it is set to copy.
		if rl.srcSuffix == suffixStar && rl.tfStr != tfCopyStr {
			if rl.tfStr != "" {
				return nil, fmt.Errorf("Rule #%d: For suffix '*' only copy transformation is allowed", i)
			}
			rl.tfStr = tfCopyStr
		}

		// get target suffix
		if key, err = getKey(sec, cfgKeyTrg, false); err != nil {
			log.Infof("Rule %d: Since no target suffix could be detected, target suffix will be set to source suffix", i)
			rl.trgSuffix = rl.srcSuffix
		} else {
			rl.trgSuffix = key.Value()
		}

		// in case of source suffix equals target suffix and empty transformation, the transformation is set to copy
		if (rl.srcSuffix == rl.trgSuffix) && rl.tfStr == "" {
			log.Infof("Rule #%d: Since source equals target format without transformation, transformation is set to copy", i)
			rl.tfStr = tfCopyStr
		}

		// check if either both suffices are '*' or both are not
		if (rl.srcSuffix == suffixStar && rl.trgSuffix != suffixStar) || (rl.srcSuffix != suffixStar && rl.trgSuffix == suffixStar) {
			log.Errorf("Rule #%d: Either both suffices need to be '*' or none", i)
			return nil, fmt.Errorf("Rule #%d: Either both suffices need to be '*' or none", i)
		}

		if rl.tfStr != tfCopyStr {
			// check if transformation is supported
			if _, ok := validTfs[tfKey{rl.srcSuffix, rl.trgSuffix}]; !ok {
				log.Errorf("Rule %d: Transformation of '%s' into '%s' not supported", i, rl.srcSuffix, rl.trgSuffix)
				return nil, fmt.Errorf("Rule %d: Transformation of '%s' into '%s' not supported", i, rl.srcSuffix, rl.trgSuffix)
			}
			// check if transformation is valid and fill in default values
			{
				tf := validTfs[tfKey{rl.srcSuffix, rl.trgSuffix}]
				if err := tf.normParams(&rl.tfStr); err != nil {
					return nil, fmt.Errorf("'%s' is not a valid transformation", rl.tfStr)
				}
			}
		}

		rls = append(rls, rl)
	}

	// raise error if no rules could be detected
	if len(rls) == 0 {
		log.Error("No transformation rules could be detected in config file")
		return nil, fmt.Errorf("No transformation rules could be detected in config file")
	}

	// allocate transformation map in config struct
	cfg.tfs = make(map[string]*tfm)

	// fill transformation map
	for _, rl := range rls {
		cfg.tfs[rl.srcSuffix] = &tfm{trgSuffix: rl.trgSuffix, tfStr: rl.tfStr}
	}

	// set target directory
	cfg.trgDirPath, _ = os.Getwd()

	return &cfg, nil
}

// getGeneralSection return a handle to the section 'GENERAL' of te config file
func getGeneralSection(cfgFile *ini.File) (*ini.Section, error) {
	// Get section "GENERAL"
	sec, err := cfgFile.GetSection(cfgSectionGeneral)
	if err != nil {
		log.Errorf("Section '%s' does not exist", cfgSectionGeneral)
		return nil, fmt.Errorf("Section '%s' does not exist", cfgSectionGeneral)
	}

	return sec, nil
}

// getKey checks if a key exists in ini file. If it exists, it'll be returned.
func getKey(sec *ini.Section, keyName string, nullOK bool) (*ini.Key, error) {
	// Get key for source directory
	if !sec.HasKey(keyName) {
		return nil, fmt.Errorf("Key '%s' does not exist", keyName)
	}
	if sec.Key(keyName).Value() == "" {
		if !nullOK {
			log.Errorf("Key '%s' has null value", keyName)
			return nil, fmt.Errorf("Key '%s' has null value", keyName)
		}
		log.Infof("Key '%s' has null value", keyName)
	}

	return sec.Key(keyName), nil
}

// summary prints a summary of the configuration to stdout
func (cfg *config) summary() {
	var (
		fmGen   = "%-15s : \033[1m%s\033[0m\n" // format string for general config values
		fmRl    string                         // format string for transformation rules
		hasStar bool
	)

	// assemble format string for transformation rules
	{
		var (
			lenTrg int
			lenSrc int
		)
		for srcSuffix, tf := range cfg.tfs {
			if len(srcSuffix) > lenSrc {
				lenSrc = len(srcSuffix)
			}
			if len(tf.trgSuffix) > lenTrg {
				lenTrg = len(tf.trgSuffix)
			}
		}
		fmRl = "\t\033[1m%-" + strconv.Itoa(lenSrc) + "s -> %-" + strconv.Itoa(lenTrg) + "s : %s\033[0m\n"
	}

	// headline
	fmt.Println("\n\033[1m\033[34m# Configuration\033[22m\033[39m")

	// source directory
	fmt.Printf(fmGen, "Source", cfg.srcDirPath)

	// target directory
	fmt.Printf(fmGen, "Destination", cfg.trgDirPath)

	// last sync time
	if cfg.lastSync.IsZero() {
		fmt.Printf(fmGen, "Last Sync", "Not set, initial sync")
	} else {
		fmt.Printf(fmGen, "Last Sync", cfg.lastSync.Local())
	}

	// number of CPU's & workers
	fmt.Printf(fmGen, "CPUs", strconv.Itoa(cfg.numCpus))
	fmt.Printf(fmGen, "Workers", strconv.Itoa(cfg.numWrkrs))

	// transformations
	fmt.Printf(fmGen, "Transformations", "")
	for srcSuffix, tf := range cfg.tfs {
		if srcSuffix == "*" {
			hasStar = true
			continue
		}
		fmt.Printf(fmRl, srcSuffix, tf.trgSuffix, tf.tfStr)
	}
	if hasStar {
		fmt.Printf(fmRl, "*", cfg.tfs["*"].trgSuffix, cfg.tfs["*"].tfStr)
	}
	fmt.Println()
}

// updateLastSync updates the last sync time in the configuration file.
// It's called after smsync has been run successfully
func (cfg *config) updateLastSync() error {
	// get configuration file handle
	cfgFile, err := getCfgFile()
	if err != nil {
		return err
	}

	// Get section "GENERAL"
	sec, err := cfgFile.GetSection(cfgSectionGeneral)
	if err != nil {
		log.Errorf("Section '%s' does not exist", cfgSectionGeneral)
		return fmt.Errorf("Section '%s' does not exist", cfgSectionGeneral)
	}

	// If key 'last_sync' doesn't exist ...
	if !sec.HasKey(cfgKeyLastSync) {
		// create it with empty value
		if _, err = sec.NewKey(cfgKeyLastSync, ""); err != nil {
			log.Errorf("Key %s cannot be created: %v", cfgKeyLastSync, err)
			err = fmt.Errorf("Key %s cannot be created: %v", cfgKeyLastSync, err)
			return err
		}
	}
	// set key value to current time in UTC
	sec.Key(cfgKeyLastSync).SetValue(time.Now().UTC().Format(time.RFC3339))

	// save config file
	if err = cfgFile.SaveTo("./" + cfgFileName); err != nil {
		log.Errorf("Configuration file cannot be saved: %v", err)
		return fmt.Errorf("Configuration file cannot be saved: %v", err)
	}
	log.Debug("Config has been saved")

	return nil
}
