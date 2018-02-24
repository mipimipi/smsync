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
	log "github.com/mipimipi/go-lazylog"
	lhlp "github.com/mipimipi/go-lhlp"
)

type (
	// transformation needs to by unique for a pair of sourc suffix and
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

// supported transformations
var (
	lame   tfLame   // LAME transformation
	ffmpeg tfFFmpeg // FFMPEG transformation
	cp     tfCopy   // copy transfromation

	// validTfs maps transformation keys (i.e. pairs of source and destination
	// suffices) to the supported transformations
	validTfs = map[tfKey]transformation{
		tfKey{"flac", "mp3"}: ffmpeg,
		tfKey{"mp3", "mp3"}:  lame,
		tfKey{"*", "*"}:      cp,
	}
)

// assembleDstFile creates the destination file path from the source file path
// (f) and the configuration
func assembleDstFile(cfg *config, srcFilePath string) string {
	var dstSuffix string

	// get transformation rule from config
	tfm, exists := cfg.getTf(srcFilePath)
	if !exists {
		panic("No transformation rule for " + srcFilePath)
	}

	// if corresponding transformation rule is for '*' ...
	if tfm.dstSuffix == suffixStar {
		// ... destination suffix is same as source suffix
		dstSuffix = lhlp.FileSuffix(srcFilePath)
	} else {
		// otherwise take destination suffix from transformation rule
		dstSuffix = tfm.dstSuffix
	}

	dstFilePath, err := lhlp.PathRelCopy(cfg.srcDirPath, lhlp.PathTrunk(srcFilePath)+"."+dstSuffix, cfg.dstDirPath)
	if err != nil {
		log.Errorf("%v", err)
		return ""
	}
	return dstFilePath
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
