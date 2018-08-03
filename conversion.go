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

import (
	"fmt"

	lhlp "github.com/mipimipi/go-lhlp"
	log "github.com/mipimipi/logrus"
)

type (
	// conversion needs to be unique for a pair of source suffix and
	// target suffix
	cvKey struct {
		srcSuffix string
		trgSuffix string
	}

	// input structure of a conversion:
	cvInput struct {
		cfg *config // configuration
		f   string  // source file
	}

	// output structure of a conversion
	cvOutput struct {
		f   string // target file
		err error  // error (that occurred during the conversion)
	}

	// conversion interface
	conversion interface {
		// execute conversion
		exec(string, string, *[]string) error

		// translates the parameters string from config file into an array of
		// parameters that is needed by the function that is used for the file
		// conversion (that's in most cases FFMPEG). Default values are applied.
		// In case the parameter string contains an invalid set of parameter,
		// an error is returned.
		// In addition, a normalized (=enriched by default values) conversion
		// string is returned
		translateParams(string) (*[]string, string, error)
	}
)

// Constants for copy
const cvCopyStr = "copy"

// constants for bit rate options
const (
	abr  = "abr"  // average bit rate
	cbr  = "cbr"  // constant bit rate
	hcbr = "hcbr" // hard constant bit rate
	vbr  = "vbr"  // variable bit rate
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
func assembleTrgFile(cfg *config, srcFilePath string) (string, error) {
	var trgSuffix string

	// get conversion rule from config
	cvm, exists := cfg.getCv(srcFilePath)
	if !exists {
		log.Errorf("No conversion rule for '%s'", srcFilePath)
		return "", fmt.Errorf("No conversion rule for '%s'", srcFilePath)
	}

	// if corresponding conversion rule is for '*' ...
	if cvm.trgSuffix == suffixStar {
		// ... target suffix is same as source suffix
		trgSuffix = lhlp.FileSuffix(srcFilePath)
	} else {
		// ... otherwise take target suffix from conversion rule
		trgSuffix = cvm.trgSuffix
	}

	trgFilePath, err := lhlp.PathRelCopy(cfg.srcDirPath, lhlp.PathTrunk(srcFilePath)+"."+trgSuffix, cfg.trgDirPath)
	if err != nil {
		log.Errorf("Target path cannot be assembled: %v", err)
		return "", err
	}
	return trgFilePath, nil
}

// convert executes conversion for one file
func convert(i cvInput) cvOutput {
	// get conversion string for f from config
	cvm, ok := i.cfg.getCv(i.f)

	// if no string found: exit
	if !ok {
		return cvOutput{"", nil}
	}

	// assemble output file
	trgFile, err := assembleTrgFile(i.cfg, i.f)
	if err != nil {
		return cvOutput{"", err}
	}

	var cv conversion

	// set transformation function
	if cvm.normCvStr == cvCopyStr {
		cv = cp
	} else {
		// determine transformation function for srcSuffix -> trgSuffix
		cv = validCvs[cvKey{lhlp.FileSuffix(i.f), cvm.trgSuffix}]
	}

	// call transformation function and return result
	return cvOutput{trgFile, cv.exec(i.f, trgFile, cvm.params)}
}
