package transform

import (
	"bytes"
	"fmt"
	"regexp"
)

var (
	// Pandoc-style file:// links (kept for backward compatibility).
	manLinkPattern = regexp.MustCompile(`file:///[^"\s]*/man([1-9])/([^"\s]+)\.[1-9](\.gz)?`)
	// Mandoc .Xr cross-references: <a class="Xr" [href="..."]>name(N)</a>.
	mandocXrTagPattern = regexp.MustCompile(`<a class="Xr"[^>]*>([a-zA-Z0-9._-]+)\(([1-9][a-z]*)\)</a>`)
	// xrefTextPattern matches name(section) in tag-stripped text.
	xrefTextPattern = regexp.MustCompile(`\b([a-zA-Z0-9][-a-zA-Z0-9._:+]*)\(([1-9][a-z]*)\)`)
	// inlineTagPattern matches opening and closing <b>/<i> tags with
	// optional attributes (e.g. <b class="Nm">).
	inlineTagPattern = regexp.MustCompile(`</?[bi]\b[^>]*>`)
	// trailingInlineOpen matches an inline opening tag at the end of a string.
	trailingInlineOpen = regexp.MustCompile(`<[bi]\b[^>]*>\z`)
	// leadingInlineClose matches an inline closing tag at the start of a string.
	leadingInlineClose = regexp.MustCompile(`\A</[bi]>`)
)

// RewriteLinks converts manpage cross-references into site-relative links.
// It handles both pandoc-style file:// URLs and mandoc-style <b>name</b>(N) references.
func RewriteLinks(release, html string) (string, error) {
	result, err := bRewriteLinks(release, []byte(html))
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// bRewriteLinks is the []byte implementation of RewriteLinks.
func bRewriteLinks(release string, html []byte) ([]byte, error) {
	if release == "" {
		return nil, fmt.Errorf("release is required")
	}
	// Pandoc-style file:// links.
	pandocReplacement := []byte("/manpages/" + release + "/man$1/$2.$1.html")
	html = manLinkPattern.ReplaceAll(html, pandocReplacement)
	// Mandoc .Xr tags: <a class="Xr" [href="..."]>name(N)</a> → linked.
	html = mandocXrTagPattern.ReplaceAllFunc(html, func(match []byte) []byte {
		parts := mandocXrTagPattern.FindSubmatch(match)
		name, section := string(parts[1]), string(parts[2])
		href := "/manpages/" + release + "/man" + section + "/" + name + "." + section + ".html"
		return []byte(`<a class="Xr" href="` + href + `">` + name + `(` + section + `)</a>`)
	})
	// Cross-references: strip inline formatting tags to find name(N)
	// patterns, then wrap them with links preserving original formatting.
	html = bRewriteXrefs(release, html)
	return html, nil
}

// bRewriteXrefs is the []byte implementation of rewriteXrefs.
func bRewriteXrefs(release string, html []byte) []byte {
	// Find all inline tag positions in one pass, then build the
	// tag-stripped text by skipping over them.  This avoids the O(n²)
	// cost of calling FindIndex at every byte position.
	tagLocs := inlineTagPattern.FindAllIndex(html, -1)
	var stripped bytes.Buffer
	var posMap []int
	tagIdx := 0
	for i := 0; i < len(html); {
		if tagIdx < len(tagLocs) && i == tagLocs[tagIdx][0] {
			i = tagLocs[tagIdx][1]
			tagIdx++
			continue
		}
		posMap = append(posMap, i)
		stripped.WriteByte(html[i])
		i++
	}

	strippedBytes := stripped.Bytes()
	locs := xrefTextPattern.FindAllSubmatchIndex(strippedBytes, -1)
	if len(locs) == 0 {
		return html
	}

	var b bytes.Buffer
	lastEnd := 0
	for _, loc := range locs {
		// Map stripped positions back to original HTML positions.
		origStart := posMap[loc[0]]
		origMatchEnd := posMap[loc[1]-1] + 1

		// Expand span to include surrounding inline formatting tags
		// so that e.g. <b>plc</b>(<b>1</b>) is fully wrapped.
		origStart = bExpandLeft(html, origStart, lastEnd)
		origEnd := bExpandRight(html, origMatchEnd)

		if bIsInsideAnchor(html[:origStart]) {
			continue
		}

		name := string(strippedBytes[loc[2]:loc[3]])
		section := string(strippedBytes[loc[4]:loc[5]])
		href := "/manpages/" + release + "/man" + section + "/" + name + "." + section + ".html"

		b.Write(html[lastEnd:origStart])
		b.WriteString(`<a href="`)
		b.WriteString(href)
		b.WriteString(`">`)
		b.Write(html[origStart:origEnd])
		b.WriteString(`</a>`)
		lastEnd = origEnd
	}
	b.Write(html[lastEnd:])
	return b.Bytes()
}

func bExpandLeft(html []byte, pos, limit int) int {
	for pos > limit {
		loc := trailingInlineOpen.FindIndex(html[limit:pos])
		if loc == nil {
			break
		}
		pos = limit + loc[0]
	}
	return pos
}

func bExpandRight(html []byte, pos int) int {
	for pos < len(html) {
		loc := leadingInlineClose.FindIndex(html[pos:])
		if loc == nil || loc[0] != 0 {
			break
		}
		pos += loc[1]
	}
	return pos
}

func bIsInsideAnchor(html []byte) bool {
	lastOpen := bytes.LastIndex(html, []byte("<a "))
	if i := bytes.LastIndex(html, []byte("<a>")); i > lastOpen {
		lastOpen = i
	}
	if lastOpen == -1 {
		return false
	}
	return lastOpen > bytes.LastIndex(html, []byte("</a>"))
}
