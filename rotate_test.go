package logger

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readGzip(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFileSinkRotatesAndCompresses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.log")
	cfg := fileConfig(path)

	cfg.SinkDetails[KeyMaxFileSize] = "40"
	sink, err := NewFileSink(cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, content := range []string{"msg-1", "msg-2", "msg-3"} {
		if err := sink.Write(Message{Level: INFO, Content: content, Timestamp: time.Unix(0, 0).UTC()}); err != nil {
			t.Fatal(err)
		}
	}

	checks := map[string]string{
		path:           "msg-3",
		path + ".1.gz": "msg-2",
		path + ".2.gz": "msg-1",
	}
	for file, want := range checks {
		var got string
		if strings.HasSuffix(file, ".gz") {
			got = readGzip(t, file)
		} else {
			got = readFile(t, file)
		}
		if !strings.Contains(got, want) || strings.Count(got, "\n") != 1 {
			t.Errorf("%s = %q, want a single line containing %q", filepath.Base(file), got, want)
		}
	}
	if _, err := os.Stat(path + ".3.gz"); err == nil {
		t.Error("unexpected application.log.3.gz")
	}
}

func TestFileSinkNoRotationWithinLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.log")
	cfg := fileConfig(path)
	cfg.SinkDetails[KeyMaxFileSize] = "1000"
	sink, err := NewFileSink(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := sink.Write(Message{Level: INFO, Content: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(path + ".1.gz"); err == nil {
		t.Error("file should not rotate below max_file_size")
	}
	if got := strings.Count(readFile(t, path), "\n"); got != 3 {
		t.Errorf("got %d lines, want 3", got)
	}
}

func TestNewFileSinkBadMaxFileSize(t *testing.T) {
	for _, v := range []string{"abc", "0", "-5"} {
		cfg := fileConfig("/tmp/x.log")
		cfg.SinkDetails[KeyMaxFileSize] = v
		if _, err := NewFileSink(cfg); err == nil {
			t.Errorf("max_file_size=%q: expected an error", v)
		}
	}
}
