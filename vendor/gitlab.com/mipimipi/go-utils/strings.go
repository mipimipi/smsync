// Copyright (C) 2019 Michael Picht
//
// This file is part of go-utils (Go utilities).
//
// go-utils is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-utils is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-utils. If not, see <http://www.gnu.org/licenses/>.

package utils

import "strings"

// SplitMulti slices s into all substrings separated by any character of sep
// and returns a slice of the substrings between those separators.
// If s does not contain any character of sep and sep is not empty, SplitMulti
// returns a slice of length 1 whose only element is s.
// If sep is empty, SplitMulti splits after each UTF-8 sequence. If both s and
// sep are empty, SplitMulti returns an empty slice.
func SplitMulti(s string, sep string) []string {
	var a []string

	// handle special cases: if sep is empty ...
	if len(sep) == 0 {
		//... and if s is empty: return an empty slice
		if len(s) == 0 {
			return a
		}
		// ... else split after each character
		return strings.Split(s, "")
	}

	// split s by the characters of sep
	for i, j := -1, 0; j <= len(s); j++ {
		if j == len(s) || strings.Contains(sep, string(s[j])) {
			if i+1 > j-1 {
				a = append(a, "")
			} else {
				a = append(a, s[i+1:j])
			}
			i = j
		}
	}

	// if s does not contain any charachter of sep: return a slice that only
	// contains s
	if len(a) == 0 {
		a = append(a, s)
	}

	return a
}
