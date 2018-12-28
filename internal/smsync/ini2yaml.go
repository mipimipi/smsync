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

// ini2yaml.go implements a conversion of the old config file SMSYNC.CONF,
// which is in ini format, into the cfgYml structure.

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-ini/ini"
	log "github.com/sirupsen/logrus"
)

// Constants for smsync configuration (ini file)
const (
	iniFileName       = "SMSYNC.CONF" // file name of config file
	iniSectionGeneral = "general"     // id of general section
	iniSectionRule    = "rule"        // base id of rule sections
	iniKeyLastSync    = "last_sync"   // id of key for last sync time
	iniKeySrcDir      = "source_dir"  // id of key for source directiory
	iniKeyNumCPUs     = "num_cpus"    // id of key for #cpus to be used
	iniKeyNumWrkrs    = "num_wrkrs"   // id of key for #workers to be created
	iniKeySrc         = "source"      // id of key for source file suffix (rules)
	iniKeyTrg         = "target"      // id of key for target file suffix (rules)
	iniKeyTransform   = "conversion"  // id of key for conversion to execute, a. k. a. conversion rule
)

// getCfgFile opens the configuration file and returns a handle
func getCfgFile() (*ini.File, error) {
	cfgFile, err := ini.Load(iniFileName)
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

// init2yaml reads the smsync configuration from the file ./SMSYNC.CONF and stores
// the configuration values in the yaml file smsync.yaml
func ini2yaml() error {
	var cfgY cfgYml

	// get handle for configuration file
	cfgFile, err := getCfgFile()
	if err != nil {
		log.Errorf("%v", err)
		return err
	}

	// get section "GENERAL"
	sec, err := getGeneralSection(cfgFile)
	if err != nil {
		return err
	}

	// get source directory and check if it exists and if it's a directory
	key, err := getKey(sec, iniKeySrcDir, false)
	if err == nil {
		cfgY.SrcDir = key.Value()
	}

	// get number of CPU's
	if key, err = getKey(sec, iniKeyNumCPUs, false); err == nil {
		i, _ := key.Int()
		cfgY.NumCPUs = uint(i)
	}

	// get number of workers (optional). Per default it's set to the number of cpus
	if key, err = getKey(sec, iniKeyNumWrkrs, false); err == nil {
		i, _ := key.Int()
		cfgY.NumWrkrs = uint(i)
	}

	// get last sync time (optional)
	if key, err = getKey(sec, iniKeyLastSync, true); err == nil {
		t, _ := key.Time()
		cfgY.LastSync = t.Format(time.RFC3339)
	}

	// get rules

	for i := 0; ; i++ {
		// get section of i-th rule. If it's not existing: leave loop
		if sec, err = cfgFile.GetSection(iniSectionRule + strconv.Itoa(i)); err != nil {
			break
		}

		var rl rule

		// get source suffix
		if key, err = getKey(sec, iniKeySrc, false); err == nil {
			rl.Source = key.Value()
		}

		// get target suffix
		if key, err = getKey(sec, iniKeyTrg, false); err == nil {
			rl.Target = key.Value()
		}

		// get conversion
		if key, err = getKey(sec, iniKeyTransform, false); err == nil {
			rl.Conversion = key.Value()
		}

		cfgY.Rules = append(cfgY.Rules, rl)
	}

	// write config file in yaml format
	cfgY.write()

	return nil
}

// getGeneralSection returns a handle to the section 'GENERAL' of the config file
func getGeneralSection(cfgFile *ini.File) (*ini.Section, error) {
	// Get section "GENERAL"
	sec, err := cfgFile.GetSection(iniSectionGeneral)
	if err != nil {
		log.Errorf("Section '%s' does not exist", iniSectionGeneral)
		return nil, fmt.Errorf("Section '%s' does not exist", iniSectionGeneral)
	}

	return sec, nil
}

// getKey checks if a key exists in the config file. If it exists, it'll be returned.
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
