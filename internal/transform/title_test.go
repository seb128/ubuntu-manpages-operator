package transform

import (
	"strings"
	"testing"
)

func TestTitleFromFilename(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"ls.1.gz", "ls"},
		{"PDL::API.1p.gz", "PDL::API"},
		{"bib2gls.1.gz", "bib2gls"},
		{"auctex.texi.gz", "auctex.texi"},
		{"RELEASE.html", "RELEASE.html"},
		{"groff.7", "groff"},
	}
	for _, tt := range tests {
		got := titleFromFilename(tt.input)
		if got != tt.want {
			t.Errorf("titleFromFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSplitManpageTitleDoubleDash(t *testing.T) {
	name, desc := SplitManpageTitle("apt-file -- APT package searching utility")
	if name != "apt-file" {
		t.Fatalf("expected name apt-file, got: %s", name)
	}
	if desc != "APT package searching utility" {
		t.Fatalf("expected description, got: %s", desc)
	}
}

func TestIsNameKeywordCaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"NAME", true},
		{"Name", true},
		{"name", true},
		{"NOMBRE", true},
		{"Nombre", true},
		{"NOM", true},
		{"Nom", true},
		{"SYNOPSIS", false},
		{"Foo", false},
	}
	for _, tt := range tests {
		got := isNameKeyword(tt.input)
		if got != tt.want {
			t.Errorf("isNameKeyword(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPrepareFragmentDescriptionCapped(t *testing.T) {
	longDesc := strings.Repeat("word ", 100) // 500 chars
	input := "<h1>NAME</h1><p>tool - " + longDesc + "</p>"

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "tool"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "tool" {
		t.Fatalf("expected title tool, got: %s", meta.Title)
	}
	if len(meta.Description) > MaxDescriptionLen+10 { // +10 for ellipsis
		t.Fatalf("description too long (%d chars): %s", len(meta.Description), meta.Description[:50])
	}
	if !strings.HasSuffix(meta.Description, " …") {
		t.Fatalf("expected truncated description to end with ellipsis, got: ...%s", meta.Description[len(meta.Description)-10:])
	}
}
