package transform

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPipeline(t *testing.T) {
	rawHTML := `<section class="Sh">
<h1 class="Sh" id="NAME"><a class="permalink" href="#NAME">NAME</a></h1>
<p class="Pp">groff - GNU roff language reference</p>
</section>
<section class="Sh">
<h1 class="Sh" id="SYNOPSIS"><a class="permalink" href="#SYNOPSIS">SYNOPSIS</a></h1>
<p class="Pp">See <a class="Xr">troff(1)</a> for details.</p>
</section>`

	doc, err := Pipeline("jammy", rawHTML, &ManpageMeta{
		PackageName:   "groff",
		SourcePackage: "groff-base",
		Filename:      "groff.7.gz",
	})
	if err != nil {
		t.Fatalf("Pipeline: %v", err)
	}

	// Title should be extracted from NAME section.
	if doc.Title != "groff" {
		t.Fatalf("expected title groff, got: %s", doc.Title)
	}
	if doc.Desc != "GNU roff language reference" {
		t.Fatalf("expected description, got: %s", doc.Desc)
	}

	body := string(doc.Body)

	// Body should start with META comment.
	if !bytes.HasPrefix(doc.Body, []byte("<!--META:")) {
		t.Fatalf("expected META comment prefix")
	}

	// NAME section should be removed.
	if bytes.Contains(doc.Body, []byte("id=\"NAME\"")) {
		t.Fatalf("NAME section should be removed from body")
	}

	// Headings should be shifted (h1→h2).
	if bytes.Contains(doc.Body, []byte("<h1")) {
		t.Fatalf("h1 tags should be shifted to h2")
	}

	// Cross-reference should be rewritten.
	if !bytes.Contains(doc.Body, []byte("/manpages/jammy/man1/troff.1.html")) {
		t.Fatalf("expected cross-reference link rewritten")
	}

	// Sections should have mp-section class.
	if !bytes.Contains(doc.Body, []byte("mp-section")) {
		t.Fatalf("expected mp-section class")
	}

	// TOC should be populated.
	if doc.TOC == "" {
		t.Fatalf("expected non-empty TOC")
	}
	if !bytes.Contains([]byte(doc.TOC), []byte("synopsis")) {
		t.Fatalf("expected synopsis in TOC, got: %s", doc.TOC)
	}

	// Parse embedded META JSON to verify structure.
	metaStart := bytes.Index(doc.Body, []byte("<!--META:")) + len("<!--META:")
	metaEnd := bytes.Index(doc.Body, []byte("-->"))
	var fm FragmentMeta
	if err := json.Unmarshal(doc.Body[metaStart:metaEnd], &fm); err != nil {
		t.Fatalf("failed to parse META JSON: %v\nbody: %s", err, body)
	}
	if fm.Title != "groff" {
		t.Fatalf("META title mismatch: %s", fm.Title)
	}
	if fm.PackageURL != "https://launchpad.net/ubuntu/jammy/+package/groff" {
		t.Fatalf("expected package URL, got: %s", fm.PackageURL)
	}
	if fm.SourceURL != "https://launchpad.net/ubuntu/jammy/+source/groff-base" {
		t.Fatalf("expected source URL, got: %s", fm.SourceURL)
	}
}

func TestPipelineTitleFromFilename(t *testing.T) {
	// No NAME section, no usable h1 — should fall back to filename.
	rawHTML := "<p>Some content without headings</p>"

	doc, err := Pipeline("noble", rawHTML, &ManpageMeta{
		PackageName: "xemacs21",
		Filename:    "auctex.texi.gz",
	})
	if err != nil {
		t.Fatalf("Pipeline: %v", err)
	}

	if doc.Title != "auctex.texi" {
		t.Fatalf("expected filename-derived title, got: %s", doc.Title)
	}
}
