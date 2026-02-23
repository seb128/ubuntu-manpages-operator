package transform

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrepareFragment(t *testing.T) {
	input := "<h1>ls(1)</h1><p>List files</p>"

	output, err := PrepareFragment("jammy", input, &ManpageMeta{PackageName: "coreutils", SourcePackage: "coreutils"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(output, "<!--META:") {
		t.Fatalf("expected META comment prefix, got: %s", output[:40])
	}
	if !strings.Contains(output, "List files") {
		t.Fatalf("expected body in output")
	}

	meta := extractMeta(t, output)
	if meta.Title != "ls(1)" {
		t.Fatalf("expected title ls(1), got: %s", meta.Title)
	}
	if meta.PackageURL != "https://launchpad.net/ubuntu/jammy/+package/coreutils" {
		t.Fatalf("expected package URL, got: %s", meta.PackageURL)
	}
	if strings.Contains(output, "<h1>") {
		t.Fatalf("first heading should be removed")
	}
}

func TestPrepareFragmentWithSource(t *testing.T) {
	input := "<h1>apt-get(8)</h1><p>Package manager</p>"

	output, err := PrepareFragment("jammy", input, &ManpageMeta{
		PackageName:   "apt",
		SourcePackage: "apt-src",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.SourceURL != "https://launchpad.net/ubuntu/jammy/+source/apt-src" {
		t.Fatalf("expected source URL, got: %s", meta.SourceURL)
	}
}

func TestPrepareFragmentUsesNameSection(t *testing.T) {
	input := "<h1>NAME</h1><p>oomctl - Analyze state</p>"

	output, err := PrepareFragment("jammy", input, &ManpageMeta{PackageName: "systemd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "oomctl" {
		t.Fatalf("expected NAME section title, got: %s", meta.Title)
	}
	if meta.Description != "Analyze state" {
		t.Fatalf("expected description, got: %s", meta.Description)
	}
}

func TestPrepareFragmentUsesNameSectionPreCode(t *testing.T) {
	input := "<h1>NAME</h1>\n<pre><code>distrobox create\ndistrobox-create</code></pre>\n<h1>DESCRIPTION</h1><p>desc</p>"

	output, err := PrepareFragment("jammy", input, &ManpageMeta{PackageName: "distrobox"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "distrobox create" {
		t.Fatalf("expected first line of pre/code NAME, got: %s", meta.Title)
	}
}

func TestPrepareFragmentMultiLineNameDescription(t *testing.T) {
	input := "<h1>NAME</h1>\n<p>anc-describe-instance - describes all values for a specific Cloud\nServer ID from the Atlantic.Net Cloud.</p>\n<h1>DESCRIPTION</h1><p>details</p>"

	output, err := PrepareFragment("jammy", input, &ManpageMeta{PackageName: "anc-api-tools"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "anc-describe-instance" {
		t.Fatalf("expected title anc-describe-instance, got: %s", meta.Title)
	}
	expected := "describes all values for a specific Cloud Server ID from the Atlantic.Net Cloud."
	if meta.Description != expected {
		t.Fatalf("expected full multi-line description, got: %s", meta.Description)
	}
}

func TestPrepareFragmentUsesTranslatedNameSection(t *testing.T) {
	input := "<h1>BEZEICHNUNG</h1><p>apt-get - APT-Paketverwaltung</p><h1>ÜBERSICHT</h1><p>synopsis</p>"

	output, err := PrepareFragment("jammy", input, &ManpageMeta{PackageName: "apt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "apt-get" {
		t.Fatalf("expected translated NAME section title, got: %s", meta.Title)
	}
	if meta.Description != "APT-Paketverwaltung" {
		t.Fatalf("expected translated description, got: %s", meta.Description)
	}
}

func TestPrepareFragmentShiftsHeadings(t *testing.T) {
	input := "<h1>NAME</h1><p>test - a test page</p><h1>SYNOPSIS</h1><p>synopsis</p><h2>Options</h2><p>options</p>"

	output, err := PrepareFragment("jammy", input, &ManpageMeta{PackageName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "<h2") || !strings.Contains(output, "SYNOPSIS</h2>") {
		t.Fatalf("expected SYNOPSIS shifted to h2")
	}
	if !strings.Contains(output, "<h3") || !strings.Contains(output, "Options</h3>") {
		t.Fatalf("expected Options shifted to h3")
	}
	if strings.Contains(output, "<h1") {
		t.Fatalf("section h1 tags should be shifted to h2")
	}
}

func TestPrepareFragmentMandocNameSection(t *testing.T) {
	input := `<section class="Sh">
<h1 class="Sh" id="NAME"><a class="permalink" href="#NAME">NAME</a></h1>
<p class="Pp">groff - GNU roff language reference</p>
</section>
<section class="Sh">
<h1 class="Sh" id="Description"><a class="permalink" href="#Description">Description</a></h1>
<p class="Pp">groff is a typesetting system.</p>
</section>`

	output, err := PrepareFragment("jammy", input, &ManpageMeta{PackageName: "groff"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "groff" {
		t.Fatalf("expected title groff, got: %s", meta.Title)
	}
	if meta.Description != "GNU roff language reference" {
		t.Fatalf("expected description, got: %s", meta.Description)
	}
	if strings.Contains(output, "id=\"NAME\"") {
		t.Fatalf("NAME section heading should be removed")
	}
}

func TestPrepareFragmentHTMLEntityDash(t *testing.T) {
	input := `<section class="Sh">
<h1 class="Sh" id="NAME"><a class="permalink" href="#NAME">NAME</a></h1>
<p class="Pp">a2ps-lpr-wrapper &#x2014; lp/lpr wrapper script for GNU a2ps on Debian</p>
</section>`

	output, err := PrepareFragment("bionic", input, &ManpageMeta{PackageName: "a2ps"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "a2ps-lpr-wrapper" {
		t.Fatalf("expected title a2ps-lpr-wrapper, got: %s", meta.Title)
	}
	if meta.Description != "lp/lpr wrapper script for GNU a2ps on Debian" {
		t.Fatalf("expected description, got: %s", meta.Description)
	}
}

func TestPrepareFragmentTrailingEmDash(t *testing.T) {
	input := `<section class="Sh">
<h1 class="Sh" id="NAME"><a class="permalink" href="#NAME">NAME</a></h1>
<p class="Pp"><b class="Nm">systemd.cron</b> &#8212;</p>
</section>
<section class="Sh">
<h1 class="Sh" id="SYNOPSIS"><a class="permalink" href="#SYNOPSIS">SYNOPSIS</a></h1>
<p class="Pp">synopsis</p>
</section>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "systemd-cron"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "systemd.cron" {
		t.Fatalf("expected title systemd.cron, got: %s", meta.Title)
	}
	if meta.Description != "" {
		t.Fatalf("expected empty description, got: %s", meta.Description)
	}
}

func TestPrepareFragmentRemovesEmptySections(t *testing.T) {
	input := `<section class="Sh">
<h1 class="Sh" id="NAME"><a class="permalink" href="#NAME">NAME</a></h1>
<p class="Pp">pdftex - test</p>
</section>
<section class="Sh">
<h1 class="Sh" id="NOTES"><a class="permalink" href="#NOTES">NOTES</a></h1>
</section>
<section class="Sh">
<h1 class="Sh" id="BUGS"><a class="permalink" href="#BUGS">BUGS</a></h1>
<p class="Pp">Some bug text.</p>
</section>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "texlive-binaries"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(output, "NOTES") {
		t.Fatalf("empty NOTES section should be removed, got: %s", output)
	}
	if !strings.Contains(output, "BUGS") {
		t.Fatalf("non-empty BUGS section should be kept")
	}
	if !strings.Contains(output, "Some bug text.") {
		t.Fatalf("BUGS content should be kept")
	}
}

func TestPrepareFragmentStripsLeadingBreaks(t *testing.T) {
	// Mandoc leaves <br/> between the removed NAME section and the next section.
	input := `<section class="Sh">
<h1 class="Sh" id="NAME"><a class="permalink" href="#NAME">NAME</a></h1>
<p class="Pp">a2query - retrieve runtime configuration</p>
</section>
<br/>

<section class="Sh">
<h1 class="Sh" id="SYNOPSIS"><a class="permalink" href="#SYNOPSIS">SYNOPSIS</a></h1>
<p class="Pp"><b>a2query</b> [-h]</p>
</section>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "apache2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := strings.SplitN(output, "\n", 2)[1] // skip META line
	if strings.HasPrefix(body, "<br") {
		t.Fatalf("leading <br/> should be stripped, got: %s", body[:40])
	}
	if !strings.Contains(output, "SYNOPSIS") {
		t.Fatalf("SYNOPSIS section should be kept")
	}
}

func TestPrepareFragmentDoubleDashSeparator(t *testing.T) {
	input := "<h1>NAME</h1><p>apt-file -- APT package searching utility -- command-line</p>"

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "apt-file"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "apt-file" {
		t.Fatalf("expected title apt-file, got: %s", meta.Title)
	}
	if meta.Description != "APT package searching utility -- command-line" {
		t.Fatalf("expected description, got: %s", meta.Description)
	}
}

func TestPrepareFragmentTrailingDoubleDash(t *testing.T) {
	input := "<h1>NAME</h1><p>foo --</p><h1>SYNOPSIS</h1><p>foo</p>"

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "foo" {
		t.Fatalf("expected title foo, got: %s", meta.Title)
	}
	if meta.Description != "" {
		t.Fatalf("expected empty description, got: %s", meta.Description)
	}
}

func TestPrepareFragmentTitleCaseNameKeyword(t *testing.T) {
	// Spanish "Nombre" (title-case) should be recognized like "NOMBRE".
	input := `<section class="Sh">
<h1 class="Sh" id="Nombre">Nombre</h1>
<p class="Pp">mattrib - change MSDOS file attribute flags</p>
</section>
<section class="Sh">
<h1 class="Sh" id="Nota">Nota</h1>
<p>warning text</p>
</section>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "manpages-es"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "mattrib" {
		t.Fatalf("expected title mattrib, got: %s", meta.Title)
	}
	if meta.Description != "change MSDOS file attribute flags" {
		t.Fatalf("expected description, got: %s", meta.Description)
	}
}

func TestPrepareFragmentFrenchNom(t *testing.T) {
	input := `<section class="Sh">
<h1 class="Sh" id="Nom">Nom</h1>
<p class="Pp">multistrap - multiple repository bootstraps</p>
</section>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "multistrap"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "multistrap" {
		t.Fatalf("expected title multistrap, got: %s", meta.Title)
	}
}

func TestPrepareFragmentMultilineH1Collapsed(t *testing.T) {
	// GRASS GIS pages have h1 text split across lines.
	input := "<h1>Cairo\n  DISPLAY DRIVER</h1><p>body text</p>"

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "grass-doc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "Cairo DISPLAY DRIVER" {
		t.Fatalf("expected collapsed title, got: %s", meta.Title)
	}
}

func TestPrepareFragmentEmptyNameFallsThrough(t *testing.T) {
	// NAME section exists but has no content. Should NOT return "NAME".
	input := `<h1>NAME</h1>
<h1>DESCRIPTION</h1><p>Perl data type class</p>
<h1>Details</h1><p>more</p>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "zoneminder"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	// Should skip NAME (keyword) and DESCRIPTION (section heading), use "Details".
	if meta.Title != "Details" {
		t.Fatalf("expected fallback to non-section h1, got: %s", meta.Title)
	}
}

func TestPrepareFragmentSkipsSYNOPSISFallback(t *testing.T) {
	// No NAME section, first h1 is SYNOPSIS. Should not use it as title.
	input := `<h1>SYNOPSIS</h1><p>usage: foo</p><h1>FooTool</h1><p>details</p>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "FooTool" {
		t.Fatalf("expected FooTool as fallback, got: %s", meta.Title)
	}
}

func TestPrepareFragmentAllSectionHeadingsFallback(t *testing.T) {
	// All h1's are section headings. Should return "Ubuntu Manpage".
	input := `<h1>SYNOPSIS</h1><p>usage</p><h1>DESCRIPTION</h1><p>desc</p>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "Ubuntu Manpage" {
		t.Fatalf("expected Ubuntu Manpage fallback, got: %s", meta.Title)
	}
}

func TestPrepareFragmentFilenameFallback(t *testing.T) {
	// No usable h1 at all — should derive title from filename.
	input := "<p>Some content without any headings</p>"

	output, err := PrepareFragment("noble", input, &ManpageMeta{
		PackageName: "xemacs21",
		Filename:    "auctex.texi.gz",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "auctex.texi" {
		t.Fatalf("expected filename-derived title, got: %s", meta.Title)
	}
}

func TestPrepareFragmentFilenameFallbackManpage(t *testing.T) {
	input := "<h1>SYNOPSIS</h1><p>usage</p><h1>DESCRIPTION</h1><p>desc</p>"

	output, err := PrepareFragment("noble", input, &ManpageMeta{
		PackageName: "bib2gls",
		Filename:    "bib2gls.1.gz",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.Title != "bib2gls" {
		t.Fatalf("expected filename-derived title, got: %s", meta.Title)
	}
}

func TestPrepareFragmentIncludesTOC(t *testing.T) {
	input := "<h1>NAME</h1><p>test - a tool</p><h1>SYNOPSIS</h1><p>usage</p><h2>Options</h2><p>opts</p>"

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := extractMeta(t, output)
	if meta.TOC == "" {
		t.Fatalf("expected non-empty TOC in metadata")
	}
	if !strings.Contains(meta.TOC, `href="#synopsis"`) {
		t.Fatalf("expected TOC to contain synopsis link, got: %s", meta.TOC)
	}
}

func TestPrepareFragmentSectionWrapping(t *testing.T) {
	// Mandoc-style input with sections.
	input := `<section class="Sh">
<h1 class="Sh" id="NAME"><a class="permalink" href="#NAME">NAME</a></h1>
<p class="Pp">test - a tool</p>
</section>
<section class="Sh">
<h1 class="Sh" id="SYNOPSIS"><a class="permalink" href="#SYNOPSIS">SYNOPSIS</a></h1>
<p class="Pp">test [options]</p>
</section>`

	output, err := PrepareFragment("noble", input, &ManpageMeta{PackageName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After ingestion the section should have mp-section class.
	if !strings.Contains(output, "mp-section") {
		t.Fatalf("expected mp-section class in output, got:\n%s", output)
	}
}

func extractMeta(t *testing.T, fragment string) FragmentMeta {
	t.Helper()
	start := strings.Index(fragment, "<!--META:")
	end := strings.Index(fragment, "-->")
	if start == -1 || end == -1 {
		t.Fatalf("no META comment found in fragment")
	}
	jsonStr := fragment[len("<!--META:"):end]
	var meta FragmentMeta
	if err := json.Unmarshal([]byte(jsonStr), &meta); err != nil {
		t.Fatalf("failed to parse META JSON: %v", err)
	}
	return meta
}
