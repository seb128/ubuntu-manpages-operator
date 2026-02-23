package transform

import (
	"bytes"
	"regexp"
	"strings"
)

var (
	nameSectionPattern  = regexp.MustCompile(`(?is)<section[^>]*>\s*<h1[^>]*>.*?</h1>.*?</section>`)
	emptySectionTag     = regexp.MustCompile(`</h[12]>\s*</section>\s*`)
	leadingBreakPattern = regexp.MustCompile(`^(\s*<br\s*/?\s*>\s*)+`)

	h2TagPattern = regexp.MustCompile(`(?i)<(/?)h2([\s>])`)
	h1TagPattern = regexp.MustCompile(`(?i)<(/?)h1([\s>])`)

	mandocSectionH2 = regexp.MustCompile(
		`(?s)(<section[^>]*>)\s*(<h2[^>]*>.*?</h2>)\s*(.*?)(</section>)`,
	)
	mandocSectionH3 = regexp.MustCompile(
		`(?s)(<section[^>]*>)\s*(<h3[^>]*>.*?</h3>)\s*(.*?)(</section>)`,
	)
	sectionClassAttr = regexp.MustCompile(`class="([^"]*)"`)
	h2OpenTag        = regexp.MustCompile(`(?s)<h2[^>]*>.*?</h2>`)

	sectionOpenTag = []byte("<section")
)

func bRemoveFirstHeading(html []byte) []byte {
	if loc := nameSectionPattern.FindIndex(html); loc != nil {
		result := make([]byte, 0, len(html)-(loc[1]-loc[0]))
		result = append(result, html[:loc[0]]...)
		result = append(result, html[loc[1]:]...)
		return result
	}
	matches := manpageH1Pattern.FindAllSubmatchIndex(html, 2)
	if len(matches) == 0 {
		return html
	}
	loc := matches[0]
	inner := html[loc[4]:loc[5]]
	text := strings.TrimSpace(manpageStripTags.ReplaceAllString(string(inner), ""))
	if isNameKeyword(text) {
		end := len(html)
		if len(matches) > 1 {
			end = matches[1][0]
		}
		result := make([]byte, 0, loc[0]+len(html)-end)
		result = append(result, html[:loc[0]]...)
		result = append(result, html[end:]...)
		return result
	}
	result := make([]byte, 0, loc[0]+len(html)-loc[1])
	result = append(result, html[:loc[0]]...)
	result = append(result, html[loc[1]:]...)
	return result
}

func bStripLeadingBreaks(html []byte) []byte {
	return leadingBreakPattern.ReplaceAll(html, nil)
}

func bRemoveEmptySections(html []byte) []byte {
	for {
		loc := emptySectionTag.FindIndex(html)
		if loc == nil {
			break
		}
		start := bytes.LastIndex(html[:loc[0]], sectionOpenTag)
		if start < 0 {
			html = append(html[:loc[0]:loc[0]], html[loc[1]:]...)
			continue
		}
		html = append(html[:start:start], html[loc[1]:]...)
	}
	return html
}

func bShiftHeadings(html []byte) []byte {
	html = h2TagPattern.ReplaceAll(html, []byte("<${1}h3${2}"))
	html = h1TagPattern.ReplaceAll(html, []byte("<${1}h2${2}"))
	return html
}

func bWrapSections(html []byte) []byte {
	if mandocSectionH2.Match(html) {
		return mandocSectionH2.ReplaceAllFunc(html, func(match []byte) []byte {
			parts := mandocSectionH2.FindSubmatch(match)
			openTag, h2, body, closeTag := parts[1], parts[2], parts[3], parts[4]
			if sectionClassAttr.Match(openTag) {
				openTag = sectionClassAttr.ReplaceAll(openTag, []byte(`class="$1 mp-section"`))
			} else {
				openTag = bytes.Replace(openTag, []byte(">"), []byte(` class="mp-section">`), 1)
			}
			var b bytes.Buffer
			b.Grow(len(h2) + 1 + len(openTag) + len(body) + len(closeTag))
			b.Write(h2)
			b.WriteByte('\n')
			b.Write(openTag)
			b.Write(body)
			b.Write(closeTag)
			return b.Bytes()
		})
	}

	locs := h2OpenTag.FindAllIndex(html, -1)
	if len(locs) == 0 {
		if mandocSectionH3.Match(html) {
			return bWrapSectionsH3(html)
		}
		return html
	}

	var b bytes.Buffer
	b.Grow(len(html) + len(locs)*len(`<div class="mp-section"></div>`))
	b.Write(html[:locs[0][0]])

	for i, loc := range locs {
		h2 := html[loc[0]:loc[1]]
		contentStart := loc[1]
		contentEnd := len(html)
		if i+1 < len(locs) {
			contentEnd = locs[i+1][0]
		}
		content := html[contentStart:contentEnd]

		b.Write(h2)
		b.WriteString("\n<div class=\"mp-section\">")
		b.Write(content)
		b.WriteString("</div>")
	}
	return b.Bytes()
}

func bWrapSectionsH3(html []byte) []byte {
	return mandocSectionH3.ReplaceAllFunc(html, func(match []byte) []byte {
		parts := mandocSectionH3.FindSubmatch(match)
		openTag, h3, body, closeTag := parts[1], parts[2], parts[3], parts[4]
		if sectionClassAttr.Match(openTag) {
			openTag = sectionClassAttr.ReplaceAll(openTag, []byte(`class="$1 mp-section"`))
		} else {
			openTag = bytes.Replace(openTag, []byte(">"), []byte(` class="mp-section">`), 1)
		}
		var b bytes.Buffer
		b.Grow(len(h3) + 1 + len(openTag) + len(body) + len(closeTag))
		b.Write(h3)
		b.WriteByte('\n')
		b.Write(openTag)
		b.Write(body)
		b.Write(closeTag)
		return b.Bytes()
	})
}

// wrapSections is the string wrapper used by PrepareFragment and tests.
func wrapSections(html string) string {
	return string(bWrapSections([]byte(html)))
}
