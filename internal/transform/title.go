package transform

import (
	htmlutil "html"
	"regexp"
	"strings"
)

var (
	// Matches an h1 tag and its content. Used to find NAME sections by
	// stripping tags and comparing the inner text.
	manpageH1Pattern    = regexp.MustCompile(`(?is)(<h1[^>]*>)(.*?)(</h1>)`)
	manpageTitlePattern = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	manpageStripTags    = regexp.MustCompile(`(?is)<[^>]+>`)
)

// nameKeywords are the heading texts (uppercased) that identify a NAME
// section, including common translations.
var nameKeywords = map[string]bool{
	"NAME": true, "BEZEICHNUNG": true, "NOMBRE": true,
	"NOM": true, "NOME": true, "NAAM": true,
	"NAZWA": true, "NAZWA:": true,
}

func isNameKeyword(text string) bool {
	return nameKeywords[strings.ToUpper(text)]
}

// sectionHeadings are common section heading texts that should never be
// used as a page title.
var sectionHeadings = map[string]bool{
	"SYNOPSIS": true, "DESCRIPTION": true, "OPTIONS": true,
	"SEE ALSO": true, "AUTHOR": true, "AUTHORS": true, "BUGS": true,
	"EXAMPLES": true, "EXIT STATUS": true, "RETURN VALUE": true,
	"ENVIRONMENT": true, "FILES": true, "NOTES": true, "HISTORY": true,
	"STANDARDS": true, "CONFORMING TO": true,
}

func isSectionHeading(text string) bool {
	return sectionHeadings[strings.ToUpper(text)] || isNameKeyword(text)
}

// collapseWhitespace replaces runs of whitespace (including newlines)
// with a single space.
func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func extractManpageTitle(html string) string {
	// Walk all h1 headings looking for one whose text matches a NAME keyword.
	matches := manpageH1Pattern.FindAllStringSubmatchIndex(html, -1)
	for i, loc := range matches {
		inner := html[loc[4]:loc[5]]
		text := strings.TrimSpace(manpageStripTags.ReplaceAllString(inner, ""))
		if !isNameKeyword(text) {
			continue
		}
		afterH1 := loc[1]
		end := len(html)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		content := html[afterH1:end]
		content = manpageStripTags.ReplaceAllString(content, "")
		content = htmlutil.UnescapeString(content)
		var lines []string
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				if len(lines) > 0 {
					break
				}
				continue
			}
			lines = append(lines, line)
		}
		if len(lines) > 0 {
			if hasManpageSeparator(lines[0]) {
				return collapseWhitespace(strings.Join(lines, " "))
			}
			return collapseWhitespace(lines[0])
		}
	}
	// Fallback: use the first h1 whose text isn't a section heading.
	allMatches := manpageTitlePattern.FindAllStringSubmatch(html, -1)
	for _, m := range allMatches {
		text := manpageStripTags.ReplaceAllString(m[1], "")
		text = htmlutil.UnescapeString(text)
		text = collapseWhitespace(text)
		if text == "" || isSectionHeading(text) {
			continue
		}
		return text
	}
	return "Ubuntu Manpage"
}

// ExtractManpageTitle returns the title extracted from raw mandoc HTML.
func ExtractManpageTitle(html string) string {
	return extractManpageTitle(html)
}

var titleSeparators = []string{" -- ", " - ", " \u2013 ", " \u2014 "}
var trailingSeparators = []string{" --", " -", " \u2013", " \u2014"}

func hasManpageSeparator(s string) bool {
	for _, sep := range titleSeparators {
		if strings.Contains(s, sep) {
			return true
		}
	}
	for _, suffix := range trailingSeparators {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// SplitManpageTitle splits "name - description" into its two parts.
// If there is no separator the full string is returned as the name.
func SplitManpageTitle(title string) (name, description string) {
	for _, sep := range titleSeparators {
		if i := strings.Index(title, sep); i >= 0 {
			return strings.TrimSpace(title[:i]), strings.TrimSpace(title[i+len(sep):])
		}
	}
	for _, suffix := range trailingSeparators {
		if strings.HasSuffix(title, suffix) {
			return strings.TrimSuffix(title, suffix), ""
		}
	}
	return title, ""
}

// MaxDescriptionLen is the maximum length of a description before truncation.
const MaxDescriptionLen = 200

func capDescription(desc string) string {
	if len(desc) <= MaxDescriptionLen {
		return desc
	}
	cut := strings.LastIndex(desc[:MaxDescriptionLen], " ")
	if cut <= 0 {
		cut = MaxDescriptionLen
	}
	return strings.TrimRight(desc[:cut], ".,;: ") + " …"
}

func titleFromFilename(filename string) string {
	name := filename
	for _, ext := range []string{".gz", ".bz2", ".xz", ".zst"} {
		name = strings.TrimSuffix(name, ext)
	}
	if dot := strings.LastIndex(name, "."); dot > 0 {
		suffix := name[dot+1:]
		if len(suffix) > 0 && suffix[0] >= '1' && suffix[0] <= '9' {
			name = name[:dot]
		}
	}
	if name == "" {
		return "Ubuntu Manpage"
	}
	return name
}

// StripHTMLTags removes all HTML tags from the input.
func StripHTMLTags(html string) string {
	return strings.TrimSpace(manpageStripTags.ReplaceAllString(html, " "))
}
