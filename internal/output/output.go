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
	"unicode/utf8"

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
	fmt.Fprintf(w, "\n🔎 搜索结果（共 %d 项）\n\n", len(results))
	columns := []tableColumn{
		{title: "项目名称", width: 24},
		{title: "关键字", width: 34},
		{title: "文件路径:行号", width: 22},
		{title: "代码片段", width: 78},
		{title: "分支", width: 10},
	}
	writeBorder(w, columns, "┏", "┳", "┓", "━")
	writeTableRow(w, columns, [][]string{{"项目名称"}, {"关键字"}, {"文件路径:行号"}, {"代码片段"}, {"分支"}})
	writeBorder(w, columns, "┣", "╋", "┫", "━")
	for i, r := range results {
		writeTableRow(w, columns, [][]string{
			{r.ProjectName},
			{r.Keyword},
			{fmt.Sprintf("%s:%d", r.FilePath, r.LineNumber)},
			previewLines(r.LineContent, 8, columns[3].width),
			{r.Branch},
		})
		if i+1 < len(results) {
			writeBorder(w, columns, "┣", "╋", "┫", "─")
		}
	}
	writeBorder(w, columns, "┗", "┻", "┛", "━")
	return nil
}

type tableColumn struct {
	title string
	width int
}

func writeBorder(w io.Writer, columns []tableColumn, left, middle, right, fill string) {
	fmt.Fprint(w, left)
	for i, column := range columns {
		fmt.Fprint(w, strings.Repeat(fill, column.width+2))
		if i+1 == len(columns) {
			fmt.Fprintln(w, right)
		} else {
			fmt.Fprint(w, middle)
		}
	}
}

func writeTableRow(w io.Writer, columns []tableColumn, cells [][]string) {
	height := 1
	for _, cell := range cells {
		if len(cell) > height {
			height = len(cell)
		}
	}
	for line := 0; line < height; line++ {
		fmt.Fprint(w, "┃")
		for i, column := range columns {
			value := ""
			if line < len(cells[i]) {
				value = truncateCell(cells[i][line], column.width)
			}
			fmt.Fprintf(w, " %s%s ┃", value, strings.Repeat(" ", column.width-displayWidth(value)))
		}
		fmt.Fprintln(w)
	}
}

func truncateCell(value string, width int) string {
	if displayWidth(value) <= width {
		return value
	}
	var b strings.Builder
	used := 0
	for _, r := range value {
		runeWidth := displayWidth(string(r))
		if used+runeWidth > width-1 {
			break
		}
		b.WriteRune(r)
		used += runeWidth
	}
	return b.String() + "…"
}

// displayWidth is sufficient for terminal tables: most CJK characters occupy
// two columns and ordinary runes occupy one.
func displayWidth(value string) int {
	width := 0
	for _, r := range value {
		switch {
		case r == '\t':
			width += 4
		case r < utf8.RuneSelf:
			width++
		default:
			width += 2
		}
	}
	return width
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
