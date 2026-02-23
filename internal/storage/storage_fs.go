package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type FSStorage struct {
	Root string
}

func NewFSStorage(root string) *FSStorage {
	return &FSStorage{Root: root}
}

func (s *FSStorage) WriteHTML(ctx context.Context, destPath string, content []byte) error {
	return s.writeFile(destPath, content)
}

func (s *FSStorage) WriteSymlink(ctx context.Context, destPath string, target string) error {
	return s.writeSymlink(destPath, target)
}

func (s *FSStorage) WriteGzip(ctx context.Context, destPath string, content []byte) error {
	return s.writeFile(destPath, content)
}

func (s *FSStorage) WriteGzipSymlink(ctx context.Context, destPath string, target string) error {
	return s.writeSymlink(destPath, target)
}

func (s *FSStorage) CheckCache(release string, pkgName string, sha1 string) bool {
	cachePath := filepath.Join(s.Root, "manpages", release, ".cache", pkgName)
	data, err := os.ReadFile(cachePath)
	return err == nil && string(data) == sha1
}

func (s *FSStorage) WriteCache(ctx context.Context, release string, pkgName string, sha1 string) error {
	if release == "" {
		return fmt.Errorf("cache release required")
	}
	cachePath := filepath.Join(s.Root, "manpages", release, ".cache", pkgName)
	return s.writeFileAbsolute(cachePath, []byte(sha1))
}

func (s *FSStorage) writeFile(destPath string, content []byte) error {
	fullPath := filepath.Join(s.Root, filepath.FromSlash(destPath))
	return s.writeFileAbsolute(fullPath, content)
}

func (s *FSStorage) writeFileAbsolute(fullPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	// Remove any existing file or symlink so os.WriteFile does not
	// follow a stale symlink left by a different package.
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing: %w", err)
	}
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (s *FSStorage) writeSymlink(destPath string, target string) error {
	fullPath := filepath.Join(s.Root, filepath.FromSlash(destPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing: %w", err)
	}
	if err := os.Symlink(target, fullPath); err != nil {
		return fmt.Errorf("symlink: %w", err)
	}
	return nil
}
