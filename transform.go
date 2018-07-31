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
	// transformation needs to by unique for a pair of source suffix and
	// destination suffix
	tfKey struct {
		srcSuffix string
		dstSuffix string
	}

	// input structure for transformation:
	tfInput struct {
		cfg *config // configuration
		f   string  // source file
	}

	// output structure of a transformation
	tfOutput struct {
		f   string // destination file
		err error  // error (that occurred during the transformation)
	}

	// transformation interface
	transformation interface {
		isValid(string) bool        // checks if s represents a valid transformation
		exec(*config, string) error // executes transformation
	}
)

// Constants for copy
const tfCopyStr = "copy"

// constants for bit rate options
const (
	abr = "abr" // average bit rate
	cbr = "cbr" // constant bit rate
	vbr = "vbr" // variable bit rate
)

// supported transformations
var (
	all2FLAC tfAll2FLAC // conversion of all types to FLAC
	all2MP3  tfAll2MP3  // conversion of all types to MP3
	all2OGG  tfAll2OGG  // conversion of all types to OGG
	cp       tfCopy     // copy transfromation

	// validTfs maps transformation keys (i.e. pairs of source and destination
	// suffices) to the supported transformations
	validTfs = map[tfKey]transformation{
		// valid conversions to FLAC
		tfKey{"flac", "flac"}: all2FLAC,
		tfKey{"wav", "flac"}:  all2FLAC,
		// valid conversions to MP3
		tfKey{"flac", "mp3"}: all2MP3,
		tfKey{"mp3", "mp3"}:  all2MP3,
		tfKey{"ogg", "mp3"}:  all2MP3,
		tfKey{"wav", "mp3"}:  all2MP3,
		// valid conversions to OGG
		tfKey{"flac", "ogg"}: all2OGG,
		tfKey{"mp3", "ogg"}:  all2OGG,
		tfKey{"ogg", "ogg"}:  all2OGG,
		tfKey{"wav", "ogg"}:  all2OGG,
		// copy
		tfKey{"*", "*"}: cp,
	}
)

// assembleDstFile creates the destination file path from the source file path
// (f) and the configuration
func assembleDstFile(cfg *config, srcFilePath string) (string, error) {
	var dstSuffix string

	// get transformation rule from config
	tfm, exists := cfg.getTf(srcFilePath)
	if !exists {
		log.Errorf("No transformation rule for '%s'", srcFilePath)
		return "", fmt.Errorf("No transformation rule for '%s'", srcFilePath)
	}

	// if corresponding transformation rule is for '*' ...
	if tfm.dstSuffix == suffixStar {
		// ... destination suffix is same as source suffix
		dstSuffix = lhlp.FileSuffix(srcFilePath)
	} else {
		// ... otherwise take destination suffix from transformation rule
		dstSuffix = tfm.dstSuffix
	}

	dstFilePath, err := lhlp.PathRelCopy(cfg.srcDirPath, lhlp.PathTrunk(srcFilePath)+"."+dstSuffix, cfg.dstDirPath)
	if err != nil {
		log.Errorf("Destination path cannot be assembled: %v", err)
		return "", err
	}
	return dstFilePath, nil
}

// transform executes transformation/conversion for one file
func transform(i tfInput) tfOutput {
	var tf transformation

	// get transformation string for f from config
	tfm, ok := i.cfg.getTf(i.f)
	// if no string found: exit
	if !ok {
		return tfOutput{"", nil}
	}

	// set transformation function
	if tfm.tfStr == tfCopyStr {
		tf = cp
	} else {
		// determine transformation function for srcSuffix -> dstSuffix
		tf = validTfs[tfKey{lhlp.FileSuffix(i.f), tfm.dstSuffix}]
	}

	// call transformation function and return result
	return tfOutput{i.f, tf.exec(i.cfg, i.f)}
}
