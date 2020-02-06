package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// Version stores version information. It's filled by make (see Makefile)
var Version string

func main() {
	log.Debug("cli.main: START")
	defer log.Debug("cli.main: END")

	if err := execute(); err != nil {
		if _, e := fmt.Fprintln(os.Stderr, err); e != nil {
			panic(e.Error())
		}
		os.Exit(1)
	}
}
