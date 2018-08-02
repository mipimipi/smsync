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

// implementation of of interface "conversion" for conversions to OGG (Vorbis)
type cvAll2OGG struct{}

// isOGGVBRQuality checks if the input is a valid OGG VBR quality
// (i.e. X = -1.0, ..., 10.0)
func isOGGVBRQuality(s string) bool {
	var b = true

	if re, _ := regexp.Compile(`[-+]?\d{1,2}.\d{1}?`); re.FindString(s) != s {
		b = false
	} else {
		f, _ := strconv.ParseFloat(s, 64)
		if f < -1.0 || f > 10.0 {
			b = false
		}
	}

	if !b {
		log.Errorf("'%s' is no a valid OGG VBR quality", s)
		return false

	}
	return true
}

// normParams checks if the string contains valid conversion Params and
// normalizes it (e.g. removes blanks and sets default values)
func (cvAll2OGG) normParams(s *string) error {
	// set *s to lower case and remove blanks
	*s = strings.Trim(strings.ToLower(*s), " ")

	// if params string is empty, set default compression level (=3.0) and exit
	if *s == "" {
		*s = "vbr:3.0"
		log.Infof("Set OGG conversion to default: vbr:3.0", *s)
		return nil
	}

	// handle more complex case
	{
		var isValid = true

		a := strings.Split(*s, ":")

		if len(a) != 2 {
			isValid = false
		} else {
			switch a[0] {
			case abr:
				//check if a[1] is a valid OGG bit rate
				if re, _ := regexp.Compile(`\d{1,3}`); re.FindString(a[1]) != a[1] {
					isValid = false
				} else {
					i, _ := strconv.Atoi(a[1])
					isValid = (8 <= i && i <= 500)
				}
				if !isValid {
					log.Errorf("'%s' is not a valid OGG bitrate", a[1])
				}
			case vbr:
				// check if a[1] is a valid OGG VBR quality
				if re, _ := regexp.Compile(`[-+]?\d{1,2}.\d{1}?`); re.FindString(a[1]) != a[1] {
					isValid = false
				} else {
					f, _ := strconv.ParseFloat(a[1], 64)
					if f < -1.0 || f > 10.0 {
						isValid = false
					}
				}
				if !isValid {
					log.Errorf("'%s' is not a valid OGG VBR quality", s)
				}
			default:
				isValid = false
			}
		}

		// conversion is not valid: error
		if !isValid {
			return fmt.Errorf("'%s' is not a valid OGG conversion", *s)
		}

		// everything's fine
		return nil
	}
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG for OGG/VORBIS encoding, see
// http://ffmpeg.org/ffmpeg-codecs.html#libvorbis
func (cvAll2OGG) exec(cfg *config, f string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", f)

	// only audio
	args = append(args, "-codec:a")

	// use vorbis codec
	args = append(args, "libvorbis")

	// assemble options
	{
		// split conversion string into array
		cv := strings.Split(cfg.cvs[path.Ext(f)[1:]].cvStr, "|")

		switch cv[0] {
		case abr:
			args = append(args, "-b", cv[1]+"k")
		case vbr:
			args = append(args, "-q:a", cv[1][1:])
		}
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
