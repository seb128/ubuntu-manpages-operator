// Package transform implements the HTML transformation pipeline for
// converting raw mandoc output into web-ready manpage fragments.
//
// The pipeline runs as a sequence of named stages:
//  1. Rewrite cross-reference links
//  2. Extract title and remove NAME section
//  3. Strip leading <br> tags
//  4. Remove empty sections
//  5. Shift headings (h1→h2, h2→h3)
//  6. Wrap sections with mp-section class
//  7. Generate TOC with slug IDs
//  8. Prepend metadata JSON header
package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Doc holds the mutable state of an HTML fragment as it passes through
// the transformation pipeline. Using []byte for Body avoids redundant
// string copies between stages.
type Doc struct {
	Release string
	Meta    *ManpageMeta
	Body    []byte // mutable HTML body
	Title   string // extracted title (set by stage 2)
	Desc    string // extracted description (set by stage 2)
	TOC     string // generated TOC HTML (set by stage 7)
}

// Pipeline runs all HTML transformation stages on converter output and
// returns the result.
func Pipeline(release string, rawHTML string, meta *ManpageMeta) (Doc, error) {
	doc := Doc{
		Release: release,
		Meta:    meta,
		Body:    []byte(rawHTML),
	}

	// Stage 1: Rewrite cross-reference links.
	if err := stageRewriteLinks(&doc); err != nil {
		return doc, fmt.Errorf("rewrite links: %w", err)
	}

	// Stage 2: Extract title and remove NAME section (fused).
	stageExtractTitleAndRemoveName(&doc)

	// Stage 3: Strip leading <br> tags.
	doc.Body = bStripLeadingBreaks(doc.Body)

	// Stage 4: Remove empty sections.
	doc.Body = bRemoveEmptySections(doc.Body)

	// Stage 5: Shift headings (h1→h2, h2→h3).
	doc.Body = bShiftHeadings(doc.Body)

	// Stage 6: Wrap sections with mp-section class.
	doc.Body = bWrapSections(doc.Body)

	// Stage 7: Generate TOC.
	doc.Body, doc.TOC = bGenerateTOC(doc.Body)

	// Stage 8: Prepend metadata JSON.
	if err := stagePrependMeta(&doc); err != nil {
		return doc, fmt.Errorf("prepend meta: %w", err)
	}

	return doc, nil
}

// stageRewriteLinks applies cross-reference link rewriting to doc.Body.
func stageRewriteLinks(doc *Doc) error {
	result, err := bRewriteLinks(doc.Release, doc.Body)
	if err != nil {
		return err
	}
	doc.Body = result
	return nil
}

// stageExtractTitleAndRemoveName extracts the title and description from
// the NAME section, then removes the NAME heading/section. This fuses what
// were previously two separate passes (extractManpageTitle + removeFirstHeading)
// into a single logical step.
func stageExtractTitleAndRemoveName(doc *Doc) {
	html := string(doc.Body)
	fullTitle := extractManpageTitle(html)
	if fullTitle == "Ubuntu Manpage" && doc.Meta != nil && doc.Meta.Filename != "" {
		fullTitle = titleFromFilename(doc.Meta.Filename)
	}
	doc.Title, doc.Desc = SplitManpageTitle(fullTitle)
	doc.Desc = capDescription(doc.Desc)
	doc.Body = bRemoveFirstHeading(doc.Body)
}

// stagePrependMeta builds the FragmentMeta JSON and prepends it as a
// <!--META:...--> comment to doc.Body.
func stagePrependMeta(doc *Doc) error {
	fm := FragmentMeta{
		Title:       doc.Title,
		Description: doc.Desc,
		Package:     buildPackageLabel(doc.Meta),
		PackageURL:  buildPackageURL(doc.Release, doc.Meta),
		Source:      buildSourceLabel(doc.Meta),
		SourceURL:   buildSourceURL(doc.Release, doc.Meta),
		BugURL:      buildBugURL(doc.Release, doc.Meta),
		TOC:         doc.TOC,
	}

	metaJSON, err := json.Marshal(fm)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	b.Grow(len("<!--META:") + len(metaJSON) + len("-->\n") + len(doc.Body))
	b.WriteString("<!--META:")
	b.Write(metaJSON)
	b.WriteString("-->\n")
	b.Write(doc.Body)
	doc.Body = b.Bytes()
	return nil
}
