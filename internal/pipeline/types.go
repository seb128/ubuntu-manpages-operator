package pipeline

import (
	"github.com/canonical/ubuntu-manpages-operator/internal/transform"
)

type ManpageFile struct {
	Path          string
	RelativePath  string
	IsSymlink     bool
	SymlinkTarget string
	Meta          transform.ManpageMeta
}

// ReleaseStatus represents the current progress of a single release ingest.
type ReleaseStatus struct {
	Release      string
	Stage        string // "waiting", "fetching", "processing", "done", "error"
	Total        int
	Done         int
	Skipped      int
	Errors       int
	FailuresPath string
}

// ConvertError wraps a mandoc conversion failure so callers can
// distinguish it from other pipeline errors (e.g. to treat it as non-fatal).
type ConvertError struct{ Err error }

func (e *ConvertError) Error() string { return e.Err.Error() }
func (e *ConvertError) Unwrap() error { return e.Err }
