package smsync

import (
	"fmt"
	"strings"

	"gitlab.com/go-utilities/file"
)

// implementation of interface "conversion" for simple file copy
type cvCopy struct{}

// exec executes simple file copy
func (cvCopy) exec(srcFile string, trgFile string, cvStr string) error {
	return file.Copy(srcFile, trgFile)
}

// normCvStr checks if the parameters string from config file is either empty
// or equals "copy". If that's the case, "copy" is returned. Otherwise an error
// is returned.
func (cvCopy) normCvStr(s string) (string, error) {
	// set s to lower case and remove blanks
	s = strings.Trim(strings.ToLower(s), " ")

	if s != cvCopyStr && s != "" {
		return "", fmt.Errorf("'%s' is not a valid copy conversion", s)
	}
	return cvCopyStr, nil
}
