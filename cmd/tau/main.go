// Command tau is the TauGo CLI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/tau"
)

var (
	buildTimestamp = "dev" //nolint:gochecknoglobals // build-time variable set via -ldflags; cannot be const
	version        = "0.0.2-alpha"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-version":
			fmt.Printf("tau %s (build %s)\n", version, buildTimestamp)
			os.Exit(0)
		case "decide":
			runDecide()
			return
		case "calibrate":
			runCalibrate(os.Args[2:])
			return
		}
	}
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `tau — TauGo kernel CLI (V0.1)

USAGE:
    tau <command> [flags]

COMMANDS:
    decide      Decide a regime for one exchange (reads JSON Exchange on stdin)
    calibrate   Run adaptive calibration on a corpus (M5+)
    --version   Print version

Specification: PRD.md
`)
	}
	flag.Parse()
	flag.Usage()
	os.Exit(1)
}

func runDecide() {
	var x tau.Exchange
	if err := json.NewDecoder(os.Stdin).Decode(&x); err != nil {
		fmt.Fprintln(os.Stderr, "error decoding stdin:", err)
		os.Exit(2)
	}
	d := app.NewDispatcher()
	decision, err := d.Decide(context.Background(), x)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error deciding:", err)
		os.Exit(3)
	}
	if err := json.NewEncoder(os.Stdout).Encode(decision); err != nil {
		fmt.Fprintln(os.Stderr, "error encoding decision:", err)
		os.Exit(4)
	}
}
