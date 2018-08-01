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

	lhlp "github.com/mipimipi/go-lhlp"
	log "github.com/mipimipi/logrus"
)

type tfAll2MP3 struct{}

// isMP3Bitrate checks if the input is a valid MP3 bitrate (i.e. 8, 16,
// 24, ..., 320 kbps)
func isMP3Bitrate(s string) bool {
	var b = true

	br := []int{8, 16, 24, 32, 40, 48, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}

	if re, _ := regexp.Compile(`\d{1,3}`); re.FindString(s) != s {
		b = false
	} else {
		i, _ := strconv.Atoi(s)
		b = lhlp.Contains(br, i)
	}

	if !b {
		log.Errorf("'%s' is no a valid MP3 bitrate", s)
	}

	return b
}

// isMP3CompLevel checks if the input is a valid MP3 compression_level
// (i.e. "cl:X" with X = 0,1, ..., 9)
func isMP3CompLevel(s string) bool {
	if re, _ := regexp.Compile(`\d{1}`); re.FindString(s) != s {
		log.Errorf("'%s' is no a valid MP3 quality", s)
		return false
	}

	return true
}

// isMP3VBRQuality checks if the input is a valid MP3 VBR quality
// (i.e. s = 0, ..., 9.999)
func isMP3VBRQuality(s string) bool {
	if re, _ := regexp.Compile(`\d{1}(.\d{1,3})?`); re.FindString(s) != s {
		log.Errorf("'%s' is no a valid MP3 VBR quality", s)
		return false
	}

	return true
}

// normParams checks if the string contains a valid set of parameters and
// normalizes it (e.g. removes blanks and sets default values)
func (tfAll2MP3) normParams(s *string) error {
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
					isValid = isMP3Bitrate(b[1])
				case vbr:
					isValid = isMP3VBRQuality(b[1])
				default:
					isValid = false
				}
			}
		}
		// check compression level
		if isValid {
			isValid = isMP3CompLevel(a[1][3:])
		}
	}

	if !isValid {
		log.Errorf("'%s' is not a valid MP3 transformation", *s)
		return fmt.Errorf("'%s' is not a valid MP3 transformation", *s)
	}

	log.Infof("'%s' is a valid MP3 transformation", *s)
	return nil
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG for MP3 encoding, see
// https://trac.ffmpeg.org/wiki/Encode/MP3
func (tfAll2MP3) exec(cfg *config, f string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", f)

	// only audio
	args = append(args, "-codec:a")

	// use mp3 codec
	args = append(args, "libmp3lame")

	// assemble options
	{
		a := strings.Split(cfg.tfs[path.Ext(f)[1:]].tfStr, "|")

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
