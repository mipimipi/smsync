package smsync

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	s "gitlab.com/go-utilities/strings"
)

// implementation of interface "conversion" for conversions to FLAC
type cvAll2FLAC struct{}

// exec executes the conversion to FLAC
func (cvAll2FLAC) exec(srcFile string, trgFile string, cvStr string) error {
	var params []string

	// set FLAC codec
	params = append(params, "-codec:a", "flac")

	// set compression level
	params = append(params, "-compression_level", s.SplitMulti(cvStr, "|:")[1])

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
