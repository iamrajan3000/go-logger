package logger

import (
	"errors"
	"sync"
)

var ErrClosed = errors.New("logger: closed")

const asyncBufferSize = 1024

func wrapSink(sink Sink, cfg Config) Sink {
	if cfg.WriteMode == Async {

		return newAsyncSink(sink)
	}
	if cfg.ThreadModel == MultiThread {
		return &lockedSink{sink: sink}
	}
	return sink
}

type lockedSink struct {
	mu   sync.Mutex
	sink Sink
}

func (s *lockedSink) Write(msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sink.Write(msg)
}

type asyncSink struct {
	sink  Sink
	queue chan Message
	done  chan struct{}
	errs  []error

	mu     sync.RWMutex
	closed bool
}

func newAsyncSink(sink Sink) *asyncSink {
	s := &asyncSink{
		sink:  sink,
		queue: make(chan Message, asyncBufferSize),
		done:  make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *asyncSink) run() {
	defer close(s.done)
	for msg := range s.queue {
		if err := s.sink.Write(msg); err != nil {
			s.errs = append(s.errs, err)
		}
	}
}

func (s *asyncSink) Write(msg Message) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ErrClosed
	}
	s.queue <- msg
	return nil
}

func (s *asyncSink) close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.queue)
	s.mu.Unlock()

	<-s.done
	return errors.Join(s.errs...)
}
