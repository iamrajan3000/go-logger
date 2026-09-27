package logger

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestWrapSink(t *testing.T) {
	base := &memorySink{}
	tests := []struct {
		thread ThreadModel
		mode   WriteMode
		want   string
	}{
		{SingleThread, Sync, "*logger.memorySink"},
		{MultiThread, Sync, "*logger.lockedSink"},
		{SingleThread, Async, "*logger.asyncSink"},
		{MultiThread, Async, "*logger.asyncSink"},
	}
	for _, tt := range tests {
		got := wrapSink(base, Config{ThreadModel: tt.thread, WriteMode: tt.mode})
		if name := fmt.Sprintf("%T", got); name != tt.want {
			t.Errorf("%s/%s: got %s, want %s", tt.thread, tt.mode, name, tt.want)
		}
		if a, ok := got.(*asyncSink); ok {
			if err := a.close(); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestLockedSinkConcurrentWrites(t *testing.T) {
	inner := &memorySink{}
	sink := &lockedSink{sink: inner}

	var wg sync.WaitGroup
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if err := sink.Write(Message{Content: "x"}); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()

	if len(inner.msgs) != 1000 {
		t.Errorf("got %d messages, want 1000", len(inner.msgs))
	}
}

func TestAsyncSinkWritesInOrderAndFlushesOnClose(t *testing.T) {
	inner := &memorySink{}
	sink := newAsyncSink(inner)

	for i := 0; i < 2000; i++ {
		if err := sink.Write(Message{Content: fmt.Sprint(i)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := sink.close(); err != nil {
		t.Fatal(err)
	}

	if len(inner.msgs) != 2000 {
		t.Fatalf("got %d messages, want 2000", len(inner.msgs))
	}
	for i, m := range inner.msgs {
		if m.Content != fmt.Sprint(i) {
			t.Fatalf("message %d out of order: %q", i, m.Content)
		}
	}
}

func TestAsyncSinkReportsErrorsOnClose(t *testing.T) {
	boom := errors.New("disk full")
	sink := newAsyncSink(&memorySink{err: boom})

	if err := sink.Write(Message{}); err != nil {
		t.Fatalf("Write should only queue, got %v", err)
	}
	if err := sink.close(); !errors.Is(err, boom) {
		t.Errorf("close: got %v, want disk full", err)
	}
}

func TestAsyncSinkAfterClose(t *testing.T) {
	sink := newAsyncSink(&memorySink{})
	if err := sink.close(); err != nil {
		t.Fatal(err)
	}

	if err := sink.Write(Message{}); !errors.Is(err, ErrClosed) {
		t.Errorf("got %v, want ErrClosed", err)
	}
	if err := sink.close(); err != nil {
		t.Errorf("second close: got %v, want nil", err)
	}
}
