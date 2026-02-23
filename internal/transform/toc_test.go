package transform

import (
	"strings"
	"testing"
)

func TestGenerateTOC(t *testing.T) {
	input := `<h2>SYNOPSIS</h2><p>usage</p><h3>Options</h3><p>opts</p>`

	body, toc := generateTOC(input)

	if !strings.Contains(body, `id="synopsis"`) {
		t.Fatalf("expected id on h2, got:\n%s", body)
	}
	if !strings.Contains(toc, `href="#synopsis"`) {
		t.Fatalf("expected TOC link for synopsis, got:\n%s", toc)
	}
	if !strings.Contains(toc, ">SYNOPSIS<") {
		t.Fatalf("expected SYNOPSIS text in TOC, got:\n%s", toc)
	}
	// h3 sub-headings should NOT appear in the TOC (only h2 section headings).
	if strings.Contains(toc, "Options") {
		t.Fatalf("h3 sub-heading should not appear in TOC, got:\n%s", toc)
	}
}

func TestGenerateTOCDuplicateSlugs(t *testing.T) {
	input := `<h2>BUGS</h2><p>text1</p><h2>BUGS</h2><p>text2</p>`

	body, toc := generateTOC(input)

	if !strings.Contains(body, `id="bugs"`) {
		t.Fatalf("expected first bugs id, got:\n%s", body)
	}
	if !strings.Contains(body, `id="bugs-1"`) {
		t.Fatalf("expected deduplicated second bugs id, got:\n%s", body)
	}
	if !strings.Contains(toc, `href="#bugs-1"`) {
		t.Fatalf("expected deduplicated TOC link, got:\n%s", toc)
	}
}

func TestGenerateTOCReplacesExistingID(t *testing.T) {
	input := `<h2 class="Sh" id="OLD_ID">SYNOPSIS</h2><p>content</p>`

	body, _ := generateTOC(input)

	if strings.Contains(body, "OLD_ID") {
		t.Fatalf("expected old id to be replaced, got:\n%s", body)
	}
	if !strings.Contains(body, `id="synopsis"`) {
		t.Fatalf("expected new slug id, got:\n%s", body)
	}
	// Class should be preserved.
	if !strings.Contains(body, `class="Sh"`) {
		t.Fatalf("expected class preserved, got:\n%s", body)
	}
}

func TestGenerateTOCNoHeadings(t *testing.T) {
	input := "<p>no headings here</p>"
	body, toc := generateTOC(input)
	if body != input {
		t.Fatalf("expected unmodified body, got:\n%s", body)
	}
	if toc != "" {
		t.Fatalf("expected empty TOC, got:\n%s", toc)
	}
}

func TestGenerateTOCHTMLEntitiesInHeading(t *testing.T) {
	input := `<h2>A &amp; B</h2><p>content</p>`

	body, toc := generateTOC(input)

	if !strings.Contains(body, `id="a-b"`) {
		t.Fatalf("expected slug from decoded text, got:\n%s", body)
	}
	if !strings.Contains(toc, `>A &amp; B<`) {
		t.Fatalf("expected HTML-escaped text in TOC, got:\n%s", toc)
	}
}

func TestGenerateTOCPermalinkHrefRewritten(t *testing.T) {
	// Mandoc emits uppercase permalink hrefs; generateTOC should rewrite
	// them to match the new lowercased slug ID.
	input := `<h2 id="SYNOPSIS"><a class="permalink" href="#SYNOPSIS">SYNOPSIS</a></h2><p>usage</p>`

	body, toc := generateTOC(input)

	if !strings.Contains(body, `id="synopsis"`) {
		t.Fatalf("expected lowercased id, got:\n%s", body)
	}
	if strings.Contains(body, `href="#SYNOPSIS"`) {
		t.Fatalf("permalink href should be rewritten to lowercase, got:\n%s", body)
	}
	if !strings.Contains(body, `href="#synopsis"`) {
		t.Fatalf("expected lowercased permalink href, got:\n%s", body)
	}
	if !strings.Contains(toc, `href="#synopsis"`) {
		t.Fatalf("expected TOC link, got:\n%s", toc)
	}
}

func TestGenerateTOCCollapseNewlines(t *testing.T) {
	// Mandoc may output heading text with newlines (e.g. "SEE\n  ALSO").
	input := "<h2>SEE\n  ALSO</h2><p>refs</p>"

	body, toc := generateTOC(input)

	if !strings.Contains(body, `id="see-also"`) {
		t.Fatalf("expected slug from collapsed text, got:\n%s", body)
	}
	if !strings.Contains(toc, ">SEE ALSO<") {
		t.Fatalf("expected collapsed TOC text, got:\n%s", toc)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"SYNOPSIS", "synopsis"},
		{"SEE ALSO", "see-also"},
		{"Options & Flags", "options-flags"},
		{"---leading---", "leading"},
		{"hello123", "hello123"},
		{"", ""},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
