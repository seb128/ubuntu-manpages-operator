package transform

import (
	"strings"
	"testing"
)

func TestRewriteLinksMandocXref(t *testing.T) {
	input := `See <i>groff_char</i>(7) and <b>troff</b>(1) for details.`
	output, err := RewriteLinks("jammy", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, `<a href="/manpages/jammy/man7/groff_char.7.html"><i>groff_char</i>(7)</a>`) {
		t.Fatalf("expected mandoc xref link for groff_char(7), got: %s", output)
	}
	if !strings.Contains(output, `<a href="/manpages/jammy/man1/troff.1.html"><b>troff</b>(1)</a>`) {
		t.Fatalf("expected mandoc xref link for troff(1), got: %s", output)
	}
}

func TestRewriteLinksMandocXrefBoldSection(t *testing.T) {
	input := `<p class="Pp"><b>plc</b>(<b>1</b>), <b>amplist</b>(<b>1</b>),
    <b>amprate</b>(<b>1</b>)</p>`
	output, err := RewriteLinks("noble", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, `<a href="/manpages/noble/man1/plc.1.html"><b>plc</b>(<b>1</b>)</a>`) {
		t.Fatalf("expected bold-section xref link for plc(1), got: %s", output)
	}
	if !strings.Contains(output, `<a href="/manpages/noble/man1/amplist.1.html"><b>amplist</b>(<b>1</b>)</a>`) {
		t.Fatalf("expected bold-section xref link for amplist(1), got: %s", output)
	}
	if !strings.Contains(output, `<a href="/manpages/noble/man1/amprate.1.html"><b>amprate</b>(<b>1</b>)</a>`) {
		t.Fatalf("expected bold-section xref link for amprate(1), got: %s", output)
	}
}

func TestRewriteLinksMixedTags(t *testing.T) {
	input := `<i>groff_char</i>(<b>7</b>)`
	output, err := RewriteLinks("jammy", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `<a href="/manpages/jammy/man7/groff_char.7.html"><i>groff_char</i>(<b>7</b>)</a>`
	if !strings.Contains(output, expected) {
		t.Fatalf("expected mixed-tag xref link, got: %s", output)
	}
}

func TestRewriteLinksPandocFileURL(t *testing.T) {
	input := `<a href="file:///usr/share/man/man1/ls.1.gz">ls(1)</a>`
	output, err := RewriteLinks("jammy", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, `/manpages/jammy/man1/ls.1.html`) {
		t.Fatalf("expected pandoc file:// link rewritten, got: %s", output)
	}
}

func TestRewriteLinksXrTag(t *testing.T) {
	input := `See <a class="Xr">asfxload(1)</a> for details.`
	output, err := RewriteLinks("jammy", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `<a class="Xr" href="/manpages/jammy/man1/asfxload.1.html">asfxload(1)</a>`
	if !strings.Contains(output, expected) {
		t.Fatalf("expected Xr tag rewritten, got: %s", output)
	}
}

func TestRewriteLinksXrTagWithHref(t *testing.T) {
	input := `See <a class="Xr" href="syslog.3.html">syslog(3)</a>.`
	output, err := RewriteLinks("noble", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `<a class="Xr" href="/manpages/noble/man3/syslog.3.html">syslog(3)</a>`
	if !strings.Contains(output, expected) {
		t.Fatalf("expected Xr tag href rewritten, got: %s", output)
	}
}

func TestRewriteLinksPlainText(t *testing.T) {
	input := `<p>hugo(1), hugo-list-all(1), hugo-list-drafts(1)</p>`
	output, err := RewriteLinks("noble", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, `<a href="/manpages/noble/man1/hugo.1.html">hugo(1)</a>`) {
		t.Fatalf("expected plain text hugo(1) linked, got: %s", output)
	}
	if !strings.Contains(output, `<a href="/manpages/noble/man1/hugo-list-all.1.html">hugo-list-all(1)</a>`) {
		t.Fatalf("expected plain text hugo-list-all(1) linked, got: %s", output)
	}
	if !strings.Contains(output, `<a href="/manpages/noble/man1/hugo-list-drafts.1.html">hugo-list-drafts(1)</a>`) {
		t.Fatalf("expected plain text hugo-list-drafts(1) linked, got: %s", output)
	}
}

func TestRewriteLinksPlainTextSkipsAnchors(t *testing.T) {
	input := `<a href="/manpages/noble/man1/ls.1.html">ls(1)</a>, hugo(1)`
	output, err := RewriteLinks("noble", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ls(1) inside existing anchor should not be double-wrapped.
	if strings.Contains(output, `<a href="/manpages/noble/man1/ls.1.html"><a href=`) {
		t.Fatalf("ls(1) inside anchor should not be re-linked, got: %s", output)
	}
	// hugo(1) outside anchor should be linked.
	if !strings.Contains(output, `<a href="/manpages/noble/man1/hugo.1.html">hugo(1)</a>`) {
		t.Fatalf("expected plain text hugo(1) linked, got: %s", output)
	}
}

func TestRewriteLinksPermalinkUnchanged(t *testing.T) {
	input := `<dt id="-D"><a class="permalink" href="#-D"><b>-D</b></a> [<i>file</i>]</dt>`
	output, err := RewriteLinks("jammy", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != input {
		t.Fatalf("permalink anchor was modified, got: %s", output)
	}
}
