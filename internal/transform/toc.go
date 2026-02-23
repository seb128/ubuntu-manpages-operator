package transform

import (
	"bytes"
	"fmt"
	htmlutil "html"
	"regexp"
	"strings"
)

var (
	tocHeadingPattern    = regexp.MustCompile(`(?s)<(h2)(\s[^>]*)?>(.+?)</h2>`)
	headingIDAttr        = regexp.MustCompile(`\s*id="[^"]*"`)
	nonAlphanumeric      = regexp.MustCompile(`[^a-z0-9]+`)
	permalinkHrefPattern = regexp.MustCompile(`<a class="permalink" href="#[^"]*"`)
)

func slugify(text string) string {
	slug := strings.ToLower(text)
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ""
	}
	return slug
}

func bGenerateTOC(html []byte) ([]byte, string) {
	matches := tocHeadingPattern.FindAllSubmatchIndex(html, -1)
	if len(matches) == 0 {
		return html, ""
	}

	seen := map[string]bool{}
	type tocEntry struct {
		id   string
		text string
	}
	var entries []tocEntry

	var body bytes.Buffer
	body.Grow(len(html) + len(matches)*32)
	lastEnd := 0

	for i, loc := range matches {
		fullStart, fullEnd := loc[0], loc[1]
		tag := string(html[loc[2]:loc[3]])
		attrs := ""
		if loc[4] >= 0 {
			attrs = string(html[loc[4]:loc[5]])
		}
		inner := html[loc[6]:loc[7]]

		text := collapseWhitespace(manpageStripTags.ReplaceAllString(string(inner), ""))
		if text == "" {
			body.Write(html[lastEnd:fullEnd])
			lastEnd = fullEnd
			continue
		}

		slug := slugify(htmlutil.UnescapeString(text))
		if slug == "" {
			slug = fmt.Sprintf("heading-%d", i)
		}
		if seen[slug] {
			slug = fmt.Sprintf("%s-%d", slug, i)
		}
		seen[slug] = true

		entries = append(entries, tocEntry{id: slug, text: htmlutil.EscapeString(htmlutil.UnescapeString(text))})

		rewrittenInner := permalinkHrefPattern.ReplaceAll(inner, []byte(`<a class="permalink" href="#`+slug+`"`))

		cleanAttrs := headingIDAttr.ReplaceAllString(attrs, "")

		body.Write(html[lastEnd:fullStart])
		body.WriteByte('<')
		body.WriteString(tag)
		body.WriteString(` id="`)
		body.WriteString(slug)
		body.WriteByte('"')
		body.WriteString(cleanAttrs)
		body.WriteByte('>')
		body.Write(rewrittenInner)
		body.WriteString("</")
		body.WriteString(tag)
		body.WriteByte('>')
		lastEnd = fullEnd
	}
	body.Write(html[lastEnd:])

	var toc strings.Builder
	for _, e := range entries {
		toc.WriteString(`<li class="p-table-of-contents__item">`)
		toc.WriteString(`<a class="p-table-of-contents__link" href="#`)
		toc.WriteString(e.id)
		toc.WriteString(`">`)
		toc.WriteString(e.text)
		toc.WriteString(`</a></li>`)
		toc.WriteByte('\n')
	}
	return body.Bytes(), toc.String()
}

// generateTOC is the string wrapper used by PrepareFragment and tests.
func generateTOC(html string) (string, string) {
	body, toc := bGenerateTOC([]byte(html))
	return string(body), toc
}
