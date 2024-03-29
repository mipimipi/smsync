package smsync

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	s "gitlab.com/go-utilities/strings"
)

// implementation of interface "conversion" for conversions to MP3
type cvAll2MP3 struct{}

// exec executes the conversion to MP3
func (cv cvAll2MP3) exec(srcFile string, trgFile string, cvStr string) error {
	var params []string

	// set MP3 codec
	params = append(params, "-codec:a", "libmp3lame")

	a := s.SplitMulti(cvStr, "|:")

	switch a[0] {
	case abr:
		params = append(params, "-b:a", a[1]+"k", "-abr", "1")
	case cbr:
		params = append(params, "-b:a", a[1]+"k")
	case vbr:
		params = append(params, "-q:a", a[1])
	}

	// set compression level
	params = append(params, "-compression_level", a[3])

	//execute ffmpeg
	return execFFMPEG(srcFile, trgFile, &params)
}

// normCvStr normalizes the conversion string: Blanks are removed and default
// values are applied. In case the conversion string contains an invalid set
// of parameters, an error is returned.
func (cvAll2MP3) normCvStr(s string) (string, error) {
	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	var isValid = true

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
				case abr, cbr:
					if !isValidBitrate(b[1], 8, 500) {
						log.Errorf("'%s' is not a valid MP3 bit rate", b[1])
						isValid = false
					}
				case vbr:
					// check if b[1] is a valid MP3 VBR quality
					if re, _ := regexp.Compile(`\d{1}(.\d{1,3})?`); re.FindString(b[1]) != b[1] {
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
			if re, _ := regexp.Compile(`cl:\d{1}`); re.FindString(a[1]) != a[1] {
				log.Errorf("'%s' is not a valid MP3 quality", a[1])
				isValid = false
			}
		}
	}

	// conversion is not valid: error
	if !isValid {
		return "", fmt.Errorf("'%s' is not a valid MP3 conversion", s)
	}

	// everything's fine
	return s, nil
}
