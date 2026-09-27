package logger

import (
	"fmt"
	"strings"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = [...]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

func (l Level) String() string {
	if l < DEBUG || l > FATAL {
		return fmt.Sprintf("LEVEL(%d)", int(l))
	}
	return levelNames[l]
}

func ParseLevel(s string) (Level, error) {
	name := strings.ToUpper(strings.TrimSpace(s))
	for i, n := range levelNames {
		if n == name {
			return Level(i), nil
		}
	}
	return 0, fmt.Errorf("unknown log level %q", s)
}
