// Command tau is the TauGo CLI. M0: --help only.
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	buildTimestamp = "dev" // set by `make build-reproducible`
	version        = "0.0.1-alpha"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "--version" {
		fmt.Printf("tau %s (build %s)\n", version, buildTimestamp)
		os.Exit(0)
	}
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `tau — TauGo kernel CLI (V0.1)

USAGE:
    tau <command> [flags]

COMMANDS:
    decide      Decide a regime for one exchange (M1+)
    calibrate   Run adaptive calibration on a corpus (M5+)
    --version   Print version

Specification: PRD.md
`)
	}
	flag.Parse()
	flag.Usage()
}
