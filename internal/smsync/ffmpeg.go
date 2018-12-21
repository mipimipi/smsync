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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mipimipi/go-lhlp"

	log "github.com/sirupsen/logrus"
)

// execFFMPEG calls ffmpeg to convert srcFile to trgFile using the
// conversion-specific parameters *params
func execFFMPEG(srcFile string, trgFile string, params *[]string) error {
	var (
		args []string // arguments for FFMPEG
		out  []byte   // to capture output of FFMPEG
		err  error
	)

	// add input file
	args = append(args, "-i", srcFile)

	// add conversion-specific parameters
	args = append(args, *params...)

	// overwrite output file (in case it's existing)
	args = append(args, "-y")

	// set logging
	args = append(args, "-loglevel", "repeat+level+verbose")

	// add target file
	args = append(args, trgFile)

	log.Debugf("FFmpeg command: ffmpeg %s", strings.Join(args, " "))

	// execute FFMPEG command
	cmd := exec.Command("ffmpeg", args...)
	if out, err = cmd.CombinedOutput(); err != nil { // nolint
		log.Errorf("Executed FFMPEG for %s: %v", srcFile, err)

		// if error directory doesn't exist: create it
		if _, e0 := os.Stat(filepath.Join(".", errDir)); os.IsNotExist(e0) {
			if e1 := os.Mkdir(filepath.Join(".", errDir), os.ModeDir|0755); e1 != nil {
				log.Errorf("Error from Mkdir('%s'): %v", errDir, e1)
			}
		}
		// assemble error file name
		errFile := "smsync.err/" + filepath.Base(lhlp.PathTrunk(trgFile)) + ".log"
		// write stdout into error file
		if e := ioutil.WriteFile(errFile, out, 0644); e != nil {
			log.Errorf("Couldn't write FFMPEG error file '%s's: %v", errFile, e)
		}

		return err
	}

	// everything's fine
	return nil
}
