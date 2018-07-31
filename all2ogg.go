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

type tfAll2OGG struct{}

// isOGGBitrate checks if the input is a valid OGG bitrate (i.e. between
// 8 and 500 kbps)
func isOGGBitrate(s string) bool {
	var b = true

	if re, _ := regexp.Compile(`\d{1,3}`); re.FindString(s) != s {
		b = false
	} else {
		i, _ := strconv.Atoi(s)
		b = (8 <= i && i <= 500)
	}

	if !b {
		log.Errorf("'%s' is no a valid OGG bitrate", s)
	}

	return b
}

// isOGGVBRQuality checks if the input is a valid OGG VBR quality
// (i.e. s="X" with X = -1.0, ..., 10.0)
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

// isValid checks if s is a valid parameter string
func (tfAll2OGG) isValid(s string) bool {
	var b = true

	a := strings.Split(s, "|")

	if len(a) != 2 {
		b = false
	} else {
		switch a[0] {
		case abr:
			b = isOGGBitrate(a[1])
		case vbr:
			b = isOGGVBRQuality(a[1])
		default:
			b = false
		}
	}

	if b {
		log.Infof("'%s' is a valid OGG transformation", s)
	} else {
		log.Errorf("'%s' is not a valid OGG transformation", s)
	}

	return b
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG for OGG/VORBIS encoding, see
// http://ffmpeg.org/ffmpeg-codecs.html#libvorbis
func (tfAll2OGG) exec(cfg *config, f string) error {
	var args []string

	// assemble input file
	args = append(args, "-i", f)

	// only audio
	args = append(args, "-codec:a")

	// use vorbis codec
	args = append(args, "libvorbis")

	// assemble options
	{
		// split transformation string into array
		tf := strings.Split(cfg.tfs[path.Ext(f)[1:]].tfStr, "|")

		switch tf[0] {
		case abr:
			args = append(args, "-b", tf[1]+"k")
		case vbr:
			args = append(args, "-q:a", tf[1][1:])
		}
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
