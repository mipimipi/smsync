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
	"strconv"
	"strings"

	log "github.com/mipimipi/logrus"
)

type cvAll2OPUS struct{}

// normParams checks if the string contains a valid set of parameters and
// normalizes it (e.g. removes blanks and sets default values)
func (cvAll2OPUS) normParams(s *string) error {
	// set *s to lower case and remove blanks
	*s = strings.Trim(strings.ToLower(*s), " ")

	var isValid = true

	a := strings.Split(*s, "|")

	if len(a) == 0 || len(a) > 2 {
		isValid = false
	} else {
		if len(a) == 1 {
			*s += "|cl:10"
		}
		// check bit rate stuff
		{
			b := strings.Split(a[0], ":")

			if len(b) != 2 {
				isValid = false
			} else {
				isValid = b[0] == "abr" || b[0] == "cbr" || b[0] == "hcbr"

				if isValid {
					var (
						i   int
						err error
					)
					if i, err = strconv.Atoi(b[1]); err != nil {
						isValid = false
					} else {
						if i < 6 || i > 510 {
							isValid = false
						}
					}
				}
			}
		}
		// check compression level
		if isValid && len(a) == 2 {
			var (
				i   int
				err error
			)
			if i, err = strconv.Atoi(a[1][3:]); err != nil {
				isValid = false
			} else {
				if i < 0 || i > 10 {
					isValid = false
				}
			}
		}
	}

	// conversion is not valid: error
	if !isValid {
		return fmt.Errorf("'%s' is not a valid OPUS conversion", *s)
	}

	// everything's fine
	return nil
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG for OPUS encoding, see
// http://ffmpeg.org/ffmpeg-codecs.html#libopus-1
func (cvAll2OPUS) exec(cfg *config, f string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", f)

	// only audio
	args = append(args, "-codec:a")

	// use mp3 codec
	args = append(args, "libopus")

	// assemble options
	{
		a := strings.Split(cfg.cvs[path.Ext(f)[1:]].cvStr, "|")

		// assemble bitrate stuff
		{
			b := strings.Split(a[0], ":")

			args = append(args, "-b:a", b[1]+"k")

			switch b[0] {
			case abr:
				args = append(args, "-vbr", "on")
			case cbr:
				args = append(args, "-vbr", "off")
			case hcbr:
				args = append(args, "-vbr", "constrained")
			}
		}
		// assemble compression level
		args = append(args, "-compression_level", a[1][3:])
	}

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
