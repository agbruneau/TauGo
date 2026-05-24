package agentmeshkafka

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sync"
)

// FileAdapter reads JSONL from a local file and publishes each parsed
// AgentMeshExchange onto the exchanges channel. Parse errors are emitted on
// the error channel without stopping the stream (resilient). Closes cleanly
// on EOF, ctx.Done(), or Close(). Close() is idempotent via sync.Once.
type FileAdapter struct {
	path string
	mu   sync.Mutex
	once sync.Once
	stop context.CancelFunc
}

// NewFileAdapter constructs a FileAdapter bound to path. The file must exist
// at construction time; it is opened lazily in Stream.
func NewFileAdapter(path string) (*FileAdapter, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("agentmeshkafka: stat %s: %w", path, err)
	}
	return &FileAdapter{path: path}, nil
}

// Stream opens the JSONL file and emits parsed exchanges. If topics is
// non-empty, only lines whose SourceTopic is in the list are published.
// Both returned channels are closed when the stream ends.
func (f *FileAdapter) Stream(ctx context.Context, topics []string) (exchanges <-chan AgentMeshExchange, errc <-chan error) {
	ex := make(chan AgentMeshExchange)
	errs := make(chan error, 8)

	subCtx, cancel := context.WithCancel(ctx)
	f.mu.Lock()
	f.stop = cancel
	f.mu.Unlock()

	go func() {
		defer cancel()
		defer close(ex)
		defer close(errs)
		f.scan(subCtx, ex, errs, topics)
	}()
	return ex, errs
}

// scan is the inner read loop, extracted to keep Stream's cognitive complexity low.
func (f *FileAdapter) scan(ctx context.Context, ex chan<- AgentMeshExchange, errs chan<- error, topics []string) {
	file, err := os.Open(f.path)
	if err != nil {
		sendErr(errs, fmt.Errorf("agentmeshkafka: open %s: %w", f.path, err))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var x AgentMeshExchange
		if err := json.Unmarshal(line, &x); err != nil {
			sendErr(errs, fmt.Errorf("agentmeshkafka: parse: %w", err))
			continue
		}
		if len(topics) > 0 && !slices.Contains(topics, x.SourceTopic) {
			continue
		}
		select {
		case ex <- x:
		case <-ctx.Done():
			return
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		sendErr(errs, fmt.Errorf("agentmeshkafka: scan: %w", scanErr))
	}
}

// sendErr is a non-blocking helper that drops the error if the buffer is full.
func sendErr(errs chan<- error, err error) {
	select {
	case errs <- err:
	default:
	}
}

// Close cancels any active Stream and releases resources. Safe to call
// multiple times; subsequent calls are no-ops.
func (f *FileAdapter) Close() error {
	f.once.Do(func() {
		f.mu.Lock()
		fn := f.stop
		f.mu.Unlock()
		if fn != nil {
			fn()
		}
	})
	return nil
}
