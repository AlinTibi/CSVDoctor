package doctor

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func opts(delimiter string) ParseOptions {
	return ParseOptions{Encoding: "auto", Delimiter: delimiter, Header: true, QuoteMode: "strict"}
}
func output() ExportOptions {
	return ExportOptions{Encoding: "utf8", Delimiter: ",", LineEnding: "\r\n"}
}
func cells(rows []Row) [][]string {
	out := make([][]string, len(rows))
	for i, r := range rows {
		out[i] = r.Cells
	}
	return out
}

func TestDelimiterDetection(t *testing.T) {
	for _, delimiter := range []string{",", ";", "\t", "|"} {
		t.Run(delimiter, func(t *testing.T) {
			input := strings.Join([]string{"id", "name", "note"}, delimiter) + "\n" + strings.Join([]string{"001", "Mara", `"hello` + delimiter + `world"`}, delimiter) + "\n" + strings.Join([]string{"002", "Ionuț", "simple"}, delimiter) + "\n"
			got, confirmed, _ := DetectDelimiter(input)
			if got != delimiter || !confirmed {
				t.Fatalf("got %q confirmed=%v", got, confirmed)
			}
		})
	}
}
func TestAmbiguousDelimiterBlocksExport(t *testing.T) {
	for _, s := range []string{"a,b;c\n1,2;3\n", "single\nvalue\n", ""} {
		t.Run(s, func(t *testing.T) {
			doc := Analyze([]byte(s), "sample.csv", opts("auto"))
			if doc.DelimiterConfirmed {
				t.Fatal("ambiguous delimiter confirmed")
			}
			if Review(doc, Repairs{KeepUneven: true}, output()).CanExport {
				t.Fatal("ambiguous export allowed")
			}
		})
	}
	doc := Analyze([]byte("a,b;c\n1,2;3\n"), "sample.csv", opts(";"))
	if !Review(doc, Repairs{}, output()).CanExport {
		t.Fatal("explicit delimiter must allow reviewed export")
	}
}
func TestStrictQuotes(t *testing.T) {
	for name, input := range map[string]string{"unclosed": "a,b\n1,\"broken", "bare": "a,b\n1,br\"oken\n", "trailing": "a,b\n1,\"ok\"bad\n", "space-after-close": "a,b\n1,\"ok\" \n"} {
		t.Run(name, func(t *testing.T) {
			doc := Analyze([]byte(input), "sample.csv", opts(","))
			if doc.Parsed || doc.Counts["parse-error"] == 0 {
				t.Fatal("malformed quoting accepted")
			}
			if Review(doc, Repairs{KeepUneven: true}, output()).CanExport {
				t.Fatal("failed parse exported")
			}
		})
	}
}
func TestLiteralQuoteChoice(t *testing.T) {
	o := opts(",")
	o.QuoteMode = "literal"
	doc := Analyze([]byte("a,b\n1,br\"oken\n"), "sample.csv", o)
	plan := Review(doc, Repairs{}, output())
	if !plan.CanExport || doc.Counts["literal-quotes"] != 1 {
		t.Fatal("literal interpretation not recorded")
	}
	rows, err := Parse(string(plan.Data), ",", "strict")
	if err != nil || rows[1].Cells[1] != `br"oken` {
		t.Fatal("literal quote text lost")
	}
}
func TestQuotedMultilineRoundTrip(t *testing.T) {
	input := "id,note\r\n001,\"hello\r\nworld, \"\"quoted\"\"\nline\rnext\"\r\n002,done\r\n"
	doc := Analyze([]byte(input), "sample.csv", opts(","))
	if !doc.Parsed || doc.RowCount != 3 || doc.Rows[2].Line != 6 {
		t.Fatalf("multiline parsing: %#v", doc.Rows)
	}
	for _, ending := range []string{"\r\n", "\n"} {
		o := output()
		o.LineEnding = ending
		plan := Review(doc, Repairs{}, o)
		rows, err := Parse(string(plan.Data), ",", "strict")
		if !plan.CanExport || err != nil || !reflect.DeepEqual(cells(doc.Rows), cells(rows)) {
			t.Fatal("cell text or embedded line endings changed")
		}
	}
}
func TestCellsRemainText(t *testing.T) {
	doc := Analyze([]byte("id,long,unicode,formula\n0000123,123456789012345678901234567890,Ștefan 😀,=1+2\n"), "sample.csv", opts(","))
	plan := Review(doc, Repairs{}, output())
	rows, err := Parse(string(plan.Data), ",", "strict")
	if err != nil || !reflect.DeepEqual(cells(rows), cells(doc.Rows)) {
		t.Fatal("text was converted")
	}
	if doc.Counts["identifier-risk"] != 2 || doc.Counts["formula-risk"] != 1 {
		t.Fatal(doc.Counts)
	}
}
func TestFormulaRisk(t *testing.T) {
	for _, s := range []string{"=1+2", "+123", "-42", "@SUM(A1)", " =cmd", "\t=1", "\rtext", "\ntext", "＝SUM(A1)", " ＋1"} {
		if !FormulaRisk(s) {
			t.Errorf("not flagged: %q", s)
		}
	}
	for _, s := range []string{"", "ordinary", "123", "'=1+2", "mail@example.com", "x=y"} {
		if FormulaRisk(s) {
			t.Errorf("false positive: %q", s)
		}
	}
}
func TestOptionalFormulaPrefix(t *testing.T) {
	doc := Analyze([]byte("name,value\nMara,=1+2\n"), "sample.csv", opts(","))
	plan := Review(doc, Repairs{FormulaProtection: true}, output())
	rows, _ := Parse(string(plan.Data), ",", "strict")
	if rows[1].Cells[1] != "'=1+2" || doc.Rows[1].Cells[1] != "=1+2" {
		t.Fatal("prefix or immutable source wrong")
	}
	if plan.Report.Remaining["formula-risk"] != 0 {
		t.Fatal("stale report")
	}
	if !strings.Contains(plan.Changes[len(plan.Changes)-2].Description, "not a universal") {
		t.Fatal("safety limitation missing")
	}
}
func TestUnevenRowsRequireChoice(t *testing.T) {
	doc := Analyze([]byte("a,b\n1\n2,3,4\n"), "sample.csv", opts(","))
	if doc.Counts["uneven-row"] != 2 {
		t.Fatal(doc.Counts)
	}
	if Review(doc, Repairs{}, output()).CanExport {
		t.Fatal("uneven rows silently repaired")
	}
	keep := Review(doc, Repairs{KeepUneven: true}, output())
	rows, _ := Parse(string(keep.Data), ",", "strict")
	if !keep.CanExport || !reflect.DeepEqual(cells(rows), cells(doc.Rows)) || keep.Report.Remaining["uneven-row"] != 2 {
		t.Fatal("keep choice wrong")
	}
	padded := Review(doc, Repairs{PadRows: true}, output())
	rows, _ = Parse(string(padded.Data), ",", "strict")
	if !padded.CanExport || len(rows) != 3 || len(rows[0].Cells) != 3 || rows[2].Cells[2] != "4" || padded.Report.Remaining["uneven-row"] != 0 {
		t.Fatal("padding removed data or report stale")
	}
}
func TestHeaderRepair(t *testing.T) {
	doc := Analyze([]byte("Name,name,,Column_3,name_2\na,b,c,d,e\n"), "sample.csv", opts(","))
	if doc.Counts["duplicate-header"] != 1 || doc.Counts["empty-header"] != 1 {
		t.Fatal(doc.Counts)
	}
	plan := Review(doc, Repairs{Headers: true}, output())
	rows, _ := Parse(string(plan.Data), ",", "strict")
	if !reflect.DeepEqual(rows[0].Cells, []string{"Name", "name_3", "Column_3_2", "Column_3", "name_2"}) {
		t.Fatal(rows[0].Cells)
	}
	if plan.Report.Remaining["duplicate-header"] != 0 || plan.Report.Remaining["empty-header"] != 0 {
		t.Fatal("header report stale")
	}
}
func TestNoHeaderInterpretation(t *testing.T) {
	o := opts(",")
	o.Header = false
	doc := Analyze([]byte("001,Mara\n002,Ionuț\n"), "sample.csv", o)
	if doc.Counts["empty-header"] != 0 || Review(doc, Repairs{Headers: true}, output()).CanExport {
		t.Fatal("headerless data modified")
	}
}
func TestBlankRowsArePreserved(t *testing.T) {
	input := "a\n\n\"\"\nvalue\n\n"
	doc := Analyze([]byte(input), "sample.csv", opts(","))
	if doc.RowCount != 5 || doc.Counts["blank-row"] != 3 {
		t.Fatal(doc.Counts, doc.Rows)
	}
	plan := Review(doc, Repairs{}, output())
	rows, err := Parse(string(plan.Data), ",", "strict")
	if err != nil || !reflect.DeepEqual(cells(rows), cells(doc.Rows)) {
		t.Fatal("empty records removed")
	}
	reopened := Analyze(plan.Data, "copy.csv", opts(","))
	if reopened.Counts["blank-row"] != plan.Report.Remaining["blank-row"] {
		t.Fatal("blank-row diagnostics changed after serialization")
	}
}
func TestPaddedBlankRowDiagnosticsMatchReopenedFile(t *testing.T) {
	doc := Analyze([]byte("a,b\n\n1,2\n,,\n"), "sample.csv", opts(","))
	plan := Review(doc, Repairs{PadRows: true}, output())
	reopened := Analyze(plan.Data, "copy.csv", opts(","))
	if !plan.CanExport || reopened.Counts["blank-row"] != 2 || reopened.Counts["blank-row"] != plan.Report.Remaining["blank-row"] {
		t.Fatal("empty records must stay visible in post-repair diagnostics", reopened.Counts)
	}
}
func TestUTF8AndBOM(t *testing.T) {
	for _, bom := range []bool{false, true} {
		data := []byte("name,value\nIonuț,😀\n")
		if bom {
			data = append([]byte{0xef, 0xbb, 0xbf}, data...)
		}
		text, enc, confirmed, err := Decode(data, "auto")
		if err != nil || !confirmed || !strings.Contains(text, "Ionuț") || (bom && enc != "utf8bom") {
			t.Fatal(enc, confirmed, err)
		}
		doc := Analyze(data, "sample.csv", opts(","))
		for _, dest := range []string{"utf8", "utf8bom"} {
			o := output()
			o.Encoding = dest
			p := Review(doc, Repairs{}, o)
			if bytes.HasPrefix(p.Data, []byte{0xef, 0xbb, 0xbf}) != (dest == "utf8bom") {
				t.Fatal("wrong output BOM")
			}
		}
	}
}
func utf16Bytes(s string, little, bom bool) []byte {
	var out []byte
	if bom {
		if little {
			out = []byte{0xff, 0xfe}
		} else {
			out = []byte{0xfe, 0xff}
		}
	}
	for _, u := range utf16.Encode([]rune(s)) {
		var b [2]byte
		if little {
			binary.LittleEndian.PutUint16(b[:], u)
		} else {
			binary.BigEndian.PutUint16(b[:], u)
		}
		out = append(out, b[:]...)
	}
	return out
}
func TestUTF16(t *testing.T) {
	for _, little := range []bool{true, false} {
		for _, bom := range []bool{true, false} {
			t.Run(string([]byte{byte(map[bool]int{true: 1, false: 0}[little]), byte(map[bool]int{true: 1, false: 0}[bom])}), func(t *testing.T) {
				input := "name,value\r\nIonuț,😀\r\n"
				text, enc, confirmed, err := Decode(utf16Bytes(input, little, bom), "auto")
				if err != nil || text != input || confirmed != bom {
					t.Fatal(enc, confirmed, err, text)
				}
				text, _, confirmed, err = Decode(utf16Bytes(input, little, bom), enc)
				if err != nil || !confirmed || text != input {
					t.Fatal("explicit UTF-16 failure")
				}
			})
		}
	}
}
func TestInvalidEncodingRefused(t *testing.T) {
	cases := []struct {
		data   []byte
		choice string
	}{{[]byte{0xff}, "utf8"}, {[]byte{0xff, 0xfe, 0x00}, "auto"}, {[]byte{0xff, 0xfe, 0x00, 0xd8}, "auto"}, {[]byte{0xff, 0xfe, 0x00, 0xdc}, "auto"}, {[]byte{0x81}, "windows1252"}, {[]byte{0xff, 0xfe, 0, 0, 65, 0, 0, 0}, "auto"}, {[]byte{'a', 0, 'b'}, "utf8"}, {[]byte("a,b"), "unknown"}}
	for _, c := range cases {
		if _, _, _, err := Decode(c.data, c.choice); err == nil {
			t.Errorf("invalid encoding accepted: %x %s", c.data, c.choice)
		}
	}
}
func TestWindows1252RequiresChoice(t *testing.T) {
	data := []byte("name,value\nMara,caf\xe9 \x80\n")
	doc := Analyze(data, "sample.csv", opts(","))
	if doc.Encoding != "windows1252" || doc.EncodingConfirmed || Review(doc, Repairs{}, output()).CanExport {
		t.Fatal("legacy encoding guessed without consent")
	}
	o := opts(",")
	o.Encoding = "windows1252"
	doc = Analyze(data, "sample.csv", o)
	p := Review(doc, Repairs{}, output())
	if !p.CanExport || !strings.Contains(string(p.Data), "café €") {
		t.Fatal("Windows-1252 conversion failed")
	}
}
func TestLineEndingDetection(t *testing.T) {
	for s, want := range map[string]string{"a,b": "None", "a,b\nc,d\n": "LF", "a,b\r\nc,d\r\n": "CRLF", "a,b\rc,d\r": "CR", "a,b\r\nc,d\n": "Mixed"} {
		if got := lineStyle(s); got != want {
			t.Errorf("%q got %s want %s", s, got, want)
		}
	}
}
func TestExportAllDelimiters(t *testing.T) {
	doc := Analyze([]byte("a,b,c\n\"x;y|z\",\"q\tq\",\"with \"\"quotes\"\"\"\n"), "sample.csv", opts(","))
	for _, delim := range []string{",", ";", "\t", "|"} {
		o := output()
		o.Delimiter = delim
		p := Review(doc, Repairs{}, o)
		rows, err := Parse(string(p.Data), delim, "strict")
		if !p.CanExport || err != nil || !reflect.DeepEqual(cells(rows), cells(doc.Rows)) {
			t.Fatalf("round trip %q", delim)
		}
	}
}
func TestInvalidOutputAndParseOptions(t *testing.T) {
	for _, o := range []ExportOptions{{Encoding: "utf16", Delimiter: ",", LineEnding: "\n"}, {Encoding: "utf8", Delimiter: "x", LineEnding: "\n"}, {Encoding: "utf8", Delimiter: ",", LineEnding: "\r"}} {
		if ValidateExport(o) == nil {
			t.Fatal("invalid export accepted")
		}
	}
	if _, err := Parse("a,b", "", "strict"); err == nil {
		t.Fatal("invalid delimiter accepted")
	}
	if _, err := Parse("a,b", ",", "guess"); err == nil {
		t.Fatal("quote guessing accepted")
	}
}
func TestEmptyFileAndLimits(t *testing.T) {
	if Review(Analyze(nil, "empty.csv", opts(",")), Repairs{}, output()).CanExport {
		t.Fatal("empty file exported")
	}
	if _, _, _, err := Decode(make([]byte, MaxBytes+1), "utf8"); err == nil {
		t.Fatal("oversized input accepted")
	}
	if _, err := Parse(strings.Repeat("x,", MaxColumns)+"x", ",", "strict"); err == nil {
		t.Fatal("oversized row accepted")
	}
	if _, err := Parse(strings.Repeat("a\n", MaxRows+1), ",", "strict"); err == nil {
		t.Fatal("too many rows accepted")
	}
}
func TestReportAndSafeCopy(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.csv")
	input := []byte("id,note\n001,private synthetic value\n")
	if err := os.WriteFile(source, input, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	plan := Review(Analyze(data, source, opts(",")), Repairs{}, output())
	target := filepath.Join(dir, "copy.csv")
	report, err := ExportCopy(source, target, plan)
	if err != nil {
		t.Fatal(err)
	}
	saved, _ := os.ReadFile(target)
	original, _ := os.ReadFile(source)
	if !bytes.Equal(original, input) || !bytes.Equal(saved, plan.Data) {
		t.Fatal("source changed or output mismatch")
	}
	reportData, _ := os.ReadFile(report)
	var r Report
	if json.Unmarshal(reportData, &r) != nil || r.SourceName != "source.csv" || strings.Contains(string(reportData), "private synthetic value") || strings.Contains(string(reportData), dir) {
		t.Fatal("report content or privacy failure")
	}
	if _, err = ExportCopy(source, source, plan); err == nil {
		t.Fatal("original overwrite allowed")
	}
	if _, err = ExportCopy(source, target, plan); err == nil {
		t.Fatal("copy overwrite allowed")
	}
	if _, err = ExportCopy(source, filepath.Join(dir, "copy.exe"), plan); err == nil {
		t.Fatal("invalid output extension allowed")
	}
}
func TestExistingReportRefusesWholeExport(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "copy.csv")
	plan := Review(Analyze([]byte("a,b\n1,2\n"), "source.csv", opts(",")), Repairs{}, output())
	if err := os.WriteFile(target+".report.json", []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportCopy(filepath.Join(dir, "source.csv"), target, plan); err == nil {
		t.Fatal("existing report overwritten")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("partial copy created before validation")
	}
}
func TestPreviewTruncationDoesNotTruncateExport(t *testing.T) {
	input := "a,b\n001," + strings.Repeat("😀", 1500) + "\n"
	doc := Analyze([]byte(input), "sample.csv", opts(","))
	if len(doc.Preview[1].Cells[1]) >= len(doc.Rows[1].Cells[1]) {
		t.Fatal("preview not bounded")
	}
	plan := Review(doc, Repairs{}, output())
	rows, _ := Parse(string(plan.Data), ",", "strict")
	if rows[1].Cells[1] != doc.Rows[1].Cells[1] {
		t.Fatal("export truncated")
	}
}
func TestIdentifierRisk(t *testing.T) {
	for _, s := range []string{"001", "000000000000000000000000", "1234567890123456", " 001 "} {
		if !IdentifierRisk(s) {
			t.Errorf("missed %q", s)
		}
	}
	for _, s := range []string{"0", "12345", "date", "1.0", "001x"} {
		if IdentifierRisk(s) {
			t.Errorf("false positive %q", s)
		}
	}
}

func FuzzCellRoundTrip(f *testing.F) {
	for _, s := range []string{"", "00123", "123456789012345678901", "a,b\r\nc", `he said "hello"`, "Ștefan 😀", "=1+2"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if strings.ContainsRune(s, 0) || len(s) > 10000 {
			return
		}
		rows := []Row{{Cells: []string{"key", s}}}
		data := Encode(rows, output())
		got, err := Parse(string(data), ",", "strict")
		if err != nil || !reflect.DeepEqual(cells(rows), cells(got)) {
			t.Fatalf("cell text changed: %q", s)
		}
	})
}
