// SPDX-FileCopyrightText: 2018-2020 Michael Picht <mipi@fsfe.org>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

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
