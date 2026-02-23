package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAbsolute_OverwritesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nonexistent")
	dest := filepath.Join(dir, "output.html")

	// Create a dangling symlink at the destination.
	if err := os.Symlink(target, dest); err != nil {
		t.Fatal(err)
	}

	s := &FSStorage{}
	if err := s.writeFileAbsolute(dest, []byte("hello")); err != nil {
		t.Fatalf("writeFileAbsolute failed: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}

	// Ensure it's a regular file, not a symlink.
	info, err := os.Lstat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("expected regular file, got symlink")
	}
}

func TestWriteFileAbsolute_OverwritesCircularSymlink(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.html")
	b := filepath.Join(dir, "b.html")

	// Create circular symlinks: a -> b -> a
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}

	s := &FSStorage{}
	if err := s.writeFileAbsolute(a, []byte("content")); err != nil {
		t.Fatalf("writeFileAbsolute failed: %v", err)
	}

	got, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "content" {
		t.Fatalf("got %q, want %q", got, "content")
	}
}
