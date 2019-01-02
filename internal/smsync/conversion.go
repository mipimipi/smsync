// Copyright (C) 2018-2019 Michael Picht
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
	"regexp"
	"strconv"
	"time"

	"github.com/mipimipi/go-lhlp/file"
	log "github.com/sirupsen/logrus"
)

type (
	// conversion needs to be unique for a pair of source suffix and
	// target suffix
	cvKey struct {
		srcSuffix string
		trgSuffix string
	}

	// output structure of a conversion
	cvOutput struct {
		trgFile file.Info     // target file
		dur     time.Duration // duration of conversion
		err     error         // error (that occurred during the conversion)
	}

	// conversion interface
	conversion interface {
		// execute conversion
		exec(string, string, string) error

		// normalize the conversion string
		normCvStr(string) (string, error)
	}
)

// Constants for copy
const cvCopyStr = "copy"

// constants for bit rate options
const (
	abr  = "abr"  // average bitrate
	cbr  = "cbr"  // constant bitrate
	hcbr = "hcbr" // hard constant bitrate
	vbr  = "vbr"  // variable bitrate
)

// supported conversions
var (
	all2FLAC cvAll2FLAC // conversion of all types to FLAC
	all2MP3  cvAll2MP3  // conversion of all types to MP3
	all2OGG  cvAll2OGG  // conversion of all types to OGG
	all2OPUS cvAll2OPUS // conversion of all types to OPUS
	cp       cvCopy     // copy conversionn

	// validCvs maps conversion keys (i.e. pairs of source and target
	// suffices) to the supported conversions
	validCvs = map[cvKey]conversion{
		// valid conversions to FLAC
		cvKey{"flac", "flac"}: all2FLAC,
		cvKey{"wav", "flac"}:  all2FLAC,
		// valid conversions to MP3
		cvKey{"flac", "mp3"}: all2MP3,
		cvKey{"mp3", "mp3"}:  all2MP3,
		cvKey{"ogg", "mp3"}:  all2MP3,
		cvKey{"opus", "mp3"}: all2MP3,
		cvKey{"wav", "mp3"}:  all2MP3,
		// valid conversions to OGG
		cvKey{"flac", "ogg"}: all2OGG,
		cvKey{"mp3", "ogg"}:  all2OGG,
		cvKey{"ogg", "ogg"}:  all2OGG,
		cvKey{"opus", "ogg"}: all2OGG,
		cvKey{"wav", "ogg"}:  all2OGG,
		// valid conversions to OPUS
		cvKey{"flac", "opus"}: all2OPUS,
		cvKey{"mp3", "opus"}:  all2OPUS,
		cvKey{"ogg", "opus"}:  all2OPUS,
		cvKey{"opus", "opus"}: all2OPUS,
		cvKey{"wav", "opus"}:  all2OPUS,
		// copy
		cvKey{"*", "*"}: cp,
	}
)

// assembleTrgFile creates the target file path from the source file path
// (f) and the configuration
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
		trgSuffix = file.Suffix(srcFile)
	} else {
		// ... otherwise take target suffix from conversion rule
		trgSuffix = cvm.TrgSuffix
	}

	trgFile, err := file.PathRelCopy(cfg.SrcDir, file.PathTrunk(srcFile)+"."+trgSuffix, cfg.TrgDir)
	if err != nil {
		log.Errorf("Target path cannot be assembled: %v", err)
		return ""
	}
	return trgFile
}

// convert executes conversion for one file
func convert(cfg *Config, srcFile file.Info) cvOutput {
	var (
		trgFile string
		trgInfo file.Info
		cv      conversion
		err     error
	)

	// get conversion string for f from config
	cvm, exists := cfg.getCv(srcFile.Path())

	// if no string found: exit
	if !exists {
		log.Errorf("convert: No conversion found in config for '%s'", srcFile.Name())
		return cvOutput{trgFile: nil, dur: 0, err: nil}
	}

	// assemble output file
	trgFile = assembleTrgFile(cfg, srcFile.Path())

	// if error directory doesn't exist: create it
	if err := file.MkdirAll(filepath.Dir(trgFile), os.ModeDir|0755); err != nil {
		log.Errorf("convert: %v", err)
		return cvOutput{trgFile: nil, dur: 0, err: err}
	}

	// set transformation function
	if cvm.NormCvStr == cvCopyStr {
		cv = cp
	} else {
		// determine transformation function for srcSuffix -> trgSuffix
		cv = validCvs[cvKey{srcSuffix: file.Suffix(srcFile.Path()), trgSuffix: cvm.TrgSuffix}]
	}

	// execute conversion
	start := time.Now()
	err = cv.exec(srcFile.Path(), trgFile, cvm.NormCvStr)

	if err == nil {
		trgInfo, err = file.Stat(trgFile)
	}

	// call transformation function and return result
	return cvOutput{trgFile: trgInfo, dur: time.Since(start), err: err}
}

// isValidBitrate determines if s represents a valid bit rate. I.e. it needs
// be a 1-3-digit number, which is greater or equal than min and smaller or
// equal than max
func isValidBitrate(s string, min, max int) bool {
	var isValid bool

	if re, _ := regexp.Compile(`\d{1,3}`); re.FindString(s) != s {
		isValid = false
	} else {
		i, _ := strconv.Atoi(s)
		isValid = (min <= i && i <= max)
	}

	return isValid
}
