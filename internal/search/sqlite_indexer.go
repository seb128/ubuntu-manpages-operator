package search

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

const batchSize = 500

type SQLiteIndexer struct {
	mu         sync.Mutex
	db         *sql.DB
	insertStmt *sql.Stmt
	tx         *sql.Tx
	txStmt     *sql.Stmt
	count      int
}

func NewSQLiteIndexer(path string) (*SQLiteIndexer, error) {
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	stmt, err := db.Prepare(`INSERT OR REPLACE INTO manpages (path, title, section, distro, language, content) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("prepare insert: %w", err)
	}

	return &SQLiteIndexer{
		db:         db,
		insertStmt: stmt,
	}, nil
}

func (s *SQLiteIndexer) IndexManpage(ctx context.Context, doc Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tx == nil {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		s.tx = tx
		s.txStmt = tx.Stmt(s.insertStmt)
	}

	_, err := s.txStmt.ExecContext(ctx, doc.Path, doc.Title, doc.Section, doc.Distro, doc.Language, doc.Content)
	if err != nil {
		return fmt.Errorf("index manpage %s: %w", doc.Path, err)
	}

	s.count++
	if s.count >= batchSize {
		if err := s.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteIndexer) flush() error {
	if s.tx == nil {
		return nil
	}
	err := s.tx.Commit()
	s.tx = nil
	s.txStmt = nil
	s.count = 0
	if err != nil {
		return fmt.Errorf("commit batch: %w", err)
	}
	return nil
}

func (s *SQLiteIndexer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.flush(); err != nil {
		return err
	}
	_ = s.insertStmt.Close()
	return s.db.Close()
}
