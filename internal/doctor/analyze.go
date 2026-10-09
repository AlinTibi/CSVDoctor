package doctor

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func Analyze(data []byte, name string, opts ParseOptions) Document {
	sum := sha256.Sum256(data)
	doc := Document{Name: filepath.Base(name), Bytes: len(data), SHA256: fmt.Sprintf("%x", sum), Header: opts.Header, QuoteMode: opts.QuoteMode, Counts: map[string]int{}, Issues: []Issue{}, Preview: []Row{}, Rows: []Row{}, Candidates: []Candidate{}}
	if doc.QuoteMode == "" {
		doc.QuoteMode = "strict"
	}
	text, enc, certain, err := Decode(data, opts.Encoding)
	doc.Encoding, doc.EncodingConfirmed = enc, certain
	if err != nil {
		doc.add("encoding-error", "error", err.Error(), 0, 0)
		return doc
	}
	doc.LineEnding = lineStyle(text)
	doc.RawPreview = truncate(text, 4000)
	if !certain {
		doc.add("encoding-choice", "error", "Encoding is a suggestion, not proof. Select the source encoding explicitly and check the decoded text before export.", 0, 0)
	}
	if opts.Delimiter == "" || opts.Delimiter == "auto" {
		doc.Delimiter, doc.DelimiterConfirmed, doc.Candidates = DetectDelimiter(text)
	} else {
		doc.Delimiter, doc.DelimiterConfirmed = opts.Delimiter, delimiterOK(opts.Delimiter)
	}
	if !doc.DelimiterConfirmed {
		doc.add("delimiter-choice", "error", "No unique delimiter detected. Select the delimiter explicitly before exporting.", 0, 0)
	}
	rows, err := Parse(text, doc.Delimiter, doc.QuoteMode)
	doc.Rows = rows
	doc.RowCount = len(rows)
	if err != nil {
		doc.add("parse-error", "error", err.Error(), 0, 0)
	} else {
		doc.Parsed = true
	}
	if len(rows) == 0 {
		doc.add("empty-file", "error", "No records found. Nothing will be exported.", 0, 0)
	}
	if doc.QuoteMode == "literal" {
		doc.add("literal-quotes", "warning", "Explicit interpretation: quotes are literal characters; embedded line breaks are separate records. Check the source and table before exporting.", 0, 0)
	}
	if doc.LineEnding == "Mixed" || doc.LineEnding == "CR" {
		doc.add("line-endings", "info", "Record separators can be normalized on export. Line breaks inside cells are preserved.", 0, 0)
	}
	doc.scan(rows)
	doc.Preview = preview(rows)
	return doc
}
func (d *Document) add(code, severity, message string, row, column int) {
	d.Counts[code]++
	if len(d.Issues) < MaxIssues {
		d.Issues = append(d.Issues, Issue{code, severity, message, row, column})
	}
}
func (d *Document) scan(rows []Row) {
	if len(rows) == 0 {
		return
	}
	d.Columns = len(rows[0].Cells)
	if !d.Header {
		histogram := map[int]int{}
		best := 0
		for _, r := range rows {
			if !r.Blank {
				histogram[len(r.Cells)]++
			}
		}
		for width, count := range histogram {
			if count > best || (count == best && width > d.Columns) {
				d.Columns, best = width, count
			}
		}
	}
	for r, row := range rows {
		if len(row.Cells) > d.MaxColumns {
			d.MaxColumns = len(row.Cells)
		}
		blank := true
		for _, cell := range row.Cells {
			if cell != "" {
				blank = false
				break
			}
		}
		if blank {
			d.add("blank-row", "info", "Blank record is preserved; it is never deleted automatically.", r+1, 0)
		}
		if len(row.Cells) != d.Columns {
			d.add("uneven-row", "warning", fmt.Sprintf("Expected %d fields, found %d. No extra fields will be removed.", d.Columns, len(row.Cells)), r+1, 0)
		}
		for c, cell := range row.Cells {
			if FormulaRisk(cell) {
				d.add("formula-risk", "warning", "Spreadsheet software may interpret this cell as a formula. Protection is optional and changes cell text.", r+1, c+1)
			}
			if IdentifierRisk(cell) {
				d.add("identifier-risk", "warning", "Excel may reinterpret this identifier. Import its column as Text; CSV quoting or a BOM does not prevent type conversion.", r+1, c+1)
			}
		}
	}
	if d.Header {
		seen := map[string]bool{}
		for c, cell := range rows[0].Cells {
			key := strings.ToLower(strings.TrimSpace(cell))
			if key == "" {
				d.add("empty-header", "warning", "Header is empty. Giving it a name is an explicit content change.", 1, c+1)
			} else if seen[key] {
				d.add("duplicate-header", "warning", "Header repeats another name (ignoring case and surrounding spaces). Renaming is optional.", 1, c+1)
			}
			seen[key] = true
		}
	}
}
func FormulaRisk(cell string) bool {
	if cell == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(cell)
	if first == '\t' || first == '\r' || first == '\n' {
		return true
	}
	trimmed := strings.TrimLeft(cell, " \t\r\n")
	if trimmed == "" {
		return false
	}
	first, _ = utf8.DecodeRuneInString(trimmed)
	return strings.ContainsRune("=+-@＝＋－＠", first)
}
func IdentifierRisk(cell string) bool {
	s := strings.TrimSpace(cell)
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 15 || (len(s) > 1 && s[0] == '0')
}
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}
func preview(rows []Row) []Row {
	n := len(rows)
	if n > PreviewRows {
		n = PreviewRows
	}
	out := make([]Row, n)
	for i := 0; i < n; i++ {
		out[i] = Row{Line: rows[i].Line, Blank: rows[i].Blank, Cells: make([]string, len(rows[i].Cells))}
		for c, v := range rows[i].Cells {
			out[i].Cells[c] = truncate(v, 2000)
		}
	}
	return out
}
