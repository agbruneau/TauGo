// Command tau is the TauGo CLI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/tau"
)

var (
	buildTimestamp = "dev" //nolint:gochecknoglobals // build-time variable set via -ldflags; cannot be const
	version        = "0.1.1-pre"
)

func main() {
	os.Exit(runMain(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// runMain is the testable entry point. args must exclude the program name.
// Returns an exit code: 0 success, 1 generic error, 2 usage/bad args.
func runMain(args []string, in io.Reader, out, stderr io.Writer) int {
	if len(args) >= 1 {
		switch args[0] {
		case "--version", "-version":
			fmt.Fprintf(out, "tau %s (build %s)\n", version, buildTimestamp)
			return 0
		case "decide":
			return runDecide(in, out)
		case "calibrate":
			return runCalibrate(args[1:])
		default:
			fmt.Fprintf(stderr, "tau: unknown command %q\n\n", args[0])
			printUsage(stderr)
			return 2
		}
	}
	printUsage(stderr)
	return 2
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `tau — TauGo kernel CLI (V0.1)

USAGE:
    tau <command> [flags]

COMMANDS:
    decide      Decide a regime for one exchange (reads JSON Exchange on stdin)
    calibrate   Run adaptive calibration on a corpus (M5+)
    --version   Print version

Specification: PRD.md
`)
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
