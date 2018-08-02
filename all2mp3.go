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

// implementation of of interface "conversion" for conversions to MP3
type cvAll2MP3 struct{}

// normParams checks if the string contains valid conversion params and
// normalizes it (e.g. removes blanks and sets default values)
func (cvAll2MP3) normParams(s *string) error {
	// set *s to lower case and remove blanks
	*s = strings.Trim(strings.ToLower(*s), " ")

	var isValid = true

	a := strings.Split(*s, "|")

	if len(a) != 2 {
		isValid = false
	} else {
		// check bit rate stuff
		{
			b := strings.Split(a[0], ":")

			if len(b) != 2 {
				isValid = false
			} else {
				switch b[0] {
				case abr, cbr:
					//check if b[1] is a valid MP3 bit rate
					if re, _ := regexp.Compile(`\d{1,3}`); re.FindString(b[1]) != b[1] {
						isValid = false
					} else {
						i, _ := strconv.Atoi(b[1])
						isValid = (8 <= i && i <= 500)
					}
					if !isValid {
						log.Errorf("'%s' is not a valid MP3 bitrate", b[1])
					}
				case vbr:
					// check if b[1] is a valid MP3 VBR quality
					if re, _ := regexp.Compile(`\d{1}(.\d{1,3})?`); re.FindString(b[1]) != b[1] {
						log.Errorf("'%s' is not a valid MP3 VBR quality", b[1])
						isValid = false
					}
				default:
					isValid = false
				}
			}
		}
		// check if a[1] is a valid compression level
		if isValid {
			if re, _ := regexp.Compile(`cl:\d{1}`); re.FindString(a[1]) != a[1] {
				log.Errorf("'%s' is not a valid MP3 quality", a[1])
				isValid = false
			}
		}
	}

	// conversion is not valid: error
	if !isValid {
		return fmt.Errorf("'%s' is not a valid MP3 conversion", *s)
	}

	// everything's fine
	return nil
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG for MP3 encoding, see
// https://trac.ffmpeg.org/wiki/Encode/MP3
func (cvAll2MP3) exec(cfg *config, f string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", f)

	// only audio
	args = append(args, "-codec:a")

	// use mp3 codec
	args = append(args, "libmp3lame")

	// assemble options
	{
		a := strings.Split(cfg.cvs[path.Ext(f)[1:]].cvStr, "|")

		// assemble bitrate stuff
		{
			b := strings.Split(a[0], ":")

			switch b[0] {
			case abr:
				args = append(args, "-b:a", b[1]+"k", "-abr", "1")
			case cbr:
				args = append(args, "-b:a", b[1]+"k")
			case vbr:
				args = append(args, "-q:a", b[1])
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
