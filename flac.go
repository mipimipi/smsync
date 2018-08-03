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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/mipimipi/logrus"
)

// implementation of interface "conversion" for conversions to FLAC
type cvAll2FLAC struct{}

// exec executes the conversion to FLAC
func (cvAll2FLAC) exec(srcFile string, trgFile string, params *[]string) error {
	return execFFMPEG(srcFile, trgFile, params)
}

// translateParams converts the parameters string from config file into an array
// of ffmpeg parameters. Default values are applied. In case the parameter
// string contains an invalid set of parameter, an error is returned.
// In addition, a normalized (=enriched by default values) conversion string
// is returned
func (cvAll2FLAC) translateParams(s string) (*[]string, string, error) {
	var params []string

	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	// set FLAC codec
	params = append(params, "-codec:a", "flac")

	// if params string is empty, set default compression level (=5) and exit
	if s == "" {
		log.Infof("Set FLAC conversion to default: cl:5", s)
		params = append(params, "-compression_level", "5")
		return &params, "cl:5", nil
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
			return nil, "", fmt.Errorf("'%s' is not a valid FLAC conversion", s)
		}

		// everythings fine
		params = append(params, "-compression_level", s[3:])
		return &params, s, nil
	}
}
