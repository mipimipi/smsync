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
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	lhlp "github.com/mipimipi/go-lhlp"
	log "github.com/mipimipi/logrus"
)

type (
	// conversion needs to be unique for a pair of source suffix and
	// target suffix
	cvKey struct {
		srcSuffix string
		trgSuffix string
	}

	// input structure of a conversion:
	cvInput struct {
		cfg *config // configuration
		f   string  // source file
	}

	// output structure of a conversion
	cvOutput struct {
		f   string // target file
		err error  // error (that occurred during the conversion)
	}

	// conversion interface
	conversion interface {
		// converts the parameters string from config file into an array of
		// ffmpeg parameters. Default values are applied. In case the parameter
		// string contains an invalid set of parameter, an error is returned.
		// In addition, a normalized (=enriched by default values) conversion
		// string is returned
		getParams(string) (*[]string, string, error)
	}

	// implementations interface "conversion" for different target formats
	cvAll2FLAC struct{} // FLAC
	cvAll2MP3  struct{} // MP3
	cvAll2OGG  struct{} // OGG (Vorbis)
	cvAll2OPUS struct{} // OPUS
	cvCopy     struct{} // simple file copy

)

// Constants for copy
const cvCopyStr = "copy"

// constants for bit rate options
const (
	abr  = "abr"  // average bit rate
	cbr  = "cbr"  // constant bit rate
	hcbr = "hcbr" // hard constant bit rate
	vbr  = "vbr"  // variable bit rate
)

// supported conversions
var (
	all2FLAC cvAll2FLAC // conversion of all types to FLAC
	all2MP3  cvAll2MP3  // conversion of all types to MP3
	all2OGG  cvAll2OGG  // conversion of all types to OGG
	all2OPUS cvAll2OPUS // conversion of all types to OPUS
	cp       cvCopy     // copy conversionn

	// validCvs maps conversion keys (i.e. pairs of source and target
	// suffices) to the supported conversions
	validCvs = map[cvKey]conversion{
		// valid conversions to FLAC
		cvKey{"flac", "flac"}: all2FLAC,
		cvKey{"wav", "flac"}:  all2FLAC,
		// valid conversions to MP3
		cvKey{"flac", "mp3"}: all2MP3,
		cvKey{"mp3", "mp3"}:  all2MP3,
		cvKey{"ogg", "mp3"}:  all2MP3,
		cvKey{"opus", "mp3"}: all2MP3,
		cvKey{"wav", "mp3"}:  all2MP3,
		// valid conversions to OGG
		cvKey{"flac", "ogg"}: all2OGG,
		cvKey{"mp3", "ogg"}:  all2OGG,
		cvKey{"ogg", "ogg"}:  all2OGG,
		cvKey{"opus", "ogg"}: all2OGG,
		cvKey{"wav", "ogg"}:  all2OGG,
		// valid conversions to OPUS
		cvKey{"flac", "opus"}: all2OPUS,
		cvKey{"mp3", "opus"}:  all2OPUS,
		cvKey{"ogg", "opus"}:  all2OPUS,
		cvKey{"opus", "opus"}: all2OPUS,
		cvKey{"wav", "opus"}:  all2OPUS,
		// copy
		cvKey{"*", "*"}: cp,
	}
)

// assembleTrgFile creates the target file path from the source file path
// (f) and the configuration
func assembleTrgFile(cfg *config, srcFilePath string) (string, error) {
	var trgSuffix string

	// get conversion rule from config
	cvm, exists := cfg.getCv(srcFilePath)
	if !exists {
		log.Errorf("No conversion rule for '%s'", srcFilePath)
		return "", fmt.Errorf("No conversion rule for '%s'", srcFilePath)
	}

	// if corresponding conversion rule is for '*' ...
	if cvm.trgSuffix == suffixStar {
		// ... target suffix is same as source suffix
		trgSuffix = lhlp.FileSuffix(srcFilePath)
	} else {
		// ... otherwise take target suffix from conversion rule
		trgSuffix = cvm.trgSuffix
	}

	trgFilePath, err := lhlp.PathRelCopy(cfg.srcDirPath, lhlp.PathTrunk(srcFilePath)+"."+trgSuffix, cfg.trgDirPath)
	if err != nil {
		log.Errorf("Target path cannot be assembled: %v", err)
		return "", err
	}
	return trgFilePath, nil
}

// convert executes conversion for one file
func convert(i cvInput) cvOutput {
	var args []string

	// get conversion string for f from config
	cvm, ok := i.cfg.getCv(i.f)
	// if no string found: exit
	if !ok {
		return cvOutput{"", nil}
	}

	// assemble output file
	trgFile, err := assembleTrgFile(i.cfg, i.f)
	if err != nil {
		return cvOutput{"", err}
	}

	// execute conversion
	if cvm.normCvStr == cvCopyStr {
		// copy
		return cvOutput{trgFile, lhlp.CopyFile(i.f, trgFile)}
	}

	// assemble input file
	args = append(args, "-i", i.f)

	// add conversion-specific parameters
	args = append(args, *cvm.params...)

	// overwrite output file (in case it's existing)
	args = append(args, "-y")

	// assemble output file
	trgFile, err = assembleTrgFile(i.cfg, i.f)
	if err != nil {
		return cvOutput{"", err}
	}
	args = append(args, trgFile)

	log.Debugf("FFmpeg command: ffmpeg %s", strings.Join(args, " "))

	// execute FFMPEG command
	if err := exec.Command("ffmpeg", args...).Run(); err != nil {
		log.Errorf("Executed FFMPEG for %s: %v", i.f, err)
		return cvOutput{trgFile, err}
	}

	// everything's fine
	return cvOutput{trgFile, nil}
}

// getParams converts the parameters string from config file into an array
// of ffmpeg parameters. Default values are applied. In case the parameter
// string contains an invalid set of parameter, an error is returned.
// In addition, a normalized (=enriched by default values) conversion string
// is returned
func (cvAll2FLAC) getParams(s string) (*[]string, string, error) {
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

// getParams converts the parameters string from config file into an array
// of ffmpeg parameters. Default values are applied. In case the parameter
// string contains an invalid set of parameter, an error is returned.
// In addition, a normalized (=enriched by default values) conversion string
// is returned
func (cvAll2MP3) getParams(s string) (*[]string, string, error) {
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

// getParams converts the parameters string from config file into an array
// of ffmpeg parameters. Default values are applied. In case the parameter
// string contains an invalid set of parameter, an error is returned.
// In addition, a normalized (=enriched by default values) conversion string
// is returned
func (cvAll2OGG) getParams(s string) (*[]string, string, error) {
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

// getParams converts the parameters string from config file into an array
// of ffmpeg parameters. Default values are applied. In case the parameter
// string contains an invalid set of parameter, an error is returned.
// In addition, a normalized (=enriched by default values) conversion string
// is returned
func (cvAll2OPUS) getParams(s string) (*[]string, string, error) {
	// set ss to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	var (
		isValid = true
		params  []string
	)

	// set OPUS codec
	params = append(params, "-codec:a", "libopus")

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
			isValid = b[0] == "abr" || b[0] == "cbr" || b[0] == "hcbr"

			if isValid {
				var (
					i   int
					err error
				)
				if i, err = strconv.Atoi(b[1]); err != nil {
					isValid = false
				} else {
					if i < 6 || i > 510 {
						isValid = false
					}
				}

				if isValid {
					// set bit rate
					params = append(params, "-b:a", b[1]+"k")

					// set vbr type
					switch b[0] {
					case abr:
						params = append(params, "-vbr", "on")
					case cbr:
						params = append(params, "-vbr", "off")
					case hcbr:
						params = append(params, "-vbr", "constrained")
					}

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
				params = append(params, "-compression_level", "10")
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
				if isValid {
					// set compression level
					params = append(params, "-compression_level", a[1][3:])
				}
			}
		}
	}

	// conversion is not valid: error
	if !isValid {
		return nil, "", fmt.Errorf("'%s' is not a valid OPUS conversion", s)
	}

	// everything's fine
	return &params, s, nil
}

// getParams checks if the parameters string from config file is either empty
// or equals "copy". If that's the case, the resulting array is ["copy"].
// Otherwise an error is returned.
func (cvCopy) getParams(s string) (*[]string, string, error) {
	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	if s != cvCopyStr {
		if s == "" {
			s = cvCopyStr
		} else {
			return nil, "", fmt.Errorf("'%s' is not a valid copy conversion", s)
		}
	}
	return &([]string{cvCopyStr}), cvCopyStr, nil
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
