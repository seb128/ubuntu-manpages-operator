package pipeline

import "testing"

func TestParseManpagePath(t *testing.T) {
	paths, err := ParseManpagePath("jammy", "./usr/share/man/man1/ls.1.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths.HTMLPath != "manpages/jammy/man1/ls.1.html" {
		t.Fatalf("unexpected html path: %s", paths.HTMLPath)
	}
	if paths.GzipPath != "manpages.gz/jammy/man1/ls.1.gz" {
		t.Fatalf("unexpected gzip path: %s", paths.GzipPath)
	}
	if paths.Section != 1 {
		t.Fatalf("unexpected section: %d", paths.Section)
	}
}

func TestParseSectionFromFilenameFallback(t *testing.T) {
	paths, err := ParseManpagePath("jammy", "./usr/share/man/ls.1.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths.Section != 1 {
		t.Fatalf("expected section 1, got %d", paths.Section)
	}
}

func TestParseManpagePathEnglishLanguage(t *testing.T) {
	paths, err := ParseManpagePath("noble", "./usr/share/man/man1/ls.1.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths.Language != "" {
		t.Fatalf("expected empty language for English, got %q", paths.Language)
	}
	if paths.Section != 1 {
		t.Fatalf("expected section 1, got %d", paths.Section)
	}
}

func TestParseManpagePathTranslatedLanguage(t *testing.T) {
	paths, err := ParseManpagePath("noble", "./usr/share/man/de/man1/ls.1.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths.Language != "de" {
		t.Fatalf("expected language %q, got %q", "de", paths.Language)
	}
	if paths.Section != 1 {
		t.Fatalf("expected section 1, got %d", paths.Section)
	}
	if paths.HTMLPath != "manpages/noble/de/man1/ls.1.html" {
		t.Fatalf("unexpected html path: %s", paths.HTMLPath)
	}
}

func TestParseManpagePathTranslatedLanguageComplex(t *testing.T) {
	paths, err := ParseManpagePath("noble", "./usr/share/man/zh_CN/man8/apt-get.8.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths.Language != "zh_CN" {
		t.Fatalf("expected language %q, got %q", "zh_CN", paths.Language)
	}
	if paths.Section != 8 {
		t.Fatalf("expected section 8, got %d", paths.Section)
	}
}
