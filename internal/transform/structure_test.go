package transform

import (
	"strings"
	"testing"
)

func TestWrapSectionsMandoc(t *testing.T) {
	input := `<section class="Sh">
<h2 class="Sh" id="SYNOPSIS">SYNOPSIS</h2>
<p>content1</p>
</section>
<section class="Sh">
<h2 class="Sh" id="DESCRIPTION">DESCRIPTION</h2>
<p>content2</p>
</section>`

	output := wrapSections(input)

	// h2 should be extracted before the section.
	if !strings.Contains(output, `<h2 class="Sh" id="SYNOPSIS">SYNOPSIS</h2>`) {
		t.Fatalf("expected h2 extracted before section, got:\n%s", output)
	}
	// Section should have mp-section class added.
	if !strings.Contains(output, `class="Sh mp-section"`) {
		t.Fatalf("expected mp-section class on section, got:\n%s", output)
	}
	// h2 should NOT be inside the section anymore.
	if strings.Contains(output, `<section class="Sh mp-section">
<h2`) {
		t.Fatalf("h2 should be outside the section, got:\n%s", output)
	}
}

func TestWrapSectionsPandoc(t *testing.T) {
	input := `<h2>SYNOPSIS</h2>
<p>content1</p>
<h2>DESCRIPTION</h2>
<p>content2</p>`

	output := wrapSections(input)

	if !strings.Contains(output, `<div class="mp-section">`) {
		t.Fatalf("expected mp-section wrapper div, got:\n%s", output)
	}
	if !strings.Contains(output, `<h2>SYNOPSIS</h2>`) {
		t.Fatalf("expected h2 preserved, got:\n%s", output)
	}
	// Content should be inside wrapper div.
	if !strings.Contains(output, "<div class=\"mp-section\">\n<p>content1</p>\n</div>") {
		t.Fatalf("expected content1 wrapped, got:\n%s", output)
	}
}

func TestWrapSectionsPandocPreservesLeadingContent(t *testing.T) {
	input := `<p>intro</p>
<h2>SYNOPSIS</h2>
<p>content</p>`

	output := wrapSections(input)

	// Content before first h2 should not be wrapped.
	if !strings.HasPrefix(output, "<p>intro</p>") {
		t.Fatalf("expected leading content preserved unwrapped, got:\n%s", output)
	}
}

func TestWrapSectionsNoHeadings(t *testing.T) {
	input := "<p>just some text</p>"
	output := wrapSections(input)
	if output != input {
		t.Fatalf("expected no change, got:\n%s", output)
	}
}

func TestWrapSectionsH3Only(t *testing.T) {
	// Pages with only .Ss (subsection) macros produce <section><h3> after
	// heading shift, with no <h2> elements.
	input := `<section class="Ss">
<h3 class="Ss" id="subsection1">Subsection 1</h3>
<p>content1</p>
</section>
<section class="Ss">
<h3 class="Ss" id="subsection2">Subsection 2</h3>
<p>content2</p>
</section>`

	output := wrapSections(input)

	if !strings.Contains(output, "mp-section") {
		t.Fatalf("expected mp-section class for h3-only pages, got:\n%s", output)
	}
	// h3 should be extracted before the section.
	if !strings.Contains(output, `<h3 class="Ss" id="subsection1">Subsection 1</h3>`) {
		t.Fatalf("expected h3 preserved, got:\n%s", output)
	}
}
