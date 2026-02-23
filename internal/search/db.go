package search

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// schema drops and recreates all tables. The index is rebuilt from scratch
// on each ingest so there is no need for migrations.
const schema = `
DROP TRIGGER IF EXISTS manpages_au;
DROP TRIGGER IF EXISTS manpages_ad;
DROP TRIGGER IF EXISTS manpages_ai;
DROP TABLE IF EXISTS manpages_fts;
DROP TABLE IF EXISTS manpages;

CREATE TABLE manpages (
	path TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	section INTEGER NOT NULL,
	distro TEXT NOT NULL,
	language TEXT NOT NULL DEFAULT '',
	content TEXT NOT NULL
);

CREATE VIRTUAL TABLE manpages_fts USING fts5(
	title, content,
	content='manpages',
	content_rowid='rowid'
);

CREATE TRIGGER manpages_ai AFTER INSERT ON manpages BEGIN
	INSERT INTO manpages_fts(rowid, title, content)
	VALUES (new.rowid, new.title, new.content);
END;

CREATE TRIGGER manpages_ad AFTER DELETE ON manpages BEGIN
	INSERT INTO manpages_fts(manpages_fts, rowid, title, content)
	VALUES ('delete', old.rowid, old.title, old.content);
END;

CREATE TRIGGER manpages_au AFTER UPDATE ON manpages BEGIN
	INSERT INTO manpages_fts(manpages_fts, rowid, title, content)
	VALUES ('delete', old.rowid, old.title, old.content);
	INSERT INTO manpages_fts(rowid, title, content)
	VALUES (new.rowid, new.title, new.content);
END;
`

func openDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open search db: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	return db, nil
}
