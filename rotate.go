package logger

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

func rotate(path string) error {
	if err := shiftArchives(path); err != nil {
		return err
	}
	if err := compressFile(path, archiveName(path, 1)); err != nil {
		return err
	}
	return os.Remove(path)
}

func archiveName(path string, n int) string {
	return fmt.Sprintf("%s.%d.gz", path, n)
}

func shiftArchives(path string) error {
	highest := 0
	for {
		_, err := os.Stat(archiveName(path, highest+1))
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return err
		}
		highest++
	}
	for n := highest; n >= 1; n-- {
		if err := os.Rename(archiveName(path, n), archiveName(path, n+1)); err != nil {
			return err
		}
	}
	return nil
}

func compressFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(out)
	_, copyErr := io.Copy(gz, in)
	gzErr := gz.Close()
	outErr := out.Close()
	if err := errors.Join(copyErr, gzErr, outErr); err != nil {
		os.Remove(dst)
		return err
	}
	return nil
}
