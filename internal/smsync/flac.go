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

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gitlab.com/mipimipi/go-utils"

	log "github.com/sirupsen/logrus"
)

// implementation of interface "conversion" for conversions to FLAC
type cvAll2FLAC struct{}

// exec executes the conversion to FLAC
func (cvAll2FLAC) exec(srcFile string, trgFile string, cvStr string) error {
	var params []string

	// set FLAC codec
	params = append(params, "-codec:a", "flac")

	// set compression level
	params = append(params, "-compression_level", utils.SplitMulti(cvStr, "|:")[1])

	// execute ffmpeg
	return execFFMPEG(srcFile, trgFile, &params)
}

// normCvStr normalizes the conversion string: Blanks are removed and default
// values are applied. In case the conversion string contains an invalid set
// of parameters, an error is returned.
func (cvAll2FLAC) normCvStr(s string) (string, error) {
	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	// if params string is empty, set default compression level (=5) and exit
	if s == "" {
		log.Infof("Set FLAC conversion to default: cl:5")
		return "cl:5", nil
	}

	// handle more complex cases
	{
		var isValid = true

		// check if conversion parameter is like 'cl:X', where X is
		// 0, 1, ..., 12
		if re, _ := regexp.Compile(`cl:\d{1,2}`); re.FindString(s) != s {
			isValid = false
		} else {
			var (
				i   int
				err error
			)

			if i, err = strconv.Atoi((s)[3:]); err != nil {
				isValid = false
			} else {
				if i < 0 || i > 12 {
					isValid = false
				}
			}
		}

		// conversion is not valid: error
		if !isValid {
			return "", fmt.Errorf("'%s' is not a valid FLAC conversion", s)
		}

		// everythings fine
		return s, nil
	}
}
