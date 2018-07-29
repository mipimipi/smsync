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

	lhlp "github.com/mipimipi/go-lhlp"
	log "github.com/mipimipi/logrus"
)

type tfAll2MP3 struct{}

// isMP3Bitrate checks if the input is a valid MP3 bitrate (i.e. 8, 16,
// 24, ..., 320)
func isMP3Bitrate(s string) bool {
	var b bool

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

// isMP3Quality checks if the input is a valid MP3 quality (i.e. "qX"
// with s="X" = 0,1, ..., 9)
func isMP3Quality(s string) bool {
	if re, _ := regexp.Compile(`q\d{1}`); re.FindString(s) != s {
		log.Errorf("'%s' is no a valid MP3 quality", s)
		return false
	}

	return true
}

// isMP3VBRQuality checks if the input is a valid MP3 VBR quality
// (i.e. s="vX" with X =0, ..., 9.999)
func isMP3VBRQuality(s string) bool {
	if re, _ := regexp.Compile(`v\d{1}(.\d{1,3})?`); re.FindString(s) != s {
		log.Errorf("'%s' is no a valid MP3 VBR quality", s)
		return false
	}

	return true
}

// isValid checks if s is a valid parameter string
func (tfAll2MP3) isValid(s string) bool {
	var b bool

	a := strings.Split(s, "|")

	if len(a) < 2 || len(a) > 3 {
		b = false
	} else {
		switch a[0] {
		case abr, cbr:
			b = isMP3Bitrate(a[1]) && (len(a) < 3 || isMP3Quality(a[2]))
		case vbr:
			b = isMP3VBRQuality(a[1]) && (len(a) < 3 || isMP3Quality(a[2]))
		default:
			b = false
		}
	}

	if b {
		log.Infof("'%s' is a valid transformation", s)
	} else {
		log.Errorf("'%s' is not a valid transformation", s)
	}

	return b
}

// exec assembles and executes the FFMPEG command. For details about the
// parameters of FFMPEG see https://trac.ffmpeg.org/wiki/Encode/MP3
func (tfAll2MP3) exec(cfg *config, f string) error {
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
