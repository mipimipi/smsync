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
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	log "github.com/mipimipi/logrus"
)

type tfAll2FLAC struct{}

// isValid checks if s is a valid parameter string
func (tfAll2FLAC) isValid(s string) bool {
	var (
		b   = true
		i   int
		err error
	)

	// check if transformation parameter is like 'q{x}', where x is
	// 0, 1, ..., 12
	if re, _ := regexp.Compile(`q\d{1,2}`); re.FindString(s) != s {
		b = false
	} else {
		if i, err = strconv.Atoi(s[1:]); err != nil {
			b = false
		} else {
			if i < 0 || i > 12 {
				b = false
			}
		}
	}

	if b {
		log.Infof("'%s' is a valid FLAC transformation", s)
	} else {
		log.Errorf("'%s' is not a valid FLAC quality", s)
	}

	return b
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG for FLAC encoding, see
// http://ffmpeg.org/ffmpeg-codecs.html#flac-2
func (tfAll2FLAC) exec(cfg *config, f string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", f)

	// only audio
	args = append(args, "-codec:a")

	// use flac codec
	args = append(args, "flac")

	// assemble options
	{
		// split transformation string into array
		tf := strings.Split(cfg.tfs[path.Ext(f)[1:]].tfStr, "|")

		args = append(args, "-compression_level", tf[0][1:])
	}

	// overwrite output file (in case it's existing)
	args = append(args, "-y")

	// assemble output file
	dstFile, err := assembleDstFile(cfg, f)
	if err != nil {
		return err
	}
	args = append(args, dstFile)

	log.Debugf("FFmpeg command: ffmpeg %s", strings.Join(args, " "))

	// execute FFMPEG command
	if err := exec.Command("ffmpeg", args...).Run(); err != nil {
		log.Errorf("Executed FFMPEG for %s: %v", f, err)
		return err
	}

	return nil
}
