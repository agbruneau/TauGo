// Command tau is the TauGo CLI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
			os.Exit(runDecide(os.Stdin, os.Stdout))
		case "calibrate":
			os.Exit(runCalibrate(os.Args[2:]))
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

// runDecide reads a JSON Exchange from in, decides a regime, and writes the
// JSON Decision to out. Returns an exit code: 0 success, 2 bad input,
// 3 decide error, 4 encode error.
func runDecide(in io.Reader, out io.Writer) int {
	var x tau.Exchange
	if err := json.NewDecoder(in).Decode(&x); err != nil {
		fmt.Fprintln(os.Stderr, "error decoding stdin:", err)
		return 2
	}
	d := app.NewDispatcher()
	decision, err := d.Decide(context.Background(), x)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error deciding:", err)
		return 3
	}
	if err := json.NewEncoder(out).Encode(decision); err != nil {
		fmt.Fprintln(os.Stderr, "error encoding decision:", err)
		return 4
	}
	return 0
}
