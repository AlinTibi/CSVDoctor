package doctor

import (
	"fmt"
	"sort"
	"strings"
)

func delimiterOK(s string) bool { return s == "," || s == ";" || s == "\t" || s == "|" }

// Parse preserves CR/LF inside quoted fields, blank records and every cell's text.
// Quotes are valid only at a field's start, or doubled within a quoted field.
func Parse(text, delim, mode string) ([]Row, error) {
	if !delimiterOK(delim) {
		return nil, fmt.Errorf("Choose comma, semicolon, tab or pipe")
	}
	if mode != "strict" && mode != "literal" {
		return nil, fmt.Errorf("Choose strict or literal quote interpretation")
	}
	if text == "" {
		return []Row{}, nil
	}
	var rows []Row
	var cells []string
	var cell strings.Builder
	state, line, start, recordStart, total := 0, 1, 1, 0, 0 // 0=start, 1=unquoted, 2=quoted, 3=closed quote
	finishCell := func() error {
		cells = append(cells, cell.String())
		cell.Reset()
		state = 0
		total++
		if len(cells) > MaxColumns || total > MaxCells {
			return fmt.Errorf("Preview limits reached: at most 256 columns and 1,000,000 cells")
		}
		return nil
	}
	finishRow := func(end int) error {
		if err := finishCell(); err != nil {
			return err
		}
		rows = append(rows, Row{Cells: cells, Line: start, Blank: recordStart == end})
		cells = nil
		if len(rows) > MaxRows {
			return fmt.Errorf("This candidate supports at most 100,000 records")
		}
		return nil
	}
	problem := func(message string) ([]Row, error) {
		return rows, fmt.Errorf("Line %d, column %d: %s. No repaired export is available under strict parsing", line, len(cells)+1, message)
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		if mode == "strict" && state == 2 {
			if c == '"' {
				if i+1 < len(text) && text[i+1] == '"' {
					cell.WriteByte('"')
					i++
				} else {
					state = 3
				}
			} else {
				cell.WriteByte(c)
				if c == '\n' || (c == '\r' && (i+1 >= len(text) || text[i+1] != '\n')) {
					line++
				}
			}
			continue
		}
		if c == delim[0] {
			if err := finishCell(); err != nil {
				return rows, err
			}
			continue
		}
		if c == '\r' || c == '\n' {
			if err := finishRow(i); err != nil {
				return rows, err
			}
			if c == '\r' && i+1 < len(text) && text[i+1] == '\n' {
				i++
			}
			line++
			start = line
			recordStart = i + 1
			continue
		}
		if mode == "strict" {
			if state == 3 {
				return problem("Unexpected text after a closing quote")
			}
			if c == '"' {
				if state != 0 {
					return problem("Quote inside an unquoted field")
				}
				state = 2
				continue
			}
		}
		state = 1
		cell.WriteByte(c)
	}
	if mode == "strict" && state == 2 {
		return problem("Unclosed quoted field")
	}
	if recordStart < len(text) {
		if err := finishRow(len(text)); err != nil {
			return rows, err
		}
	}
	return rows, nil
}

func DetectDelimiter(text string) (string, bool, []Candidate) {
	candidates := make([]Candidate, 0, 4)
	for _, delim := range []string{",", ";", "\t", "|"} {
		rows, err := Parse(text, delim, "strict")
		if err != nil || len(rows) == 0 {
			rows, _ = Parse(text, delim, "literal")
		}
		histogram := map[int]int{}
		n := 0
		for _, row := range rows {
			if row.Blank {
				continue
			}
			histogram[len(row.Cells)]++
			n++
			if n >= 200 {
				break
			}
		}
		columns, best := 1, 0
		for width, count := range histogram {
			if count > best || (count == best && width > columns) {
				columns, best = width, count
			}
		}
		consistency := 0.0
		if n > 0 && columns > 1 {
			consistency = float64(best) / float64(n)
		}
		candidates = append(candidates, Candidate{delim, columns, consistency})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Consistency > candidates[j].Consistency })
	best := candidates[0]
	certain := best.Columns > 1 && best.Consistency >= 0.8 && best.Consistency-candidates[1].Consistency >= 0.15
	return best.Delimiter, certain, candidates
}
