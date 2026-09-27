package logger

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
)

const (
	SinkTypeFile    = "FILE"
	KeyFileLocation = "file_location"
	KeyMaxFileSize  = "max_file_size"
	filePermissions = 0o644
	fileOpenFlags   = os.O_CREATE | os.O_APPEND | os.O_WRONLY
)

type FileSink struct {
	path        string
	maxFileSize int64
	formatter   *Formatter
}

func NewFileSink(cfg Config) (*FileSink, error) {
	path := cfg.SinkDetails[KeyFileLocation]
	if path == "" {
		return nil, fmt.Errorf("file sink: missing %s", KeyFileLocation)
	}

	var maxFileSize int64
	if v, ok := cfg.SinkDetails[KeyMaxFileSize]; ok {
		size, err := strconv.ParseInt(v, 10, 64)
		if err != nil || size <= 0 {
			return nil, fmt.Errorf("file sink: %s must be a positive number of bytes, got %q", KeyMaxFileSize, v)
		}
		maxFileSize = size
	}

	return &FileSink{
		path:        path,
		maxFileSize: maxFileSize,
		formatter:   NewFormatter(cfg.TimeLayout),
	}, nil
}

func (s *FileSink) Write(msg Message) error {
	line := s.formatter.Format(msg) + "\n"

	if err := s.rotateIfNeeded(int64(len(line))); err != nil {
		return fmt.Errorf("file sink: rotate: %w", err)
	}

	f, err := os.OpenFile(s.path, fileOpenFlags, filePermissions)
	if err != nil {
		return fmt.Errorf("file sink: %w", err)
	}
	_, writeErr := f.WriteString(line)
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("file sink: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("file sink: %w", closeErr)
	}
	return nil
}

func (s *FileSink) rotateIfNeeded(n int64) error {
	if s.maxFileSize == 0 {
		return nil
	}
	info, err := os.Stat(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() == 0 || info.Size()+n <= s.maxFileSize {
		return nil
	}
	return rotate(s.path)
}
