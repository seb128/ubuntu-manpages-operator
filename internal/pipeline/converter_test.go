package pipeline

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestNeedsTblPreprocessing(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"no tables", "plain manpage content\n.SH NAME\ntest", false},
		{"TS at start", ".TS\nallbox;\nc.\ndata\n.TE\n", true},
		{"TS mid content", ".SH TABLES\n.TS\nallbox;\nc.\ndata\n.TE\n", true},
		{"TS without newline boundary", "some.TS.thing", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsTblPreprocessing(tt.content)
			if got != tt.want {
				t.Errorf("needsTblPreprocessing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertManpageTimeout(t *testing.T) {
	// Create a script that sleeps forever to simulate a hanging mandoc.
	// Use exec so the shell replaces itself; otherwise the child sleep
	// process survives the context-triggered kill.
	script := "#!/bin/sh\nexec sleep 60\n"
	tmp, err := os.CreateTemp(t.TempDir(), "fake-mandoc-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(script); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a simple manpage to convert.
	mp := t.TempDir() + "/test.1"
	if err := os.WriteFile(mp, []byte(".TH TEST 1\n.SH NAME\ntest\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewConverter(tmp.Name())
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err = c.ConvertManpage(ctx, mp)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestConvertManpageWithTbl(t *testing.T) {
	if _, err := exec.LookPath("tbl"); err != nil {
		t.Skip("tbl not available")
	}
	if _, err := exec.LookPath("mandoc"); err != nil {
		t.Skip("mandoc not available")
	}

	// A manpage with a simple .TS/.TE table block.
	content := `.TH TEST 1
.SH NAME
test \- a test page
.SH TABLE
.TS
allbox;
c c.
A	B
1	2
.TE
`
	mp := t.TempDir() + "/test.1"
	if err := os.WriteFile(mp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewConverter("")
	html, err := c.ConvertManpage(context.Background(), mp)
	if err != nil {
		t.Fatalf("ConvertManpage with tbl failed: %v", err)
	}
	if html == "" {
		t.Fatal("expected non-empty HTML output")
	}
	// The table content should appear in the output.
	if !containsStr(html, "A") || !containsStr(html, "B") {
		t.Errorf("expected table data in output, got: %s", html)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestConvertBulletLists(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "bullet list converts to ul",
			in: `<dl class="Bl-tag">
  <dt>&#x2022;</dt>
  <dd>First item</dd>
  <dt>&#x2022;</dt>
  <dd>Second item</dd>
</dl>`,
			want: `<ul>
  <li>First item</li>
  <li>Second item</li>
</ul>`,
		},
		{
			name: "non-bullet dl is preserved",
			in: `<dl class="Bl-tag">
  <dt><code class="Fl">-v</code></dt>
  <dd>Verbose mode</dd>
</dl>`,
			want: `<dl class="Bl-tag">
  <dt><code class="Fl">-v</code></dt>
  <dd>Verbose mode</dd>
</dl>`,
		},
		{
			name: "mixed dt content is preserved",
			in: `<dl class="Bl-tag">
  <dt>&#x2022;</dt>
  <dd>Bullet</dd>
  <dt>tag</dt>
  <dd>Not bullet</dd>
</dl>`,
			want: `<dl class="Bl-tag">
  <dt>&#x2022;</dt>
  <dd>Bullet</dd>
  <dt>tag</dt>
  <dd>Not bullet</dd>
</dl>`,
		},
		{
			name: "surrounding content preserved",
			in: `<p>Before</p>
<dl class="Bl-tag">
  <dt>&#x2022;</dt>
  <dd>Item</dd>
</dl>
<p>After</p>`,
			want: `<p>Before</p>
<ul>
  <li>Item</li>
</ul>
<p>After</p>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertBulletLists(tt.in)
			if got != tt.want {
				t.Errorf("convertBulletLists():\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}
