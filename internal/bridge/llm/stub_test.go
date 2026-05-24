package llm_test

import (
	"context"
	"testing"

	"github.com/agbruneau/taugo/internal/bridge/llm"
)

func TestStub_Fingerprint(t *testing.T) {
	t.Parallel()
	var c llm.Client = llm.Stub{}
	if c.Fingerprint() != "stub:v0" {
		t.Fatalf("fingerprint = %q, want \"stub:v0\"", c.Fingerprint())
	}
}

func TestStub_Interpret_DeterministicAndBounded(t *testing.T) {
	t.Parallel()
	var c llm.Client = llm.Stub{}
	ctx := context.Background()
	cases := []string{"", "a", "hello world", "the quick brown fox"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			a, err := c.Interpret(ctx, in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b, _ := c.Interpret(ctx, in)
			if a != b {
				t.Fatalf("non-deterministic: got %f then %f for %q", a, b, in)
			}
			if a < 0 || a >= 1 {
				t.Fatalf("score %f out of [0, 1) for %q", a, in)
			}
		})
	}
}
