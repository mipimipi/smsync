// Copyright (C) 2018 Michael Picht
//
// This file is part of go-lhlp (Go's little helper).
//
// go-lhlp is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-lhlp is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-lhlp. If not, see <http://www.gnu.org/licenses/>.

package lhlp

import "time"

// SplitDuration disaggregates a duration and returns it splitted into hours,
// minutes, seconds and nanoseconds
func SplitDuration(d time.Duration) map[time.Duration]time.Duration {
	var (
		out  = make(map[time.Duration]time.Duration)
		cmps = []time.Duration{time.Hour, time.Minute, time.Second, time.Nanosecond}
	)

	for _, cmp := range cmps {
		out[cmp] = d / cmp
		d -= out[cmp] * cmp
	}

	return out
}
