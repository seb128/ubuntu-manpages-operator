package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Converter struct {
	Binary string
}

func NewConverter(binary string) *Converter {
	if binary == "" {
		binary = "mandoc"
	}
	return &Converter{Binary: binary}
}

var (
	mandocHeadTable = regexp.MustCompile(`(?s)<table class="head">.*?</table>\s*`)
	mandocFootTable = regexp.MustCompile(`(?s)<table class="foot">.*?</table>\s*`)
	mandocManualDiv = regexp.MustCompile(`(?s)^<div class="manual-text">\s*`)
	mandocManualEnd = regexp.MustCompile(`(?s)\s*</div>\s*$`)
	// mandocPreBlock matches <pre>...</pre> elements.
	mandocPreBlock = regexp.MustCompile(`(?s)<pre>(.*?)</pre>`)
	// mandocBreakTag matches a <br/> tag on its own line inside a <pre> block.
	mandocBreakTag = regexp.MustCompile(`\n<br/>\n`)
)

func (c *Converter) ConvertManpage(ctx context.Context, inputPath string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	content, err := readManpageContent(inputPath)
	if err != nil {
		return "", err
	}

	// Always try mandoc first — its built-in tbl handling produces
	// better HTML than the external tbl(1) preprocessor.  If mandoc
	// hangs (some complex tables cause this), fall back to tbl piping.
	var raw string
	if needsTblPreprocessing(content) {
		tblCtx, tblCancel := context.WithTimeout(ctx, 10*time.Second)
		raw, err = c.runMandoc(tblCtx, content)
		tblCancel()
		if err != nil {
			raw, err = c.runWithTbl(ctx, content)
		}
	} else {
		raw, err = c.runMandoc(ctx, content)
	}
	if err != nil {
		return "", err
	}

	html := raw
	html = mandocHeadTable.ReplaceAllString(html, "")
	html = mandocFootTable.ReplaceAllString(html, "")
	html = mandocManualDiv.ReplaceAllString(html, "")
	html = mandocManualEnd.ReplaceAllString(html, "")
	html = stripBreaksInPre(html)
	html = convertBulletLists(html)
	return strings.TrimSpace(html), nil
}

// needsTblPreprocessing reports whether a manpage source contains tbl
// table directives (.TS/.TE) that should be preprocessed with tbl(1)
// before mandoc. Some complex tables cause mandoc to hang indefinitely.
func needsTblPreprocessing(content string) bool {
	return strings.Contains(content, "\n.TS\n") || strings.HasPrefix(content, ".TS\n")
}

func (c *Converter) runMandoc(ctx context.Context, content string) (string, error) {
	cmd := exec.CommandContext(ctx, c.Binary, "-T", "html", "-O", "fragment")
	cmd.Stdin = strings.NewReader(content)
	cmd.WaitDelay = 5 * time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mandoc failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func (c *Converter) runWithTbl(ctx context.Context, content string) (string, error) {
	tbl := exec.CommandContext(ctx, "tbl")
	tbl.Stdin = strings.NewReader(content)
	tbl.WaitDelay = 5 * time.Second

	mandoc := exec.CommandContext(ctx, c.Binary, "-T", "html", "-O", "fragment")
	mandoc.WaitDelay = 5 * time.Second
	var stdout, stderr bytes.Buffer
	mandoc.Stdout = &stdout
	mandoc.Stderr = &stderr

	pipe, err := tbl.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("tbl stdout pipe: %w", err)
	}
	mandoc.Stdin = pipe

	if err := tbl.Start(); err != nil {
		return "", fmt.Errorf("start tbl: %w", err)
	}
	if err := mandoc.Start(); err != nil {
		_ = tbl.Process.Kill()
		return "", fmt.Errorf("start mandoc: %w", err)
	}

	if err := tbl.Wait(); err != nil {
		_ = mandoc.Process.Kill()
		return "", fmt.Errorf("tbl failed: %w", err)
	}
	if err := mandoc.Wait(); err != nil {
		return "", fmt.Errorf("mandoc failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}

// stripBreaksInPre removes <br/> tags inside <pre> blocks. Mandoc inserts
// these where blank lines existed in the source, but inside <pre> the
// newlines are already preserved, so the <br/> causes double-spacing.
func stripBreaksInPre(html string) string {
	return mandocPreBlock.ReplaceAllStringFunc(html, func(match string) string {
		inner := mandocPreBlock.FindStringSubmatch(match)[1]
		return "<pre>" + mandocBreakTag.ReplaceAllString(inner, "\n") + "</pre>"
	})
}

// bulletDTDD matches a bullet <dt> followed by the opening <dd> tag.
var bulletDTDD = regexp.MustCompile(`<dt>\s*&#x2022;\s*</dt>\s*<dd>`)

// convertBulletLists converts mandoc bullet-style <dl class="Bl-tag"> lists
// to semantic <ul>/<li> elements. Mandoc renders .Bl-bullet lists as
// definition lists with &#x2022; (bullet) as the <dt>, but these are
// semantically unordered lists and render better with <ul> styling.
func convertBulletLists(html string) string {
	const open = `<dl class="Bl-tag">`
	const close = `</dl>`

	var b strings.Builder
	for {
		idx := strings.Index(html, open)
		if idx < 0 {
			b.WriteString(html)
			break
		}
		b.WriteString(html[:idx])
		after := html[idx+len(open):]

		// Find the matching </dl>, counting nesting.
		depth := 1
		i := 0
		for i < len(after) && depth > 0 {
			if strings.HasPrefix(after[i:], "<dl") {
				depth++
				i += 3
			} else if strings.HasPrefix(after[i:], close) {
				depth--
				if depth == 0 {
					break
				}
				i += len(close)
			} else {
				i++
			}
		}
		if depth != 0 {
			b.WriteString(open)
			html = after
			continue
		}

		inner := after[:i]
		html = after[i+len(close):]

		// Convert only simple (non-nested) bullet lists.
		if strings.Contains(inner, "<dl") || !isBulletDL(inner) {
			b.WriteString(open)
			b.WriteString(inner)
			b.WriteString(close)
			continue
		}

		converted := bulletDTDD.ReplaceAllString(inner, "<li>")
		converted = strings.ReplaceAll(converted, "</dd>", "</li>")
		b.WriteString("<ul>")
		b.WriteString(converted)
		b.WriteString("</ul>")
	}
	return b.String()
}

// isBulletDL reports whether the inner content of a <dl> contains only
// bullet-character <dt> elements (&#x2022;).
func isBulletDL(inner string) bool {
	const dtOpen = "<dt>"
	const dtClose = "</dt>"
	pos := 0
	found := false
	for {
		start := strings.Index(inner[pos:], dtOpen)
		if start < 0 {
			break
		}
		start += pos
		end := strings.Index(inner[start:], dtClose)
		if end < 0 {
			break
		}
		end += start
		content := strings.TrimSpace(inner[start+len(dtOpen) : end])
		if content != "&#x2022;" {
			return false
		}
		found = true
		pos = end + len(dtClose)
	}
	return found
}

func readManpageContent(path string) (string, error) {
	reader, cleanup, err := openMaybeGzipped(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = cleanup() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read manpage: %w", err)
	}

	return string(bytes.TrimSpace(data)), nil
}

func ManpageNameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".gz")
}
