package transform

import (
	"encoding/json"
	"strings"
)

// FragmentMeta is the metadata prepended to a manpage HTML fragment.
type FragmentMeta struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Package     string `json:"package,omitempty"`
	PackageURL  string `json:"packageURL,omitempty"`
	Source      string `json:"source,omitempty"`
	SourceURL   string `json:"sourceURL,omitempty"`
	BugURL      string `json:"bugURL,omitempty"`
	TOC         string `json:"toc,omitempty"`
}

// PrepareFragment transforms converter HTML into a fragment with a JSON
// metadata header. This is the legacy entry point; prefer Pipeline for
// new code.
func PrepareFragment(release string, html string, meta *ManpageMeta) (string, error) {
	fullTitle := extractManpageTitle(html)
	if fullTitle == "Ubuntu Manpage" && meta != nil && meta.Filename != "" {
		fullTitle = titleFromFilename(meta.Filename)
	}
	title, description := SplitManpageTitle(fullTitle)
	description = capDescription(description)
	body := string(bRemoveFirstHeading([]byte(html)))
	body = string(bStripLeadingBreaks([]byte(body)))
	body = string(bRemoveEmptySections([]byte(body)))
	body = string(bShiftHeadings([]byte(body)))
	body = wrapSections(body)
	var tocHTML string
	body, tocHTML = generateTOC(body)

	fm := FragmentMeta{
		Title:       title,
		Description: description,
		Package:     buildPackageLabel(meta),
		PackageURL:  buildPackageURL(release, meta),
		Source:      buildSourceLabel(meta),
		SourceURL:   buildSourceURL(release, meta),
		BugURL:      buildBugURL(release, meta),
		TOC:         tocHTML,
	}

	metaJSON, err := json.Marshal(fm)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("<!--META:")
	b.Write(metaJSON)
	b.WriteString("-->\n")
	b.WriteString(body)
	return b.String(), nil
}

func buildPackageLabel(meta *ManpageMeta) string {
	if meta == nil || meta.PackageName == "" {
		return ""
	}
	if meta.PackageVersion == "" {
		return meta.PackageName
	}
	return meta.PackageName + " (" + meta.PackageVersion + ")"
}

func buildBugURL(release string, meta *ManpageMeta) string {
	if meta == nil || meta.SourcePackage == "" || release == "" {
		return ""
	}
	return "https://bugs.launchpad.net/ubuntu/+source/" + meta.SourcePackage + "/+filebug-advanced"
}

func buildSourceLabel(meta *ManpageMeta) string {
	if meta == nil || meta.SourcePackage == "" || meta.SourcePackage == meta.PackageName {
		return ""
	}
	return meta.SourcePackage
}

func buildSourceURL(release string, meta *ManpageMeta) string {
	if meta == nil || meta.SourcePackage == "" || meta.SourcePackage == meta.PackageName || release == "" {
		return ""
	}
	return "https://launchpad.net/ubuntu/" + release + "/+source/" + meta.SourcePackage
}

func buildPackageURL(release string, meta *ManpageMeta) string {
	if meta == nil || meta.PackageName == "" || release == "" {
		return ""
	}
	return "https://launchpad.net/ubuntu/" + release + "/+package/" + meta.PackageName
}
