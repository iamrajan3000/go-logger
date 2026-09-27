package logger

import (
	"fmt"
	"strings"
)

const (
	KeyTsFormat    = "ts_format"
	KeyLogLevel    = "log_level"
	KeySinkType    = "sink_type"
	KeyThreadModel = "thread_model"
	KeyWriteMode   = "write_mode"
)

type ThreadModel string

const (
	SingleThread ThreadModel = "SINGLE"
	MultiThread  ThreadModel = "MULTI"
)

type WriteMode string

const (
	Sync  WriteMode = "SYNC"
	Async WriteMode = "ASYNC"
)

type Config struct {
	TimeLayout  string
	LogLevel    Level
	SinkType    string
	ThreadModel ThreadModel
	WriteMode   WriteMode
	SinkDetails map[string]string
}

func NewConfig(kv map[string]string) (Config, error) {
	cfg := Config{
		ThreadModel: SingleThread,
		WriteMode:   Sync,
		SinkDetails: map[string]string{},
	}

	for _, key := range []string{KeyTsFormat, KeyLogLevel, KeySinkType} {
		if strings.TrimSpace(kv[key]) == "" {
			return Config{}, fmt.Errorf("config: missing %s", key)
		}
	}

	for key, value := range kv {
		value = strings.TrimSpace(value)
		var err error
		switch key {
		case KeyTsFormat:
			cfg.TimeLayout, err = toGoLayout(value)
		case KeyLogLevel:
			cfg.LogLevel, err = ParseLevel(value)
		case KeySinkType:
			cfg.SinkType = strings.ToUpper(value)
		case KeyThreadModel:
			cfg.ThreadModel, err = parseThreadModel(value)
		case KeyWriteMode:
			cfg.WriteMode, err = parseWriteMode(value)
		default:
			cfg.SinkDetails[key] = value
		}
		if err != nil {
			return Config{}, fmt.Errorf("config: %s: %w", key, err)
		}
	}
	return cfg, nil
}

func parseThreadModel(s string) (ThreadModel, error) {
	switch m := ThreadModel(strings.ToUpper(s)); m {
	case SingleThread, MultiThread:
		return m, nil
	}
	return "", fmt.Errorf("unknown thread model %q", s)
}

func parseWriteMode(s string) (WriteMode, error) {
	switch m := WriteMode(strings.ToUpper(s)); m {
	case Sync, Async:
		return m, nil
	}
	return "", fmt.Errorf("unknown write mode %q", s)
}

func toGoLayout(format string) (string, error) {
	var b strings.Builder
	afterHour := false
	for i := 0; i < len(format); {
		switch {
		case strings.HasPrefix(format[i:], "yyyy"):
			b.WriteString("2006")
			i += 4
		case strings.HasPrefix(format[i:], "dd"):
			b.WriteString("02")
			i += 2
		case strings.HasPrefix(format[i:], "hh"):
			b.WriteString("15")
			afterHour = true
			i += 2
		case strings.HasPrefix(format[i:], "mm"):
			if afterHour {
				b.WriteString("04")
			} else {
				b.WriteString("01")
			}
			i += 2
		case strings.HasPrefix(format[i:], "ss"):
			b.WriteString("05")
			i += 2
		case strings.ContainsRune("dmyhs", rune(format[i])):
			return "", fmt.Errorf("unsupported time format %q", format)
		default:
			b.WriteByte(format[i])
			i++
		}
	}
	return b.String(), nil
}
