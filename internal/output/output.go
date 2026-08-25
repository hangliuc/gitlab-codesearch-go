package output

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gls/internal/model"
)

var header = []string{"关键字", "项目ID", "项目名", "分支", "项目地址", "文件路径", "行号", "代码片段", "代码地址"}

func Write(w io.Writer, results []model.SearchResult, filename string) error {
	if len(results) == 0 {
		_, err := fmt.Fprintln(w, "未找到任何匹配结果")
		return err
	}
	if filename == "" {
		return printTable(w, results)
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return writeCSV(filename, results)
	case ".json":
		return writeJSON(filename, results)
	case ".xlsx":
		return writeXLSX(filename, results)
	default:
		return fmt.Errorf("不支持的输出格式 %q（仅支持 .xlsx、.csv、.json）", ext)
	}
}
func rows(results []model.SearchResult) [][]string {
	data := make([][]string, 0, len(results)+1)
	data = append(data, header)
	for _, r := range results {
		data = append(data, []string{r.Keyword, strconv.Itoa(r.ProjectID), r.ProjectName, r.Branch, r.ProjectPath, r.FilePath, strconv.Itoa(r.LineNumber), r.LineContent, r.URL})
	}
	return data
}
func writeCSV(name string, r []model.SearchResult) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.UseCRLF = true
	if err := w.WriteAll(rows(r)); err != nil {
		return err
	}
	return w.Error()
}
func writeJSON(name string, r []model.SearchResult) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}
func printTable(w io.Writer, results []model.SearchResult) error {
	fmt.Fprintf(w, "\n🔎 搜索结果  共 %d 项\n\n", len(results))
	for index, r := range results {
		location := fmt.Sprintf("%s:%d", r.FilePath, r.LineNumber)
		fmt.Fprintf(w, "┌─ %d/%d  %s\n", index+1, len(results), r.ProjectName)
		fmt.Fprintf(w, "│  关键字  %s\n", r.Keyword)
		fmt.Fprintf(w, "│  位置    %s  ·  %s\n", location, r.Branch)
		fmt.Fprintln(w, "│  代码")
		for _, line := range previewLines(r.LineContent, 8, 180) {
			fmt.Fprintf(w, "│    %s\n", line)
		}
		fmt.Fprintf(w, "│  链接    %s\n", r.URL)
		fmt.Fprintln(w, "└────────────────────────────────────────────────────────────────")
		if index+1 < len(results) {
			fmt.Fprintln(w)
		}
	}
	return nil
}

// previewLines retains code readability in the terminal without allowing a
// large GitLab search snippet to visually consume all subsequent results.
func previewLines(content string, maxLines, maxRunes int) []string {
	lines := strings.Split(strings.Trim(content, "\r\n"), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return []string{"(空片段)"}
	}
	limited := len(lines) > maxLines
	if limited {
		lines = lines[:maxLines]
	}
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		runes := []rune(line)
		if len(runes) > maxRunes {
			line = string(runes[:maxRunes]) + "…"
		}
		lines[i] = line
	}
	if limited {
		lines = append(lines, "…（片段已截断）")
	}
	return lines
}

// writeXLSX writes a standards-compliant, dependency-free XLSX workbook.
func writeXLSX(name string, results []model.SearchResult) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	defer z.Close()
	files := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="搜索结果" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   sheetXML(rows(results)),
	}
	for path, data := range files {
		entry, err := z.Create(path)
		if err != nil {
			return err
		}
		if _, err = io.WriteString(entry, data); err != nil {
			return err
		}
	}
	return nil
}
func sheetXML(data [][]string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for ri, row := range data {
		b.WriteString(`<row r="`)
		b.WriteString(strconv.Itoa(ri + 1))
		b.WriteString(`">`)
		for ci, value := range row {
			cell := string(rune('A'+ci)) + strconv.Itoa(ri+1)
			b.WriteString(`<c r="` + cell + `" t="inlineStr"><is><t xml:space="preserve">` + xmlEscape(value) + `</t></is></c>`)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}
func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;").Replace(strings.ToValidUTF8(s, "�"))
}

func SuccessMessage(filename string) string {
	return fmt.Sprintf("结果已成功导出至: %s (%s)", filename, time.Now().Format(time.RFC3339))
}
