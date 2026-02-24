package sitemap

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxSitemapURLs = 50000

type sitemapURL struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapIndex struct {
	XMLName  xml.Name          `xml:"sitemapindex"`
	XMLNS    string            `xml:"xmlns,attr"`
	Sitemaps []sitemapIndexRef `xml:"sitemap"`
}

type sitemapIndexRef struct {
	XMLName xml.Name `xml:"sitemap"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod,omitempty"`
}

// SitemapGenerator creates sitemap XML files by walking the manpages directory tree.
type SitemapGenerator struct {
	Root    string // PublicHTMLDir
	SiteURL string // e.g. "https://manpages.ubuntu.com"
	Logger  *slog.Logger
}

// Generate creates sitemap files for all releases. It writes per-release-per-section
// sitemaps plus a sitemap index to {Root}/sitemaps/.
func (g *SitemapGenerator) Generate(ctx context.Context, releases []string) error {
	sitemapDir := filepath.Join(g.Root, "sitemaps")
	if err := os.MkdirAll(sitemapDir, 0o755); err != nil {
		return fmt.Errorf("create sitemaps dir: %w", err)
	}

	now := time.Now().UTC().Format("2006-01-02")
	var indexRefs []sitemapIndexRef

	// Static pages sitemap.
	staticURLs := []sitemapURL{
		{Loc: g.SiteURL + "/", LastMod: now},
		{Loc: g.SiteURL + "/search", LastMod: now},
		{Loc: g.SiteURL + "/manpages/", LastMod: now},
	}
	for _, rel := range releases {
		staticURLs = append(staticURLs, sitemapURL{
			Loc:     g.SiteURL + "/manpages/" + rel + "/",
			LastMod: now,
		})
	}
	staticFile := "sitemap-static.xml"
	if err := g.writeSitemap(filepath.Join(sitemapDir, staticFile), staticURLs); err != nil {
		return fmt.Errorf("write static sitemap: %w", err)
	}
	indexRefs = append(indexRefs, sitemapIndexRef{
		Loc:     g.SiteURL + "/sitemaps/" + staticFile,
		LastMod: now,
	})

	// Per-release, per-section sitemaps.
	for _, release := range releases {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		releaseDir := filepath.Join(g.Root, "manpages", release)
		entries, err := os.ReadDir(releaseDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read release dir %s: %w", release, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			// Match man sections (man1..man9) and language subdirectories.
			sectionDir := filepath.Join(releaseDir, name)
			refs, err := g.generateSection(ctx, sitemapDir, release, name, sectionDir)
			if err != nil {
				g.Logger.Warn("sitemap section error", "release", release, "dir", name, "error", err)
				continue
			}
			indexRefs = append(indexRefs, refs...)
		}
	}

	// Write the sitemap index.
	idx := sitemapIndex{
		XMLNS:    "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: indexRefs,
	}
	indexPath := filepath.Join(sitemapDir, "sitemap-index.xml")
	return writeXML(indexPath, idx)
}

// generateSection creates sitemaps for a single directory under a release.
// For man section directories (man1-man9) it walks .html files directly.
// For language directories, it recurses into their man sections.
func (g *SitemapGenerator) generateSection(ctx context.Context, sitemapDir, release, dirName, dirPath string) ([]sitemapIndexRef, error) {
	if strings.HasPrefix(dirName, "man") {
		return g.generateManSection(ctx, sitemapDir, release, dirName, dirPath, "")
	}

	// Language subdirectory — look for man sections within it.
	var refs []sitemapIndexRef
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "man") {
			continue
		}
		sectionDir := filepath.Join(dirPath, entry.Name())
		r, err := g.generateManSection(ctx, sitemapDir, release, entry.Name(), sectionDir, dirName)
		if err != nil {
			g.Logger.Warn("sitemap lang section error", "release", release, "lang", dirName, "section", entry.Name(), "error", err)
			continue
		}
		refs = append(refs, r...)
	}
	return refs, nil
}

func (g *SitemapGenerator) generateManSection(ctx context.Context, sitemapDir, release, section, sectionDir, lang string) ([]sitemapIndexRef, error) {
	entries, err := os.ReadDir(sectionDir)
	if err != nil {
		return nil, err
	}

	var urls []sitemapURL
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".html") {
			continue
		}
		var urlPath string
		if lang != "" {
			urlPath = fmt.Sprintf("/manpages/%s/%s/%s/%s", release, lang, section, name)
		} else {
			urlPath = fmt.Sprintf("/manpages/%s/%s/%s", release, section, name)
		}

		var lastmod string
		if info, err := entry.Info(); err == nil {
			lastmod = info.ModTime().UTC().Format("2006-01-02")
		}

		urls = append(urls, sitemapURL{
			Loc:     g.SiteURL + urlPath,
			LastMod: lastmod,
		})
	}

	if len(urls) == 0 {
		return nil, nil
	}

	// Split into chunks if exceeding the limit.
	var refs []sitemapIndexRef
	now := time.Now().UTC().Format("2006-01-02")

	chunks := splitURLs(urls, maxSitemapURLs)
	for i, chunk := range chunks {
		var filename string
		if lang != "" {
			filename = fmt.Sprintf("sitemap-%s-%s-%s", release, lang, section)
		} else {
			filename = fmt.Sprintf("sitemap-%s-%s", release, section)
		}
		if len(chunks) > 1 {
			filename = fmt.Sprintf("%s-%d", filename, i+1)
		}
		filename += ".xml"

		if err := g.writeSitemap(filepath.Join(sitemapDir, filename), chunk); err != nil {
			return nil, err
		}
		refs = append(refs, sitemapIndexRef{
			Loc:     g.SiteURL + "/sitemaps/" + filename,
			LastMod: now,
		})
	}

	return refs, nil
}

func (g *SitemapGenerator) writeSitemap(path string, urls []sitemapURL) error {
	urlset := sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}
	return writeXML(path, urlset)
}

func writeXML(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString(xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	return enc.Close()
}

func splitURLs(urls []sitemapURL, maxPerFile int) [][]sitemapURL {
	if len(urls) <= maxPerFile {
		return [][]sitemapURL{urls}
	}
	var chunks [][]sitemapURL
	for i := 0; i < len(urls); i += maxPerFile {
		end := i + maxPerFile
		if end > len(urls) {
			end = len(urls)
		}
		chunks = append(chunks, urls[i:end])
	}
	return chunks
}
