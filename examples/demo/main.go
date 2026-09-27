package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/iamrajan3000/go-logger"
)

type stdoutSink struct {
	formatter *logger.Formatter
}

func (s *stdoutSink) Write(msg logger.Message) error {
	_, err := fmt.Println(s.formatter.Format(msg))
	return err
}

func main() {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fail(err)
	}

	infoCfg := mustConfig(map[string]string{
		"ts_format":     "dd mm yyyy hh mm ss",
		"log_level":     "INFO",
		"sink_type":     "FILE",
		"file_location": filepath.Join(logDir, "application.log"),
		"max_file_size": "1024",
	})

	warnCfg := mustConfig(map[string]string{
		"ts_format":     "dd mm yyyy hh mm ss",
		"log_level":     "WARN",
		"sink_type":     "FILE",
		"file_location": filepath.Join(logDir, "application.log"),
		"max_file_size": "1024",
	})

	errorCfg := mustConfig(map[string]string{
		"ts_format": "dd:mm:yyyy hh:mm:ss",
		"log_level": "ERROR",
		"sink_type": "STDOUT",
	})

	log, err := logger.New(
		[]logger.Config{infoCfg, warnCfg, errorCfg},
		map[string]logger.SinkFactory{
			"STDOUT": func(cfg logger.Config) (logger.Sink, error) {
				return &stdoutSink{formatter: logger.NewFormatter(cfg.TimeLayout)}, nil
			},
		},
	)
	if err != nil {
		fail(err)
	}

	messages := []struct {
		level     logger.Level
		namespace string
		content   string
	}{
		{logger.INFO, "txn", "Empty txnIds Nothing to fetch"},
		{logger.WARN, "auth", "No user found for the phone number"},
		{logger.DEBUG, "txn", "not logged: no sink tied to DEBUG"},
		{logger.ERROR, "payments", "Payment gateway timed out"},
		{logger.INFO, "txn", "Fetched 3 txnIds"},
		{logger.INFO, "txn", "Fetched 5 txnIds"},
		{logger.WARN, "auth", "OTP retry limit reached"},
	}
	for _, m := range messages {
		if err := log.Log(m.level, m.namespace, m.content); err != nil {
			fail(err)
		}
	}

	if err := log.Close(); err != nil {
		fail(err)
	}

	fmt.Println("\nFiles in", logDir+":")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		fail(err)
	}
	for _, e := range entries {
		fmt.Println(" ", e.Name())
	}
}

func mustConfig(kv map[string]string) logger.Config {
	cfg, err := logger.NewConfig(kv)
	if err != nil {
		fail(err)
	}
	return cfg
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "demo:", err)
	os.Exit(1)
}
