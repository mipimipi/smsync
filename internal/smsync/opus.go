// SPDX-FileCopyrightText: 2018-2020 Michael Picht <mipi@fsfe.org>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package smsync

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.com/mipimipi/go-utils"
)

// implementation of interface "conversion" for conversions to OPUS
type cvAll2OPUS struct{}

// exec executes the conversion to OPUS
func (cv cvAll2OPUS) exec(srcFile string, trgFile string, cvStr string) error {
	var params []string

	// set OPUS codec
	params = append(params, "-codec:a", "libopus")

	a := utils.SplitMulti(cvStr, "|:")

	// set bit rate
	params = append(params, "-b:a", a[1]+"k")

	// set vbr type
	switch a[0] {
	case vbr:
		params = append(params, "-vbr", "on")
	case cbr:
		params = append(params, "-vbr", "off")
	case hcbr:
		params = append(params, "-vbr", "constrained")
	}

	// set compression level
	params = append(params, "-compression_level", a[3])

	// execute ffmpeg
	return execFFMPEG(srcFile, trgFile, &params)
}

// normCvStr normalizes the conversion string: Blanks are removed and default
// values are applied. In case the conversion string contains an invalid set
// of parameters, an error is returned.
func (cvAll2OPUS) normCvStr(s string) (string, error) {
	// set ss to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	var isValid bool

	a := strings.Split(s, "|")

	// there must either be one or tow parameters
	if len(a) == 0 || len(a) > 2 {
		isValid = false
	} else {
		// check bit rate stuff
		b := strings.Split(a[0], ":")

		if len(b) != 2 {
			isValid = false
		} else {
			isValid = b[0] == "vbr" || b[0] == "cbr" || b[0] == "hcbr"

			if isValid {
				if !isValidBitrate(b[1], 6, 510) {
					log.Errorf("'%s' is not a valid OPUS bit rate", b[1])
					isValid = false
				}

			}
		}

		// check compression level stuff
		if isValid {
			// if params string doesn't contain compression level:
			// set level to default
			if len(a) == 1 {
				// set default compression level
				log.Infof("Set OPUS compression level to default: cl:10")
				s += "|cl:10"
			} else {
				var (
					i   int
					err error
				)
				if i, err = strconv.Atoi(a[1][3:]); err != nil {
					isValid = false
				} else {
					if i < 0 || i > 10 {
						isValid = false
					}
				}
			}
		}
	}

	// conversion is not valid: error
	if !isValid {
		return "", fmt.Errorf("'%s' is not a valid OPUS conversion", s)
	}

	// everything's fine
	return s, nil
}
