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
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	log "github.com/mipimipi/logrus"
)

// implementation of of interface "conversion" for conversions to FLAC
type cvAll2FLAC struct{}

// normParams checks if the string contains valid conversion params and
// normalizes it (e.g. removes blanks and sets default values)
func (cvAll2FLAC) normParams(s *string) error {
	// set *s to lower case and remove blanks
	*s = strings.Trim(strings.ToLower(*s), " ")

	// if params string is empty, set default compression level (=5) and exit
	if *s == "" {
		*s = "cl:5"
		log.Infof("Set FLAC conversion to default: cl:5", *s)
		return nil
	}

	// handle more complex cases
	{
		var isValid = true

		// check if conversion parameter is like 'cl:X', where X is
		// 0, 1, ..., 12
		if re, _ := regexp.Compile(`cl:\d{1,2}`); re.FindString(*s) != *s {
			isValid = false
		} else {
			var (
				i   int
				err error
			)

			if i, err = strconv.Atoi((*s)[3:]); err != nil {
				isValid = false
			} else {
				if i < 0 || i > 12 {
					isValid = false
				}
			}
		}

		// conversion is not valid: error
		if !isValid {
			return fmt.Errorf("'%s' is not a valid FLAC conversion", *s)
		}

		// everythings fine
		return nil
	}
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG for FLAC encoding, see
// http://ffmpeg.org/ffmpeg-codecs.html#flac-2
func (cvAll2FLAC) exec(cfg *config, f string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", f)

	// only audio
	args = append(args, "-codec:a")

	// use flac codec
	args = append(args, "flac")

	// assemble options
	args = append(args, "-compression_level", cfg.cvs[path.Ext(f)[1:]].cvStr[3:])

	// overwrite output file (in case it's existing)
	args = append(args, "-y")

	// assemble output file
	trgFile, err := assembleTrgFile(cfg, f)
	if err != nil {
		return err
	}
	args = append(args, trgFile)

	log.Debugf("FFmpeg command: ffmpeg %s", strings.Join(args, " "))

	// execute FFMPEG command
	if err := exec.Command("ffmpeg", args...).Run(); err != nil {
		log.Errorf("Executed FFMPEG for %s: %v", f, err)
		return err
	}

	return nil
}
