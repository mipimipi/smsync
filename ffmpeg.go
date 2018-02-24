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
	"strings"

	log "github.com/mipimipi/go-lazylog"
)

type tfFFmpeg struct{}

// isValid checks if s is a valid parameter string. For FFMPEG the same
// parameters as for LAME are used
func (tfFFmpeg) isValid(s string) bool {
	return isValidLameStr(s)
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG see https://trac.ffmpeg.org/wiki/Encode/MP3
func (tfFFmpeg) exec(cfg *config, f string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", f)

	// only audio
	args = append(args, "-codec:a")

	// use lame
	args = append(args, "libmp3lame")

	// assemble options
	{
		// split transformation string into array
		tf := strings.Split(cfg.tfs[path.Ext(f)[1:]].tfStr, "|")

		switch tf[0] {
		case abr:
			args = append(args, "-b:a", tf[1]+"k", "-abr", "1")
		case cbr:
			args = append(args, "-b:a", tf[1]+"k")
		case vbr:
			args = append(args, "-q:a", tf[1][1:])
		}
		args = append(args, "-compression_level", tf[2][1:])
	}

	// overwrite output file (in case it's existing)
	args = append(args, "-y")

	// assemble output file
	args = append(args, assembleDstFile(cfg, f))

	log.Debugf("FFmpeg command: ffmpeg %s", strings.Join(args, " "))

	// execute FFMPEG command
	if err := exec.Command("ffmpeg", args...).Run(); err != nil {
		log.Errorf("Executed FFMPEG for %s: %v", f, err)
		return err
	}

	return nil
}
