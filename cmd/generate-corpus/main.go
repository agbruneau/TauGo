package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/agbruneau/taugo/internal/app"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

// run parses args, generates the corpus, and returns an exit code.
// stdout is used when --output is "-" (or omitted and output resolves to "-").
// Exit codes: 0 = success, 1 = I/O or generation error, 2 = bad arguments.
func run(args []string, stdout io.Writer) int {
	fs := flag.NewFlagSet("generate-corpus", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	count := fs.Int("count", 120, "number of exchanges to generate (>= 1)")
	seed := fs.Int64("seed", 42, "deterministic seed; same seed+count+distribution → identical output")
	output := fs.String("output", "-", "output path (.jsonl); use - for stdout")
	distr := fs.String("distribution", "balanced", "profile: balanced | i4-heavy | refus-heavy")
	annotate := fs.Bool("annotate-with-dispatcher", false, "enrich each line with expected_regime via the production Dispatcher")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "generate-corpus: %v\n", err)
		return 2
	}

	if *count < 1 {
		fmt.Fprintln(os.Stderr, "generate-corpus: --count must be >= 1")
		return 2
	}

	profile := DistributionProfile(*distr)
	if _, err := weightsFor(profile); err != nil {
		fmt.Fprintf(os.Stderr, "generate-corpus: %v\n", err)
		return 2
	}

	w := stdout
	if *output != "-" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate-corpus: create %s: %v\n", *output, err)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	g := NewGenerator(*seed)
	if *annotate {
		d := app.NewDispatcher()
		if err := g.GenerateAnnotated(context.Background(), w, *count, profile, d); err != nil {
			fmt.Fprintf(os.Stderr, "generate-corpus: %v\n", err)
			return 1
		}
		return 0
	}
	if err := g.Generate(w, *count, profile); err != nil {
		fmt.Fprintf(os.Stderr, "generate-corpus: %v\n", err)
		return 1
	}
	return 0
}
