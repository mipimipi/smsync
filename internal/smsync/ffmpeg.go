// Copyright (C) 2018-2019 Michael Picht
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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mipimipi/go-lhlp/file"

	log "github.com/sirupsen/logrus"
)

// execFFMPEG calls ffmpeg to convert srcFile to trgFile using the
// conversion-specific parameters *params
func execFFMPEG(srcFile string, trgFile string, params *[]string) error {
	var args []string // arguments for FFMPEG

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
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil { // nolint
		log.Errorf("Executed FFMPEG for %s: %v", srcFile, err)
		log.Errorf("FFmpeg command: ffmpeg %s", strings.Join(args, " "))

		// if error directory doesn't exist: create it
		if e := file.MkdirAll(filepath.Join(".", errDir), os.ModeDir|0755); e != nil {
			log.Errorf("Error from MkdirAll('%s'): %v", errDir, e)
		}

		// assemble error file name
		errFile := filepath.Join(errDir, filepath.Base(file.PathTrunk(trgFile))) + ".log"
		// write stdout into error file
		if e := ioutil.WriteFile(errFile, out, 0644); e != nil {
			log.Errorf("Couldn't write FFMPEG error file '%s's: %v", errFile, e)
		}

		return fmt.Errorf("Error during execution of FFMPEG: %v", err)
	}

	// everything's fine
	return nil
}
