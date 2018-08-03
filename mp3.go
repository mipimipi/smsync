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

// implementation of interface "conversion" for conversions to MP3
type cvAll2MP3 struct{}

// exec executes the conversion to FLAC
func (cvAll2MP3) exec(srcFile string, trgFile string, params *[]string) error {
	return execFFMPEG(srcFile, trgFile, params)
}

// translateParams converts the parameters string from config file into an array
// of ffmpeg parameters. Default values are applied. In case the parameter
// string contains an invalid set of parameter, an error is returned.
// In addition, a normalized (=enriched by default values) conversion string
// is returned
func (cvAll2MP3) translateParams(s string) (*[]string, string, error) {
	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	var (
		isValid = true
		params  []string
	)

	// set MP3 codec
	params = append(params, "-codec:a", "libmp3lame")

	a := strings.Split(s, "|")

	if len(a) != 2 {
		isValid = false
	} else {
		// check bit rate stuff
		{
			b := strings.Split(a[0], ":")

			if len(b) != 2 {
				isValid = false
			} else {
				switch b[0] {
				case abr:
					if isValidMP3Bitrate(b[1]) {
						params = append(params, "-b:a", b[1]+"k", "-abr", "1")
					} else {
						isValid = false
					}

				case cbr:
					if isValidMP3Bitrate(b[1]) {
						params = append(params, "-b:a", b[1]+"k")
					} else {
						isValid = false
					}
				case vbr:
					// check if b[1] is a valid MP3 VBR quality
					if re, _ := regexp.Compile(`\d{1}(.\d{1,3})?`); re.FindString(b[1]) == b[1] {
						params = append(params, "-q:a", b[1])
					} else {
						log.Errorf("'%s' is not a valid MP3 VBR quality", b[1])
						isValid = false
					}
				default:
					isValid = false
				}
			}
		}
		// check if a[1] is a valid compression level
		if isValid {
			if re, _ := regexp.Compile(`cl:\d{1}`); re.FindString(a[1]) == a[1] {
				params = append(params, "-compression_level", a[1][3:])
			} else {
				log.Errorf("'%s' is not a valid MP3 quality", a[1])
				isValid = false
			}
		}
	}

	// conversion is not valid: error
	if !isValid {
		return nil, "", fmt.Errorf("'%s' is not a valid MP3 conversion", s)
	}

	// everything's fine
	return &params, s, nil
}

// isValidMP3Bitrate checks if s represents a valid MP3 bit rate. If that's the
// case, true is returned, otherwise false.
func isValidMP3Bitrate(s string) bool {
	var isValid bool

	if re, _ := regexp.Compile(`\d{1,3}`); re.FindString(s) != s {
		isValid = false
	} else {
		i, _ := strconv.Atoi(s)
		isValid = (8 <= i && i <= 500)
	}
	if !isValid {
		log.Errorf("'%s' is not a valid MP3 bitrate", s)
	}
	return isValid
}
