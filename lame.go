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

	log "github.com/mipimipi/go-lazylog"
	lhlp "github.com/mipimipi/go-lhlp"
)

// constants for bit rate options
const (
	abr = "abr" // average bit rate
	cbr = "cbr" // constant bit rate
	vbr = "vbr" // variable bit rate
)

// isLameBitrate checks if the input is a valid LAME bitrate (i.e. 8, 16,
// 24, ..., 320)
func isLameBitrate(s string) bool {
	var b bool

	br := []int{8, 16, 24, 32, 40, 48, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}

	if re, _ := regexp.Compile(`\d{1,3}`); re.FindString(s) != s {
		b = false
	} else {
		i, _ := strconv.Atoi(s)
		b = lhlp.Contains(br, i)
	}

	if !b {
		log.Errorf("'%s' is no a valid LAME bitrate", s)
	}

	return b
}

// isLameQuality checks if the input is a valid LAME quality (i.e. "qX"
// with s="X" = 0,1, ..., 9)
func isLameQuality(s string) bool {
	if re, _ := regexp.Compile(`q\d{1}`); re.FindString(s) != s {
		log.Errorf("'%s' is no a valid LAME quality", s)
		return false
	}

	return true
}

// isLameVBRQuality checks if the input is a valid LAME VBR quality
// (i.e. s="vX" with X =0, ..., 9.999)
func isLameVBRQuality(s string) bool {
	if re, _ := regexp.Compile(`v\d{1}(.\d{1,3})?`); re.FindString(s) != s {
		log.Errorf("'%s' is no a valid LAME VBR quality", s)
		return false
	}

	return true
}

// isValidLameStr checks if s is a valid LAME parameter string
func isValidLameStr(s string) bool {
	var b bool

	a := strings.Split(s, "|")

	if len(a) < 2 || len(a) > 3 {
		b = false
	} else {
		switch a[0] {
		case abr, cbr:
			b = isLameBitrate(a[1]) && (len(a) < 3 || isLameQuality(a[2]))
		case vbr:
			b = isLameVBRQuality(a[1]) && (len(a) < 3 || isLameQuality(a[2]))
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

type tfLame struct{}

// isValid checks if s is a valid parameter string
func (tfLame) isValid(s string) bool {
	return isValidLameStr(s)
}

// exec assembles and executes the LAME command
func (tfLame) exec(cfg *config, f string) error {
	var args []string

	// assemble options
	{
		// split transformation string into array
		tf := strings.Split(cfg.tfs[path.Ext(f)[1:]].tfStr, "|")

		switch tf[0] {
		case abr:
			args = append(args, "--abr", tf[1])
		case cbr:
			args = append(args, "-b", tf[1])
		case vbr:
			args = append(args, "-V", tf[1][1:])
		}
		args = append(args, "-q", tf[2][1:])
	}

	// assemble input file
	args = append(args, f)

	// assemble output file
	args = append(args, assembleDstFile(cfg, f))

	log.Debugf("LAME command: lame %s", strings.Join(args, " "))

	// execute LAME command
	if err := exec.Command("lame", args...).Run(); err != nil {
		log.Errorf("Executed LAME for %s: %v", f, err)
		return err
	}

	return nil
}
