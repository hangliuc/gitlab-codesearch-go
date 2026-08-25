package output

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gls/internal/model"
)

func TestWriteXLSX(t *testing.T) {
	name := filepath.Join(t.TempDir(), "results.xlsx")
	result := []model.SearchResult{{Keyword: "a&b", ProjectID: 1, ProjectName: "demo", FilePath: "a.go", LineNumber: 2, LineContent: "a < b"}}
	if err := Write(&bytes.Buffer{}, result, name); err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(name)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if len(r.File) != 5 {
		t.Fatalf("unexpected XLSX file count: %d", len(r.File))
	}
	if _, err := os.Stat(name); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewLinesPreservesIndentAndLimitsOutput(t *testing.T) {
	got := previewLines("  one\n    two\nthree", 2, 80)
	if len(got) != 3 || got[0] != "  one" || got[1] != "    two" || got[2] != "…（片段已截断）" {
		t.Fatalf("unexpected preview: %#v", got)
	}
}

func TestTruncateCellUsesTerminalWidth(t *testing.T) {
	if got := truncateCell("abcdef", 3); got != "ab…" {
		t.Fatalf("unexpected truncation: %q", got)
	}
	if got := truncateCell("搜索结果", 5); got != "搜索…" {
		t.Fatalf("unexpected CJK truncation: %q", got)
	}
}
