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

// cfg.go implements the logic that is needed for the configuration
// of smsync.
// Get is the main function. It reads the configuration from the
// file smsync.yaml (which is stored in the target directory).

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mipimipi/go-lhlp/file"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// Constants for smsync configuration
const (
	cfgFile     = "smsync.yaml" // file name of config file
	suffixStar  = "*"           // wildcard for music file suffix
	procStatWIP = "wip"         // work in progress
)

// structure for conversion rule
type rule struct {
	Source     string `yaml:"source"`               // source file format
	Target     string `yaml:"target,omitempty"`     // target file format
	Conversion string `yaml:"conversion,omitempty"` // conversion string
}

// cfgYml is used to read from and write to the config yaml file
type cfgYml struct {
	ProcStat string   `yaml:"processing_status,omitempty"` // work in progress flag
	SrcDir   string   `yaml:"source_dir"`                  // source directory
	Excludes []string `yaml:"exclude,omitempty"`           // exclude these directories
	LastSync string   `yaml:"last_sync,omitempty"`         // timestamp when the last sync happened
	NumCPUs  uint     `yaml:"num_cpus,omitempty"`          // number of CPUs that gool is allowed to use
	NumWrkrs uint     `yaml:"num_wrkrs,omitempty"`         // number of worker Go routines to be created
	Rules    []rule   `yaml:"rules"`                       // conversion rules
}

// Config contains the enriched data that has been read from the config file
type Config struct {
	LastSync time.Time       // timestamp when the last sync happened
	WIP      bool            // work in progress flag
	SrcDir   string          // source directory
	TrgDir   string          // target directory
	Excludes []string        // exclude these directories
	NumCpus  uint            // number of CPUs that gool is allowed to use
	NumWrkrs uint            // number of worker Go routines to be created
	Cvs      map[string]*cvm // conversion rules
}

// mapping of target suffix to conversion parameter string
type cvm struct {
	TrgSuffix string
	NormCvStr string // normalized conversion string (e.g. defaults are added)
}

// Get reads the smsync configuration from the file ./SMSYNC.yaml and stores
// the configuration values in the structure *config.
func (cfg *Config) Get(init bool) error {
	log.Debug("smsync.Config.Get: START")
	defer log.Debug("smsync.Config.Get: END")

	var (
		cfgY cfgYml
		err  error
	)

	log.Info("Read config from file ...")

	// read config from file
	if err = cfgY.read(); err != nil {
		// if config file in yaml for at exists, try to convert an old
		// potentially existing ini file into a yaml file and try again
		ini2yaml()
		if err = cfgY.read(); err != nil {
			return err
		}
	}

	// set processing status
	if cfgY.ProcStat == procStatWIP {
		cfg.WIP = true
	}

	// check if the configured source dir exists and is a directory
	if err = checkDir(cfgY.SrcDir); err != nil {
		return err
	}
	cfg.SrcDir = cfgY.SrcDir

	// get directories that shall be excluded
	if len(cfgY.Excludes) > 0 {
		if err = cfg.getExcludes(&cfgY.Excludes); err != nil {
			return err
		}
	}

	// get number of CPU's (optional). Default is to use all available cpus
	if cfgY.NumCPUs == 0 {
		cfg.NumCpus = uint(runtime.NumCPU())
		log.Infof("num_cpus not configured. Use default: %d", cfg.NumCpus)
	} else {
		cfg.NumCpus = cfgY.NumCPUs
	}

	// get number of workers (optional). Per default it's set to the number of cpus
	if cfgY.NumWrkrs == 0 {
		cfg.NumWrkrs = cfg.NumCpus
		log.Infof("num_wrkrs not configured. Use default: %d", cfg.NumWrkrs)
	} else {
		cfg.NumWrkrs = cfgY.NumWrkrs
	}

	// get last sync time. If an initial sync was requested by the user (i.e.
	// init = true), nothing needs to be done)
	if !init {
		if cfg.LastSync, err = getLastSync(cfgY.LastSync); err != nil {
			return err
		}
	}

	// get rules
	var hasRule = false             // determine if there's at least one rule
	cfg.Cvs = make(map[string]*cvm) // allocate conversion map in config struct
	for i, r := range cfgY.Rules {
		var c *cvm

		c, err = cfg.getRule(&r, i+1)
		if err != nil {
			return err
		}
		cfg.Cvs[r.Source] = c
		hasRule = true
	}

	// raise error if no rules could be detected
	if !hasRule {
		log.Error("No conversion rules could be detected in config file")
		return fmt.Errorf("No conversion rules could be detected in config file")
	}

	// set target directory
	if cfg.TrgDir, err = os.Getwd(); err != nil {
		log.Errorf("Cannot determine working directory: %v", err)
		return fmt.Errorf("Cannot determine working directory: %v", err)
	}

	return nil
}

// getCv checks if the smsync conf contains a conversion rule for a given file.
// It does so by retrieving a cvm structure for that file path. In case it could be
// retrieved, a pointer to the cvm structure and true is returned, otherwise
// nil and false
func (cfg *Config) getCv(f string) (*cvm, bool) {
	if _, ok := cfg.Cvs[file.Suffix(f)]; ok {
		return cfg.Cvs[file.Suffix(f)], true
	}
	if _, ok := cfg.Cvs[suffixStar]; ok {
		return cfg.Cvs[suffixStar], true
	}
	return nil, false
}

// getExcludes expands the directories specified in the config file (which) can
// contain wildcards
func (cfg *Config) getExcludes(excls *[]string) error {
	log.Debug("smsync.Config.getExcludes: START")
	defer log.Debug("smsync.Config.getExcludes: END")

	for _, excl := range *excls {
		if excl == "" {
			continue
		}

		// expand directory
		a, err := filepath.Glob(filepath.Join(cfg.SrcDir, excl))
		if err != nil {
			return err
		}
		cfg.Excludes = append(cfg.Excludes, a...)
	}

	return nil
}

// getRule verifies that r represents a valid rule and create the
// corresponding mapping structure cvm
func (cfg *Config) getRule(r *rule, i int) (*cvm, error) {
	log.Debug("smsync.Config.getRule: START")
	defer log.Debug("smsync.Config.getRule: END")

	var (
		normCvStr string
		err       error
	)

	// check source suffix
	if len(r.Source) == 0 {
		log.Errorf("No source suffix in rule #%d", i)
		return nil, fmt.Errorf("No source suffix in rule #%d", i)
	}

	// get target suffix
	if len(r.Target) == 0 {
		log.Infof("Rule #%d: Since no target suffix could be detected, target suffix will be set to source suffix", i)
		r.Target = r.Source
	}

	// check conversion
	if len(r.Conversion) == 0 {
		log.Infof("Rule #%d: No conversion", i)
	}

	// check that conversion is copy or empty in case of suffix '*'.
	// if the conversion is empty it is set to copy.
	if r.Source == suffixStar && r.Conversion != cvCopyStr {
		if r.Conversion != "" {
			return nil, fmt.Errorf("Rule #%d: For suffix '*' only copy conversion is allowed", i)
		}
		r.Conversion = cvCopyStr
	}

	// in case of source suffix equals target suffix and empty conversion, the conversion is set to copy
	if (r.Source == r.Target) && r.Conversion == "" {
		log.Infof("Rule #%d: Since source equals target format without conversion, conversion is set to copy", i)
		r.Conversion = cvCopyStr
	}

	// check if either both suffices are '*' or both are not
	if (r.Source == suffixStar && r.Target != suffixStar) || (r.Source != suffixStar && r.Target == suffixStar) {
		log.Errorf("Rule #%d: Either both suffices need to be '*' or none", i)
		return nil, fmt.Errorf("Rule #%d: Either both suffices need to be '*' or none", i)
	}

	// check if conversion is supported
	if r.Conversion == cvCopyStr {
		if r.Source != r.Target {
			log.Errorf("Rule #%d: copy is only supported is source end target suffix are equal", i)
			return nil, fmt.Errorf("Rule #%d: copy is only supported is source end target suffix are equal", i)
		}
		return &cvm{TrgSuffix: r.Target, NormCvStr: cvCopyStr}, nil
	}

	if _, ok := validCvs[cvKey{r.Source, r.Target}]; !ok {
		log.Errorf("Rule #%d: conversion of '%s' into '%s' not supported", i, r.Source, r.Target)
		return nil, fmt.Errorf("Rule #%d: conversion of '%s' into '%s' not supported", i, r.Source, r.Target)
	}

	// validate conversion string and convert string to FFMpeg parameters
	if normCvStr, err = validCvs[cvKey{r.Source, r.Target}].normCvStr(r.Conversion); err != nil {
		log.Errorf("Rule #%d: '%s' is not a valid conversion", i, r.Conversion)
		return nil, fmt.Errorf("Rule #%d: '%s' is not a valid conversion", i, r.Conversion)
	}

	// validate that there's only one rule per source suffix
	if _, ok := cfg.Cvs[r.Source]; ok {
		log.Errorf("Rule #%d: There's already a rule for source suffix '%s'", i, r.Source)
		return nil, fmt.Errorf("Rule #%d: There's already a rule for source suffix '%s'", i, r.Source)
	}

	log.Infof("Rule #%d: '%s' is a valid conversion", i, r.Conversion)
	log.Infof("Rule #%d: Conversion string normalized to '%s'", i, normCvStr)
	return &cvm{TrgSuffix: r.Target, NormCvStr: normCvStr}, nil
}

// setProcEnd updates the file smsync.yaml after the conversions have ended
// successfully. It sets the last sync time and removes the "wip" (work in
// progress).
func (cfg *Config) setProcEnd() error {
	log.Debug("smsync.Config.setProcEnd: START")
	defer log.Debug("smsync.Config.setProcEnd: END")

	var (
		cfgY cfgYml
		err  error
	)

	// read config from file
	if err = cfgY.read(); err != nil {
		return err
	}

	// set last sync time to current time in UTC
	cfgY.LastSync = time.Now().UTC().Format(time.RFC3339)

	// adjust back processing status
	cfgY.ProcStat = ""

	// write config to file
	if err = cfgY.write(); err != nil {
		return err
	}

	log.Debug("Config.setProcEnd(): Config has been saved")

	return nil
}

// setProcStatWIP sets the processing status in the file smsync.yaml to
// "wip" (= work is progress). This status is valid as long as smsync is
// processing / converting files
func (cfg *Config) setProcStatWIP() error {
	log.Debug("smsync.Config.setProcStatWIP: START")
	defer log.Debug("smsync.Config.setProcStatWIP: END")

	var (
		cfgY cfgYml
		err  error
	)

	// read config from file
	if err = cfgY.read(); err != nil {
		return err
	}

	// adjust back processing status
	cfgY.ProcStat = procStatWIP

	// write config to file
	if err = cfgY.write(); err != nil {
		return err
	}

	log.Debug("Config.setProcStatWIP(): Config has been saved")

	return nil
}

// readCfg read the configuration from the file smsync.yaml in the current directory
func (cfgY *cfgYml) read() error {
	log.Debug("smsync.cfgYml.read: START")
	defer log.Debug("smsync.cfgYml.read: END")

	// read config file
	cfgFile, err := ioutil.ReadFile(filepath.Join(".", cfgFile))
	if err != nil {
		// determine working directory for error message
		wd, err0 := os.Getwd()
		if err0 != nil {
			log.Errorf("Cannot determine working directory: %v", err0)
			return fmt.Errorf("Cannot determine working directory: %v", err0)
		}
		log.Errorf("No configuration file found in '%s'", wd)
		return fmt.Errorf("No configuration file found in '%s'", wd)
	}
	if err = yaml.Unmarshal(cfgFile, &cfgY); err != nil {
		log.Errorf("Error during unmarshaling of config file: %v", err)
		return fmt.Errorf("Error during unmarshaling of config file: %v", err)
	}

	// clean directory names
	cfgY.SrcDir = path.Clean(cfgY.SrcDir)
	for i := range cfgY.Excludes {
		cfgY.Excludes[i] = path.Clean(cfgY.Excludes[i])
	}

	return nil
}

// write writes the configuration to the file smsync.yaml in the current directory
func (cfgY *cfgYml) write() error {
	log.Debug("smsync.cfgYml.write: START")
	defer log.Debug("smsync.cfgYml.write: END")

	var (
		out []byte
		err error
	)

	// turn config struct into a byte array
	if out, err = yaml.Marshal(&cfgY); err != nil {
		log.Errorf("Config struct could not be marshalled: %v", err)
		return fmt.Errorf("Config struct could not be marshalled: %v", err)
	}

	if err := ioutil.WriteFile(filepath.Join(".", cfgFile), out, 0777); err != nil {
		log.Errorf("Configuration file '%s' cannot be updated: %v", filepath.Join(".", cfgFile), err)
		return fmt.Errorf("Configuration file '%s' cannot be updated: %v", filepath.Join(".", cfgFile), err)
	}

	return nil
}

// checkDir checks if the source directory exists and if it's a directory
func checkDir(srcDir string) error {
	log.Debug("smsync.checkDir: START")
	defer log.Debug("smsync.checkDir: END")

	if len(srcDir) == 0 {
		log.Errorf("No source directory specified in config file")
		return fmt.Errorf("No source directory specified in config file")
	}
	fi, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Source directory '%s' doesn't exist", srcDir)
			return fmt.Errorf("Source directory '%s' doesn't exist", srcDir)
		}
		log.Errorf("Error regarding source directory '%s': %v", srcDir, err)
		return fmt.Errorf("Error regarding source directory '%s': %v", srcDir, err)
	}
	if !fi.IsDir() {
		log.Errorf("Source '%s' is no directory", srcDir)
		return fmt.Errorf("Source '%s' is no directory", srcDir)
	}

	return nil
}

// getLastSync determines the time of the last synchronization
func getLastSync(s string) (time.Time, error) {
	log.Debug("smsync.getLastSync: START")
	defer log.Debug("smsync.getLastSync: END")

	var (
		t   time.Time
		err error
	)

	if len(s) == 0 {
		log.Infof("No last sync time could be detected")
		return time.Time{}, nil
	}
	if t, err = time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	log.Errorf("Last sync time '%s' could not be parsed: %v", s, err)
	return time.Time{}, fmt.Errorf("Last sync time '%s' could not be parsed: %v", s, err)
}
