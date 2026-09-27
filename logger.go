package logger

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"time"
)

type SinkFactory func(cfg Config) (Sink, error)

type Logger struct {
	sinks      map[Level][]Sink
	asyncSinks []*asyncSink
	now        func() time.Time
}

func New(configs []Config, customSinks map[string]SinkFactory) (*Logger, error) {
	l := &Logger{
		sinks: map[Level][]Sink{},
		now:   time.Now,
	}
	builtIn := map[string]SinkFactory{SinkTypeFile: sharedFileSinks()}
	wrapped := map[Sink]wrappedSink{}

	for _, cfg := range configs {
		factory, ok := customSinks[cfg.SinkType]
		if !ok {
			factory, ok = builtIn[cfg.SinkType]
		}
		if !ok {
			return nil, fmt.Errorf("logger: unknown sink_type %q", cfg.SinkType)
		}
		sink, err := factory(cfg)
		if err != nil {
			return nil, fmt.Errorf("logger: %s sink: %w", cfg.SinkType, err)
		}

		if reflect.TypeOf(sink).Comparable() {
			if w, seen := wrapped[sink]; seen {
				if w.threadModel != cfg.ThreadModel || w.writeMode != cfg.WriteMode {
					return nil, fmt.Errorf("logger: %s sink shared by levels with different thread_model/write_mode", cfg.SinkType)
				}
				l.sinks[cfg.LogLevel] = append(l.sinks[cfg.LogLevel], w.sink)
				continue
			}
		}
		w := wrappedSink{sink: wrapSink(sink, cfg), threadModel: cfg.ThreadModel, writeMode: cfg.WriteMode}
		if reflect.TypeOf(sink).Comparable() {
			wrapped[sink] = w
		}
		if a, ok := w.sink.(*asyncSink); ok {
			l.asyncSinks = append(l.asyncSinks, a)
		}
		l.sinks[cfg.LogLevel] = append(l.sinks[cfg.LogLevel], w.sink)
	}
	return l, nil
}

type wrappedSink struct {
	sink        Sink
	threadModel ThreadModel
	writeMode   WriteMode
}

func sharedFileSinks() SinkFactory {
	type entry struct {
		sink *FileSink
		cfg  Config
	}
	byPath := map[string]entry{}
	return func(cfg Config) (Sink, error) {
		path := filepath.Clean(cfg.SinkDetails[KeyFileLocation])
		if e, ok := byPath[path]; ok {
			if e.cfg.TimeLayout != cfg.TimeLayout || !maps.Equal(e.cfg.SinkDetails, cfg.SinkDetails) {
				return nil, fmt.Errorf("%s is configured with different settings for different levels", path)
			}
			return e.sink, nil
		}
		sink, err := NewFileSink(cfg)
		if err != nil {
			return nil, err
		}
		byPath[path] = entry{sink: sink, cfg: cfg}
		return sink, nil
	}
}

func (l *Logger) Log(level Level, namespace, content string) error {
	msg := Message{
		Content:   content,
		Level:     level,
		Namespace: namespace,
		Timestamp: l.now(),
	}
	var errs []error
	for _, sink := range l.sinks[level] {
		if err := sink.Write(msg); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (l *Logger) Close() error {
	var errs []error
	for _, a := range l.asyncSinks {
		if err := a.close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
