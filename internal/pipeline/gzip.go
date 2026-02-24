package pipeline

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
)

// openMaybeGzipped opens a file and wraps it in a gzip reader when the
// path ends with ".gz". The returned cleanup function closes all
// underlying readers and must always be called.
func openMaybeGzipped(path string) (io.Reader, func() error, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open manpage: %w", err)
	}

	if !strings.HasSuffix(strings.ToLower(path), ".gz") {
		cleanup := func() error { return file.Close() }
		return file, cleanup, nil
	}

	gz, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("read gzip: %w", err)
	}
	cleanup := func() error {
		_ = gz.Close()
		return file.Close()
	}
	return gz, cleanup, nil
}
