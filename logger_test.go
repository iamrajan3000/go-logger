package logger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type memorySink struct {
	msgs []Message
	err  error
}

func (s *memorySink) Write(msg Message) error {
	s.msgs = append(s.msgs, msg)
	return s.err
}

func memoryFactory(s *memorySink) SinkFactory {
	return func(Config) (Sink, error) { return s, nil }
}

func memConfig(sinkType string, level Level) Config {
	return Config{TimeLayout: "2006-01-02 15:04:05,000", LogLevel: level, SinkType: sinkType}
}

var fixedTime = time.Date(2022, 6, 27, 11, 14, 44, 942_000_000, time.UTC)

func newTestLogger(t *testing.T, configs []Config, custom map[string]SinkFactory) *Logger {
	t.Helper()
	l, err := New(configs, custom)
	if err != nil {
		t.Fatal(err)
	}
	l.now = func() time.Time { return fixedTime }
	return l
}

func mustLog(t *testing.T, l *Logger, level Level, namespace, content string) {
	t.Helper()
	if err := l.Log(level, namespace, content); err != nil {
		t.Fatal(err)
	}
}

func TestLogRoutesByLevelAndEnrichesTimestamp(t *testing.T) {
	info, errSink := &memorySink{}, &memorySink{}
	l := newTestLogger(t,
		[]Config{memConfig("INFO_MEM", INFO), memConfig("ERROR_MEM", ERROR)},
		map[string]SinkFactory{"INFO_MEM": memoryFactory(info), "ERROR_MEM": memoryFactory(errSink)},
	)

	if err := l.Log(INFO, "txn", "Empty txnIds Nothing to fetch"); err != nil {
		t.Fatal(err)
	}
	if err := l.Log(ERROR, "payments", "charge failed"); err != nil {
		t.Fatal(err)
	}

	wantInfo := []Message{{Content: "Empty txnIds Nothing to fetch", Level: INFO, Namespace: "txn", Timestamp: fixedTime}}
	wantErr := []Message{{Content: "charge failed", Level: ERROR, Namespace: "payments", Timestamp: fixedTime}}
	if !reflect.DeepEqual(info.msgs, wantInfo) {
		t.Errorf("INFO sink got %+v", info.msgs)
	}
	if !reflect.DeepEqual(errSink.msgs, wantErr) {
		t.Errorf("ERROR sink got %+v", errSink.msgs)
	}
}

func TestLogLevelWithoutSinkIsNotLogged(t *testing.T) {
	info := &memorySink{}
	l := newTestLogger(t, []Config{memConfig("MEM", INFO)}, map[string]SinkFactory{"MEM": memoryFactory(info)})

	if err := l.Log(DEBUG, "app", "ignored"); err != nil {
		t.Fatal(err)
	}
	if len(info.msgs) != 0 {
		t.Errorf("DEBUG should not reach the INFO sink: %+v", info.msgs)
	}
}

func TestOneSinkForManyLevels(t *testing.T) {
	shared := &memorySink{}
	l := newTestLogger(t,
		[]Config{memConfig("MEM", WARN), memConfig("MEM", ERROR)},
		map[string]SinkFactory{"MEM": memoryFactory(shared)},
	)
	mustLog(t, l, WARN, "app", "w")
	mustLog(t, l, ERROR, "app", "e")
	mustLog(t, l, INFO, "app", "i")

	if len(shared.msgs) != 2 || shared.msgs[0].Content != "w" || shared.msgs[1].Content != "e" {
		t.Errorf("shared sink got %+v", shared.msgs)
	}
}

func TestManySinksForOneLevel(t *testing.T) {
	a, b := &memorySink{}, &memorySink{}
	l := newTestLogger(t,
		[]Config{memConfig("A", ERROR), memConfig("B", ERROR)},
		map[string]SinkFactory{"A": memoryFactory(a), "B": memoryFactory(b)},
	)
	mustLog(t, l, ERROR, "app", "boom")

	if len(a.msgs) != 1 || len(b.msgs) != 1 {
		t.Errorf("both sinks should receive the message: a=%d b=%d", len(a.msgs), len(b.msgs))
	}
}

func TestLogReturnsSinkErrors(t *testing.T) {
	failing := &memorySink{err: errors.New("disk full")}
	ok := &memorySink{}
	l := newTestLogger(t,
		[]Config{memConfig("FAIL", ERROR), memConfig("OK", ERROR)},
		map[string]SinkFactory{"FAIL": memoryFactory(failing), "OK": memoryFactory(ok)},
	)

	err := l.Log(ERROR, "app", "boom")
	if err == nil || !errors.Is(err, failing.err) {
		t.Errorf("got %v, want disk full", err)
	}
	if len(ok.msgs) != 1 {
		t.Error("a failing sink should not stop other sinks")
	}
}

func TestNewErrors(t *testing.T) {
	if _, err := New([]Config{memConfig("KAFKA", INFO)}, nil); err == nil {
		t.Error("expected an error for an unknown sink_type")
	}

	failing := func(Config) (Sink, error) { return nil, errors.New("bad config") }
	if _, err := New([]Config{memConfig("X", INFO)}, map[string]SinkFactory{"X": failing}); err == nil {
		t.Error("expected the factory error")
	}

	if _, err := New([]Config{memConfig(SinkTypeFile, INFO)}, nil); err == nil {
		t.Error("expected an error for a FILE sink without file_location")
	}
}

func TestLoggerWithFileSinkFromSampleConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "info.log")
	cfg, err := NewConfig(map[string]string{
		"ts_format":     "dd mm yyyy hh mm ss",
		"log_level":     "INFO",
		"sink_type":     "FILE",
		"file_location": path,
	})
	if err != nil {
		t.Fatal(err)
	}
	l := newTestLogger(t, []Config{cfg}, nil)

	if err := l.Log(INFO, "txn", "Empty txnIds Nothing to fetch"); err != nil {
		t.Fatal(err)
	}
	if err := l.Log(WARN, "auth", "No user found for the phone number"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "INFO [27 06 2022 11 14 44] Empty txnIds Nothing to fetch\n"
	if string(got) != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestMultiThreadFileSinkWithRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.log")
	cfg := func(level Level) Config {
		c := fileConfig(path)
		c.LogLevel = level
		c.ThreadModel = MultiThread
		c.SinkDetails[KeyMaxFileSize] = "500"
		return c
	}

	l := newTestLogger(t, []Config{cfg(INFO), cfg(WARN)}, nil)

	const goroutines, perGoroutine = 20, 25
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			level := INFO
			if g%2 == 1 {
				level = WARN
			}
			for i := 0; i < perGoroutine; i++ {
				if err := l.Log(level, "app", fmt.Sprintf("g%d-%d", g, i)); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()

	assertAllLinesLogged(t, path, goroutines*perGoroutine)
}

func TestAsyncFileSinkFlushesOnClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.log")
	c := fileConfig(path)
	c.ThreadModel = MultiThread
	c.WriteMode = Async
	c.SinkDetails[KeyMaxFileSize] = "500"
	l := newTestLogger(t, []Config{c}, nil)

	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				if err := l.Log(INFO, "app", fmt.Sprintf("g%d-%d", g, i)); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	assertAllLinesLogged(t, path, 300)

	if err := l.Log(INFO, "app", "late"); !errors.Is(err, ErrClosed) {
		t.Errorf("Log after Close: got %v, want ErrClosed", err)
	}
}

func TestSharedSinkMustAgreeOnModes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.log")
	info, warn := fileConfig(path), fileConfig(path)
	warn.LogLevel = WARN
	warn.WriteMode = Async
	if _, err := New([]Config{info, warn}, nil); err == nil {
		t.Error("expected an error for one file with different write modes")
	}

	warn.WriteMode = Sync
	warn.TimeLayout = "02:01:2006"
	if _, err := New([]Config{info, warn}, nil); err == nil {
		t.Error("expected an error for one file with different ts_format")
	}
}

func assertAllLinesLogged(t *testing.T, path string, want int) {
	t.Helper()
	all := readFile(t, path)
	archives, _ := filepath.Glob(path + ".*.gz")
	for _, a := range archives {
		all += readGzip(t, a)
	}
	if len(archives) == 0 {
		t.Error("expected at least one rotation")
	}

	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(all), "\n") {
		if seen[line] {
			t.Errorf("duplicate line %q", line)
		}
		seen[line] = true
	}
	if len(seen) != want {
		t.Errorf("got %d distinct lines, want %d", len(seen), want)
	}
}
