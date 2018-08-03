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

// implementation of interface "conversion" for conversions to OGG
type cvAll2OGG struct{}

// exec executes the conversion to OGG
func (cvAll2OGG) exec(srcFile string, trgFile string, params *[]string) error {
	return execFFMPEG(srcFile, trgFile, params)
}

// translateParams converts the parameters string from config file into an array
// of ffmpeg parameters. Default values are applied. In case the parameter
// string contains an invalid set of parameter, an error is returned.
// In addition, a normalized (=enriched by default values) conversion string
// is returned
func (cvAll2OGG) translateParams(s string) (*[]string, string, error) {
	var params []string

	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	// set vorbis codec
	params = append(params, "-codec:a", "libvorbis")

	// if params string is empty, set default compression level (=3.0) and exit
	if s == "" {
		log.Infof("Set OGG conversion to default: vbr:3.0", s)
		params = append(params, "-q:a", "3.0")
		return &params, "vbr:3.0", nil
	}

	// handle more complex case
	{
		var isValid = true

		a := strings.Split(s, ":")

		if len(a) != 2 {
			isValid = false
		} else {
			switch a[0] {
			case abr:
				//check if a[1] is a valid OGG bit rate
				if re, _ := regexp.Compile(`\d{1,3}`); re.FindString(a[1]) != a[1] {
					isValid = false
				} else {
					i, _ := strconv.Atoi(a[1])
					isValid = (8 <= i && i <= 500)
				}
				if !isValid {
					log.Errorf("'%s' is not a valid OGG bitrate", a[1])
				} else {
					params = append(params, "-b", a[1]+"k")
				}

			case vbr:
				// check if a[1] is a valid OGG VBR quality
				if re, _ := regexp.Compile(`[-+]?\d{1,2}.\d{1}?`); re.FindString(a[1]) != a[1] {
					isValid = false
				} else {
					f, _ := strconv.ParseFloat(a[1], 64)
					if f < -1.0 || f > 10.0 {
						isValid = false
					}
				}
				if !isValid {
					log.Errorf("'%s' is not a valid OGG VBR quality", s)
				} else {
					params = append(params, "-q:a", a[1])
				}
			default:
				isValid = false
			}
		}

		// conversion is not valid: error
		if !isValid {
			return nil, "", fmt.Errorf("'%s' is not a valid OGG conversion", s)
		}

		// everything's fine
		return &params, s, nil
	}
}
