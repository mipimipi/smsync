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

package smsync

// ffmpeg.go contains coding that is specific to the command line tool ffmpeg,
// esp. the call to ffmpeg

import (
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// execFFMPEG calls ffmpeg to convert srcFile to trgFile using the
// conversion-specific parameters *params
func execFFMPEG(srcFile string, trgFile string, params *[]string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", srcFile)

	// add conversion-specific parameters
	args = append(args, *params...)

	// overwrite output file (in case it's existing)
	args = append(args, "-y")

	// add target file
	args = append(args, trgFile)

	log.Debugf("FFmpeg command: ffmpeg %s", strings.Join(args, " "))

	// execute FFMPEG command
	if err := exec.Command("ffmpeg", args...).Run(); err != nil { // nolint
		log.Errorf("Executed FFMPEG for %s: %v", srcFile, err)
		return err
	}

	// everything's fine
	return nil
}
