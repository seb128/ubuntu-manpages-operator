package pipeline

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/canonical/ubuntu-manpages-operator/internal/transform"
)

type DebExtractor struct {
	WorkDir string
}

func NewDebExtractor(workDir string) *DebExtractor {
	return &DebExtractor{WorkDir: workDir}
}

func (e *DebExtractor) ExtractManpages(ctx context.Context, debPath string) ([]ManpageFile, func() error, error) {
	tempDir, err := os.MkdirTemp(e.WorkDir, "manpages-deb-")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp dir: %w", err)
	}

	cleanup := func() error {
		return os.RemoveAll(tempDir)
	}

	cmd := exec.CommandContext(ctx, "dpkg-deb", "-x", debPath, tempDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = cleanup()
		return nil, nil, fmt.Errorf("extract deb: %w: %s", err, strings.TrimSpace(string(output)))
	}

	manpages, err := findManpages(tempDir)
	if err != nil {
		_ = cleanup()
		return nil, nil, err
	}

	meta, err := readDebMetadata(ctx, debPath)
	if err != nil {
		_ = cleanup()
		return nil, nil, err
	}

	for i := range manpages {
		manpages[i].Meta = meta
	}

	return manpages, cleanup, nil
}

func readDebMetadata(ctx context.Context, debPath string) (transform.ManpageMeta, error) {
	cmd := exec.CommandContext(ctx, "dpkg-deb", "-f", debPath, "Package", "Version", "Source")
	output, err := cmd.Output()
	if err != nil {
		return transform.ManpageMeta{}, fmt.Errorf("read deb metadata: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	meta := transform.ManpageMeta{}
	if len(lines) > 0 {
		meta.PackageName = normalizePackageField(lines[0], "Package")
	}
	if len(lines) > 1 {
		meta.PackageVersion = strings.TrimSpace(lines[1])
	}
	if len(lines) > 2 {
		meta.SourcePackage = normalizeSourceField(lines[2])
	}
	if meta.SourcePackage == "" {
		meta.SourcePackage = meta.PackageName
	}
	return meta, nil
}

func normalizePackageField(value string, label string) string {
	value = strings.TrimSpace(value)
	prefix := label + ":"
	if strings.HasPrefix(value, prefix) {
		value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
	}
	return value
}

func normalizeSourceField(value string) string {
	value = normalizePackageField(value, "Source")
	if idx := strings.Index(value, " ("); idx > 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return value
}

func findManpages(root string) ([]ManpageFile, error) {
	var results []ManpageFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.Contains(path, "/man/") {
			return nil
		}
		if !strings.HasSuffix(path, ".gz") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("rel path: %w", err)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat manpage: %w", err)
		}

		item := ManpageFile{
			Path:         path,
			RelativePath: filepath.ToSlash(rel),
		}

		if info.Mode()&os.ModeSymlink != 0 {
			item.IsSymlink = true
			target, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("read symlink: %w", err)
			}
			item.SymlinkTarget = target
		}

		results = append(results, item)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk manpages: %w", err)
	}

	return results, nil
}
