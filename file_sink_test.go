package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fileConfig(path string) Config {
	return Config{
		TimeLayout:  "2006-01-02 15:04:05,000",
		LogLevel:    INFO,
		SinkType:    SinkTypeFile,
		SinkDetails: map[string]string{KeyFileLocation: path},
	}
}

func TestFileSinkWritesSampleExecution(t *testing.T) {
	path := filepath.Join(t.TempDir(), "info.log")
	sink, err := NewFileSink(fileConfig(path))
	if err != nil {
		t.Fatal(err)
	}

	msgs := []Message{
		{
			Level: INFO, Content: "Empty txnIds Nothing to fetch",
			Timestamp: time.Date(2022, 6, 27, 11, 14, 44, 942_000_000, time.UTC),
		},
		{
			Level: WARN, Content: "No user found for the phone number",
			Timestamp: time.Date(2022, 6, 27, 11, 28, 6, 229_000_000, time.UTC),
		},
	}
	for _, m := range msgs {
		if err := sink.Write(m); err != nil {
			t.Fatal(err)
		}
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "INFO [2022-06-27 11:14:44,942] Empty txnIds Nothing to fetch\n" +
		"WARN [2022-06-27 11:28:06,229] No user found for the phone number\n"
	if string(got) != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestFileSinkAppendsToExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "info.log")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sink, err := NewFileSink(fileConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.Write(Message{Level: INFO, Content: "new", Timestamp: time.Unix(0, 0).UTC()}); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	want := "existing\nINFO [1970-01-01 00:00:00,000] new\n"
	if string(got) != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestNewFileSinkMissingLocation(t *testing.T) {
	cfg := fileConfig("")
	if _, err := NewFileSink(cfg); err == nil {
		t.Error("expected an error for missing file_location")
	}
}

func TestFileSinkWriteFailsForBadPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", "info.log")
	sink, err := NewFileSink(fileConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.Write(Message{Level: INFO, Content: "x"}); err == nil {
		t.Error("expected an error writing to a missing directory")
	}
}
