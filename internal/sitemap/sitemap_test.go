package sitemap

import (
	"context"
	"encoding/xml"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSitemapGenerator_Generate(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a mock manpage tree.
	man1Dir := filepath.Join(dir, "manpages", "noble", "man1")
	man3Dir := filepath.Join(dir, "manpages", "noble", "man3")
	if err := os.MkdirAll(man1Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(man3Dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write some dummy HTML files.
	for _, name := range []string{"ls.1.html", "cat.1.html", "grep.1.html"} {
		if err := os.WriteFile(filepath.Join(man1Dir, name), []byte("<p>test</p>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(man3Dir, "printf.3.html"), []byte("<p>test</p>"), 0o644); err != nil {
		t.Fatal(err)
	}

	gen := &SitemapGenerator{
		Root:    dir,
		SiteURL: "https://manpages.ubuntu.com",
		Logger:  logger,
	}

	if err := gen.Generate(context.Background(), []string{"noble"}); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify sitemap index exists and is valid XML.
	indexPath := filepath.Join(dir, "sitemaps", "sitemap-index.xml")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("missing sitemap index: %v", err)
	}
	var idx sitemapIndex
	if err := xml.Unmarshal(indexData, &idx); err != nil {
		t.Fatalf("invalid sitemap index XML: %v", err)
	}

	// Should have: static, noble-man1, noble-man3.
	if len(idx.Sitemaps) < 3 {
		t.Errorf("expected at least 3 sitemaps in index, got %d", len(idx.Sitemaps))
	}

	// Verify the man1 sitemap.
	man1Path := filepath.Join(dir, "sitemaps", "sitemap-noble-man1.xml")
	man1Data, err := os.ReadFile(man1Path)
	if err != nil {
		t.Fatalf("missing noble-man1 sitemap: %v", err)
	}
	var urlset sitemapURLSet
	if err := xml.Unmarshal(man1Data, &urlset); err != nil {
		t.Fatalf("invalid man1 sitemap XML: %v", err)
	}
	if len(urlset.URLs) != 3 {
		t.Errorf("expected 3 URLs in man1 sitemap, got %d", len(urlset.URLs))
	}

	// Verify URLs contain the expected base.
	for _, u := range urlset.URLs {
		if !strings.HasPrefix(u.Loc, "https://manpages.ubuntu.com/manpages/noble/man1/") {
			t.Errorf("unexpected URL: %s", u.Loc)
		}
	}

	// Verify static sitemap exists and has the homepage.
	staticPath := filepath.Join(dir, "sitemaps", "sitemap-static.xml")
	staticData, err := os.ReadFile(staticPath)
	if err != nil {
		t.Fatalf("missing static sitemap: %v", err)
	}
	if !strings.Contains(string(staticData), "https://manpages.ubuntu.com/") {
		t.Error("static sitemap missing homepage URL")
	}
}

func TestSitemapGenerator_LanguageSubdirectory(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a language subdirectory structure.
	langDir := filepath.Join(dir, "manpages", "noble", "de", "man1")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "ls.1.html"), []byte("<p>test</p>"), 0o644); err != nil {
		t.Fatal(err)
	}

	gen := &SitemapGenerator{
		Root:    dir,
		SiteURL: "https://manpages.ubuntu.com",
		Logger:  logger,
	}

	if err := gen.Generate(context.Background(), []string{"noble"}); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for the language-specific sitemap.
	langSitemapPath := filepath.Join(dir, "sitemaps", "sitemap-noble-de-man1.xml")
	data, err := os.ReadFile(langSitemapPath)
	if err != nil {
		t.Fatalf("missing language sitemap: %v", err)
	}

	if !strings.Contains(string(data), "/manpages/noble/de/man1/ls.1.html") {
		t.Error("language sitemap missing expected URL")
	}
}

func TestSitemapGenerator_EmptyRelease(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	gen := &SitemapGenerator{
		Root:    dir,
		SiteURL: "https://manpages.ubuntu.com",
		Logger:  logger,
	}

	// Should not fail for a nonexistent release directory.
	if err := gen.Generate(context.Background(), []string{"nonexistent"}); err != nil {
		t.Fatalf("Generate failed for empty release: %v", err)
	}

	// Index should still exist with just the static sitemap.
	indexPath := filepath.Join(dir, "sitemaps", "sitemap-index.xml")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("missing sitemap index: %v", err)
	}
}

func TestSplitURLs(t *testing.T) {
	urls := make([]sitemapURL, 5)
	for i := range urls {
		urls[i] = sitemapURL{Loc: "http://example.com/" + string(rune('a'+i))}
	}

	chunks := splitURLs(urls, 2)
	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 2 {
		t.Errorf("first chunk: expected 2 URLs, got %d", len(chunks[0]))
	}
	if len(chunks[2]) != 1 {
		t.Errorf("last chunk: expected 1 URL, got %d", len(chunks[2]))
	}

	// Under limit returns single chunk.
	single := splitURLs(urls, 10)
	if len(single) != 1 {
		t.Errorf("expected 1 chunk for under-limit, got %d", len(single))
	}
}
