package doctor

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

func ValidateExport(o ExportOptions) error {
	if o.Encoding != "utf8" && o.Encoding != "utf8bom" {
		return fmt.Errorf("Choose UTF-8 or UTF-8 BOM")
	}
	if !delimiterOK(o.Delimiter) {
		return fmt.Errorf("Unsupported output delimiter")
	}
	if o.LineEnding != "\r\n" && o.LineEnding != "\n" {
		return fmt.Errorf("Choose CRLF or LF")
	}
	return nil
}
func Review(doc Document, repairs Repairs, output ExportOptions) Plan {
	plan := Plan{Blockers: []string{}, Changes: []Change{}, Before: preview(doc.Rows), After: []Row{}}
	if !doc.Parsed || len(doc.Rows) == 0 {
		plan.Blockers = append(plan.Blockers, "Parsing must succeed with at least one record before export.")
	}
	if !doc.EncodingConfirmed {
		plan.Blockers = append(plan.Blockers, "Confirm the source encoding explicitly.")
	}
	if !doc.DelimiterConfirmed {
		plan.Blockers = append(plan.Blockers, "Choose the source delimiter explicitly.")
	}
	if err := ValidateExport(output); err != nil {
		plan.Blockers = append(plan.Blockers, err.Error())
	}
	if doc.Counts["uneven-row"] > 0 && !repairs.PadRows && !repairs.KeepUneven {
		plan.Blockers = append(plan.Blockers, "Choose to pad short rows or explicitly keep uneven row lengths.")
	}
	if repairs.Headers && !doc.Header {
		plan.Blockers = append(plan.Blockers, "Header repair requires the first-record-is-header option.")
	}
	rows := make([]Row, len(doc.Rows))
	for r, row := range doc.Rows {
		rows[r] = Row{Cells: append([]string{}, row.Cells...), Line: row.Line, Blank: row.Blank}
	}
	add := func(kind string, count int, description string) {
		if count > 0 {
			plan.Changes = append(plan.Changes, Change{kind, count, description})
		}
	}
	if repairs.PadRows {
		n := 0
		for r := range rows {
			if len(rows[r].Cells) < doc.MaxColumns {
				n++
				for len(rows[r].Cells) < doc.MaxColumns {
					rows[r].Cells = append(rows[r].Cells, "")
				}
			}
		}
		add("pad-rows", n, "Adds empty fields to short records up to the widest row. No existing cell, record or extra column is removed.")
	}
	if repairs.Headers && len(rows) > 0 {
		reserved, used := map[string]bool{}, map[string]bool{}
		for _, v := range rows[0].Cells {
			reserved[strings.ToLower(strings.TrimSpace(v))] = true
		}
		n := 0
		for c, v := range rows[0].Cells {
			key := strings.ToLower(strings.TrimSpace(v))
			if key == "" || used[key] {
				base := v
				if key == "" {
					base = fmt.Sprintf("Column_%d", c+1)
				}
				next := base
				for suffix := 2; used[strings.ToLower(strings.TrimSpace(next))] || reserved[strings.ToLower(strings.TrimSpace(next))]; suffix++ {
					next = fmt.Sprintf("%s_%d", base, suffix)
				}
				rows[0].Cells[c] = next
				key = strings.ToLower(strings.TrimSpace(next))
				n++
			}
			used[key] = true
		}
		add("headers", n, "Fills empty headers and renames duplicates without changing non-duplicate names. Generated names avoid existing headers.")
	}
	if repairs.FormulaProtection {
		n := 0
		for r := range rows {
			for c, v := range rows[r].Cells {
				if FormulaRisk(v) {
					rows[r].Cells[c] = "'" + v
					n++
				}
			}
		}
		add("formula-prefix", n, "Prefixes formula-risk cells with an apostrophe. This changes the text; behavior varies by spreadsheet and may not survive resaving. It is not a universal security guarantee.")
	}
	if doc.Encoding != output.Encoding {
		add("encoding", 1, "Writes the selected Unicode UTF-8 encoding; cell text is not converted to numbers or dates.")
	}
	if doc.Delimiter != output.Delimiter {
		add("delimiter", 1, "Changes field separators with appropriate CSV quoting; separators inside cells remain text.")
	}
	add("serialization", 1, "Writes valid escaped CSV quoting and terminated records with the selected line endings. Blank records and line breaks inside cells are preserved.")
	if doc.QuoteMode == "literal" {
		add("quote-interpretation", 1, "Uses your explicit literal-quote interpretation. This may differ from the intended structure of malformed source data.")
	}
	plan.After = preview(rows)
	after := Document{Header: doc.Header, Counts: map[string]int{}, Issues: []Issue{}}
	after.scan(rows)
	plan.CanExport = len(plan.Blockers) == 0
	if plan.CanExport {
		plan.Data = Encode(rows, output)
	}
	sum := sha256.Sum256(plan.Data)
	plan.Report = Report{Application: "CSV Doctor", Version: Version, CreatedUTC: time.Now().UTC().Format(time.RFC3339), SourceName: doc.Name, SourceSHA256: doc.SHA256, SourceEncoding: doc.Encoding, SourceDelimiter: doc.Delimiter, SourceLineEnding: doc.LineEnding, QuoteMode: doc.QuoteMode, Header: doc.Header, OutputSHA256: fmt.Sprintf("%x", sum), Output: output, InputRows: len(doc.Rows), OutputRows: len(rows), Findings: doc.Counts, Changes: plan.Changes, Remaining: after.Counts, Notes: []string{"Reports contain counts and file basenames, not cell contents or full source paths.", "Export is a copy, not a claim that every spreadsheet-import problem or security risk has been resolved.", "CSV has no cell types. Import identifier columns as Text to prevent Excel number/date conversion.", "UTF-8 BOM helps encoding compatibility, not formula safety or identifier preservation in Excel."}}
	return plan
}
func Encode(rows []Row, o ExportOptions) []byte {
	var out bytes.Buffer
	if o.Encoding == "utf8bom" {
		out.Write([]byte{0xef, 0xbb, 0xbf})
	}
	for _, row := range rows {
		for c, v := range row.Cells {
			if c > 0 {
				out.WriteString(o.Delimiter)
			}
			quote := strings.ContainsAny(v, "\"\r\n"+o.Delimiter) || strings.TrimSpace(v) != v || (len(row.Cells) == 1 && v == "")
			if quote {
				out.WriteByte('"')
				out.WriteString(strings.ReplaceAll(v, "\"", "\"\""))
				out.WriteByte('"')
			} else {
				out.WriteString(v)
			}
		}
		out.WriteString(o.LineEnding)
	}
	return out.Bytes()
}
