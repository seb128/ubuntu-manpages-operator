package pipeline

import (
	"bufio"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type ManpagePaths struct {
	HTMLPath string
	GzipPath string
	Section  int
	Language string
}

func ParseManpagePath(release string, relativePath string) (ManpagePaths, error) {
	idx := strings.Index(relativePath, "/man/")
	if idx == -1 {
		return ManpagePaths{}, fmt.Errorf("missing /man/ segment")
	}

	manRel := strings.TrimPrefix(relativePath[idx+len("/man/"):], "/")
	manRel = path.Clean(filepath.ToSlash(manRel))
	manRel = strings.TrimPrefix(manRel, "../")

	// Extract language: if the first segment is not a manN directory,
	// it's a language code (e.g. "de", "fr", "zh_CN").
	lang := ""
	if first, _, ok := strings.Cut(manRel, "/"); ok {
		if len(first) < 4 || first[:3] != "man" || first[3] < '0' || first[3] > '9' {
			lang = first
		}
	}

	base := strings.TrimSuffix(manRel, ".gz")
	section := parseSection(base)
	if lang != "" {
		// For translated pages, section dir is after the language prefix.
		section = parseSection(strings.TrimPrefix(base, lang+"/"))
	}

	htmlPath := path.Join("manpages", release, base) + ".html"
	gzipPath := path.Join("manpages.gz", release, base) + ".gz"

	return ManpagePaths{
		HTMLPath: htmlPath,
		GzipPath: gzipPath,
		Section:  section,
		Language: lang,
	}, nil
}

func ConvertSymlinkTarget(target string) string {
	target = path.Clean(filepath.ToSlash(target))
	return normalizeHTMLExt(target)
}

func ConvertSoTarget(target string) string {
	target = strings.TrimSpace(target)
	target = strings.TrimPrefix(target, "/")
	target = path.Clean(path.Join("..", target))
	return normalizeHTMLExt(target)
}

// normalizeHTMLExt strips a trailing ".gz" suffix and ensures the path
// ends with ".html".
func normalizeHTMLExt(p string) string {
	if before, ok := strings.CutSuffix(p, ".gz"); ok {
		p = before
	}
	if !strings.HasSuffix(p, ".html") {
		p += ".html"
	}
	return p
}

func parseSection(manRel string) int {
	parts := strings.Split(manRel, "/")
	if len(parts) == 0 {
		return 0
	}
	sectionDir := parts[0]
	sectionDir = strings.TrimPrefix(sectionDir, "man")
	section, err := strconv.Atoi(sectionDir)
	if err != nil {
		return parseSectionFromFilename(parts[len(parts)-1])
	}
	return section
}

func parseSectionFromFilename(name string) int {
	name = strings.TrimSuffix(name, ".gz")
	idx := strings.LastIndex(name, ".")
	if idx == -1 || idx == len(name)-1 {
		return 0
	}
	sectionStr := name[idx+1:]
	sectionStr = strings.TrimLeft(sectionStr, "man")
	if sectionStr == "" {
		return 0
	}
	section, err := strconv.Atoi(sectionStr[:1])
	if err != nil {
		return 0
	}
	return section
}

// DetectSoLink checks for a leading ".so" include directive and returns the target.
func DetectSoLink(path string) (string, bool, error) {
	reader, cleanup, err := openMaybeGzipped(path)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = cleanup() }()

	br := bufio.NewReader(reader)
	line, err := br.ReadString('\n')
	if err != nil && err.Error() != "EOF" {
		return "", false, fmt.Errorf("read manpage header: %w", err)
	}

	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, ".so ") {
		target := strings.TrimSpace(strings.TrimPrefix(line, ".so "))
		return target, true, nil
	}

	return "", false, nil
}
