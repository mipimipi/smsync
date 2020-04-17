// SPDX-FileCopyrightText: 2018-2020 Michael Picht <mipi@fsfe.org>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"fmt"
	. "github.com/logrusorgru/aurora"
)

// PrintPlainln prints a plain text
func PrintPlainln(format string, a ...interface{}) {
	_, _ = fmt.Printf("    %s\n", Bold(fmt.Sprintf(format, a...)))
}

// PrintInfoln prints an info text
func PrintInfoln(format string, a ...interface{}) {
	_, _ = fmt.Printf("%s %s\n", Bold(BrightGreen("==>")), Bold(Sprintf(format, a...)))
}

// PrintMsgln prints a message
func PrintMsgln(format string, a ...interface{}) {
	_, _ = fmt.Printf("%s %s\n", Bold(BrightCyan("==>")), Bold(Sprintf(format, a...)))
}

// PrintWarnln prints a warning
func PrintWarnln(format string, a ...interface{}) {
	_, _ = fmt.Printf("%s %s\n", Bold(BrightYellow("==> WARNING:")), Bold(Sprintf(format, a...)))
}

// PrintErrorln prints an error
func PrintErrorln(format string, a ...interface{}) {
	_, _ = fmt.Printf("%s %s\n", Bold(BrightRed("==> ERROR:")), Bold(Sprintf(format, a...)))
}
