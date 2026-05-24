package agentmeshkafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
)

// newFA is a test helper that constructs a FileAdapter and registers Close on cleanup.
func newFA(t *testing.T, path string) *agentmeshkafka.FileAdapter {
	t.Helper()
	a, err := agentmeshkafka.NewFileAdapter(path)
	if err != nil {
		t.Fatalf("NewFileAdapter(%q): %v", path, err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

// TestFileAdapter_ReadsGoldenJSONL verifies that all 3 lines of golden-3.jsonl
// are read in order and the IDs match the expected sequence.
func TestFileAdapter_ReadsGoldenJSONL(t *testing.T) {
	t.Parallel()
	a := newFA(t, "testdata/golden-3.jsonl")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	ex, errs := a.Stream(ctx, nil)

	got := make([]string, 0, 3)
	for x := range ex {
		got = append(got, x.ID)
	}
	for e := range errs {
		t.Logf("non-fatal error: %v", e)
	}

	want := []string{"g-001", "g-002", "g-003"}
	if len(got) != len(want) {
		t.Fatalf("got %d exchanges, want %d; IDs: %v", len(got), len(want), got)
	}
	for i, id := range want {
		if got[i] != id {
			t.Errorf("exchange[%d]: got ID %q, want %q", i, got[i], id)
		}
	}
}

// TestFileAdapter_ContextCancellation_StopsStream cancels the context immediately
// after calling Stream and verifies both channels drain without blocking (no goroutine leak).
func TestFileAdapter_ContextCancellation_StopsStream(t *testing.T) {
	t.Parallel()
	a := newFA(t, "testdata/golden-3.jsonl")

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel before first read

	ex, errs := a.Stream(ctx, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range ex {
		}
		for range errs {
		}
	}()

	select {
	case <-done:
		// channels drained — no goroutine leak
	case <-time.After(2 * time.Second):
		t.Fatal("channels did not drain after context cancellation (goroutine leak?)")
	}
}

// TestFileAdapter_CloseIsIdempotent calls Close twice and expects no panic and no error.
func TestFileAdapter_CloseIsIdempotent(t *testing.T) {
	t.Parallel()
	a := newFA(t, "testdata/golden-3.jsonl")

	if err := a.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("second Close (must be idempotent): %v", err)
	}
}

// TestFileAdapter_MalformedLine_ReportsError_ContinuesStream uses a 3-line fixture
// with 1 invalid line in the middle. Expects exactly 1 error and 2 well-formed exchanges.
func TestFileAdapter_MalformedLine_ReportsError_ContinuesStream(t *testing.T) {
	t.Parallel()
	a := newFA(t, "testdata/golden-3-malformed.jsonl")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	ex, errs := a.Stream(ctx, nil)

	var nEx, nErr int
	exDone, errDone := false, false
	for !exDone || !errDone {
		select {
		case _, ok := <-ex:
			if !ok {
				exDone = true
				continue
			}
			nEx++
		case _, ok := <-errs:
			if !ok {
				errDone = true
				continue
			}
			nErr++
		}
	}

	if nErr == 0 {
		t.Fatal("expected at least 1 parse error for the malformed line")
	}
	if nEx != 2 {
		t.Fatalf("got %d well-formed exchanges, want 2", nEx)
	}
}

// TestFileAdapter_MissingFile_ConstructorError verifies that NewFileAdapter returns
// a non-nil error when the file does not exist.
func TestFileAdapter_MissingFile_ConstructorError(t *testing.T) {
	t.Parallel()
	_, err := agentmeshkafka.NewFileAdapter("testdata/does-not-exist.jsonl")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
