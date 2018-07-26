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

	log "github.com/mipimipi/logrus"
)

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
	dstFile, err := assembleDstFile(cfg, f)
	if err != nil {
		return err
	}
	args = append(args, dstFile)

	log.Debugf("LAME command: lame %s", strings.Join(args, " "))

	// execute LAME command
	if err := exec.Command("lame", args...).Run(); err != nil {
		log.Errorf("Executed LAME for %s: %v", f, err)
		return err
	}

	return nil
}
