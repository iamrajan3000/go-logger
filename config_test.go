package logger

import (
	"reflect"
	"testing"
)

func TestNewConfigSample1(t *testing.T) {
	cfg, err := NewConfig(map[string]string{
		"ts_format":     "dd mm yyyy hh mm ss",
		"log_level":     "INFO",
		"sink_type":     "FILE",
		"file_location": "/var/log/app/info.log",
		"thread_model":  "SINGLE",
		"write_mode":    "SYNC",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		TimeLayout:  "02 01 2006 15 04 05",
		LogLevel:    INFO,
		SinkType:    "FILE",
		ThreadModel: SingleThread,
		WriteMode:   Sync,
		SinkDetails: map[string]string{"file_location": "/var/log/app/info.log"},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("got  %+v\nwant %+v", cfg, want)
	}
}

func TestNewConfigSample2(t *testing.T) {
	cfg, err := NewConfig(map[string]string{
		"ts_format":    "dd:mm:yyyy hh:mm:ss",
		"log_level":    "ERROR",
		"sink_type":    "DB",
		"db host":      "10.0.0.1",
		"db port":      "5432",
		"thread_model": "MULTI",
		"write_mode":   "ASYNC",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		TimeLayout:  "02:01:2006 15:04:05",
		LogLevel:    ERROR,
		SinkType:    "DB",
		ThreadModel: MultiThread,
		WriteMode:   Async,
		SinkDetails: map[string]string{"db host": "10.0.0.1", "db port": "5432"},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("got  %+v\nwant %+v", cfg, want)
	}
}

func TestNewConfigErrors(t *testing.T) {
	base := func() map[string]string {
		return map[string]string{
			"ts_format": "dd mm yyyy hh mm ss",
			"log_level": "INFO",
			"sink_type": "FILE",
		}
	}
	tests := []struct {
		name   string
		modify func(map[string]string)
	}{
		{"missing ts_format", func(m map[string]string) { delete(m, "ts_format") }},
		{"missing log_level", func(m map[string]string) { delete(m, "log_level") }},
		{"missing sink_type", func(m map[string]string) { delete(m, "sink_type") }},
		{"bad log_level", func(m map[string]string) { m["log_level"] = "TRACE" }},
		{"bad ts_format", func(m map[string]string) { m["ts_format"] = "d/m/y" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv := base()
			tt.modify(kv)
			if _, err := NewConfig(kv); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestNewConfigThreadModelAndWriteMode(t *testing.T) {
	kv := func(extra map[string]string) map[string]string {
		m := map[string]string{"ts_format": "dd mm yyyy hh mm ss", "log_level": "INFO", "sink_type": "FILE"}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	cfg, err := NewConfig(kv(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ThreadModel != SingleThread || cfg.WriteMode != Sync {
		t.Errorf("defaults: got %s/%s, want SINGLE/SYNC", cfg.ThreadModel, cfg.WriteMode)
	}

	cfg, err = NewConfig(kv(map[string]string{"thread_model": "MULTI", "write_mode": "ASYNC"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ThreadModel != MultiThread || cfg.WriteMode != Async {
		t.Errorf("got %s/%s, want MULTI/ASYNC", cfg.ThreadModel, cfg.WriteMode)
	}

	for _, bad := range []map[string]string{{"thread_model": "POOL"}, {"write_mode": "LATER"}} {
		if _, err := NewConfig(kv(bad)); err == nil {
			t.Errorf("%v: expected an error", bad)
		}
	}
}
