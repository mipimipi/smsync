// SPDX-FileCopyrightText: 2018-2020 Michael Picht <mipi@fsfe.org>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package smsync

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.com/mipimipi/go-utils"
)

// implementation of interface "conversion" for conversions to OGG
type cvAll2OGG struct{}

// exec executes the conversion to OGG
func (cv cvAll2OGG) exec(srcFile string, trgFile string, cvStr string) error {
	var params []string

	// set vorbis codec
	params = append(params, "-codec:a", "libvorbis")

	a := utils.SplitMulti(cvStr, "|:")

	switch a[0] {
	case abr:
		params = append(params, "-b", a[1]+"k")
	case vbr:
		params = append(params, "-q:a", a[1])
	}

	//execute ffmpeg
	return execFFMPEG(srcFile, trgFile, &params)
}

// normCvStr normalizes the conversion string: Blanks are removed and default
// values are applied. In case the conversion string contains an invalid set
// of parameters, an error is returned.
func (cvAll2OGG) normCvStr(s string) (string, error) {
	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	// if params string is empty, set default compression level (=3.0) and exit
	if s == "" {
		log.Infof("Set OGG conversion to default: vbr:3.0")
		return "vbr:3.0", nil
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
				if !isValidBitrate(a[1], 8, 500) {
					log.Errorf("'%s' is not a valid OGG bitrate", a[1])
					isValid = false
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
				}
			default:
				isValid = false
			}
		}

		// conversion is not valid: error
		if !isValid {
			return "", fmt.Errorf("'%s' is not a valid OGG conversion", s)
		}

		// everything's fine
		return s, nil
	}
}
