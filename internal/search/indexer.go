package search

import "context"

// Indexer abstracts search indexing so the pipeline package does not depend
// on a specific search implementation.
type Indexer interface {
	IndexManpage(ctx context.Context, doc Document) error
	Close() error
}

// Document represents a manpage document to be indexed for search.
type Document struct {
	Title    string
	Path     string
	Section  int
	Distro   string
	Language string
	Content  string
}
