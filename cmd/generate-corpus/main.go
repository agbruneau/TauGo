package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/agbruneau/taugo/internal/app"
)

func main() {
	var (
		count    = flag.Int("count", 120, "number of exchanges to generate (>= 1)")
		seed     = flag.Int64("seed", 42, "deterministic seed; same seed+count+distribution → identical output")
		output   = flag.String("output", "testdata/synthetic-corpus.jsonl", "output path (.jsonl); use - for stdout")
		distr    = flag.String("distribution", "balanced", "profile: balanced | i4-heavy | refus-heavy")
		annotate = flag.Bool("annotate-with-dispatcher", false, "enrich each line with expected_regime via the production Dispatcher")
	)
	flag.Parse()

	if *count < 1 {
		fmt.Fprintln(os.Stderr, "generate-corpus: --count must be >= 1")
		os.Exit(2)
	}

	profile := DistributionProfile(*distr)
	if _, err := weightsFor(profile); err != nil {
		fmt.Fprintf(os.Stderr, "generate-corpus: %v\n", err)
		os.Exit(2)
	}

	if err := run(*seed, *count, profile, *output, *annotate); err != nil {
		fmt.Fprintf(os.Stderr, "generate-corpus: %v\n", err)
		os.Exit(1)
	}
}

func run(seed int64, count int, profile DistributionProfile, output string, annotate bool) error {
	w := os.Stdout
	if output != "-" {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("create %s: %w", output, err)
		}
		defer func() { _ = f.Close() }()
		w = f
	}
	g := NewGenerator(seed)
	if annotate {
		d := app.NewDispatcher()
		return g.GenerateAnnotated(context.Background(), w, count, profile, d)
	}
	return g.Generate(w, count, profile)
}
