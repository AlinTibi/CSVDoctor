package doctor

const Version = "1.0.0"
const MaxBytes = 16 * 1024 * 1024
const MaxRows = 100000
const MaxColumns = 256
const MaxCells = 1000000
const PreviewRows = 100
const MaxIssues = 200

type ParseOptions struct {
	Encoding  string `json:"encoding"`
	Delimiter string `json:"delimiter"`
	Header    bool   `json:"header"`
	QuoteMode string `json:"quoteMode"`
}
type ExportOptions struct {
	Encoding   string `json:"encoding"`
	Delimiter  string `json:"delimiter"`
	LineEnding string `json:"lineEnding"`
}
type Repairs struct {
	Headers           bool `json:"headers"`
	PadRows           bool `json:"padRows"`
	KeepUneven        bool `json:"keepUneven"`
	FormulaProtection bool `json:"formulaProtection"`
}
type Row struct {
	Cells []string `json:"cells"`
	Line  int      `json:"line"`
	Blank bool     `json:"blank"`
}
type Issue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Row      int    `json:"row"`
	Column   int    `json:"column"`
}
type Candidate struct {
	Delimiter   string  `json:"delimiter"`
	Columns     int     `json:"columns"`
	Consistency float64 `json:"consistency"`
}
type Document struct {
	Name               string         `json:"name"`
	Bytes              int            `json:"bytes"`
	SHA256             string         `json:"sha256"`
	Encoding           string         `json:"encoding"`
	EncodingConfirmed  bool           `json:"encodingConfirmed"`
	Delimiter          string         `json:"delimiter"`
	DelimiterConfirmed bool           `json:"delimiterConfirmed"`
	Candidates         []Candidate    `json:"candidates"`
	LineEnding         string         `json:"lineEnding"`
	Parsed             bool           `json:"parsed"`
	Header             bool           `json:"header"`
	QuoteMode          string         `json:"quoteMode"`
	RowCount           int            `json:"rowCount"`
	Columns            int            `json:"columns"`
	MaxColumns         int            `json:"maxColumns"`
	Counts             map[string]int `json:"counts"`
	Issues             []Issue        `json:"issues"`
	Preview            []Row          `json:"preview"`
	RawPreview         string         `json:"rawPreview"`
	Rows               []Row          `json:"-"`
}
type Change struct {
	Kind        string `json:"kind"`
	Count       int    `json:"count"`
	Description string `json:"description"`
}
type Report struct {
	Application      string         `json:"application"`
	Version          string         `json:"version"`
	CreatedUTC       string         `json:"createdUTC"`
	SourceName       string         `json:"sourceName"`
	SourceSHA256     string         `json:"sourceSHA256"`
	SourceEncoding   string         `json:"sourceEncoding"`
	SourceDelimiter  string         `json:"sourceDelimiter"`
	SourceLineEnding string         `json:"sourceLineEnding"`
	QuoteMode        string         `json:"quoteMode"`
	Header           bool           `json:"header"`
	OutputSHA256     string         `json:"outputSHA256"`
	Output           ExportOptions  `json:"output"`
	InputRows        int            `json:"inputRows"`
	OutputRows       int            `json:"outputRows"`
	Findings         map[string]int `json:"findings"`
	Changes          []Change       `json:"changes"`
	Remaining        map[string]int `json:"remaining"`
	Notes            []string       `json:"notes"`
}
type Plan struct {
	CanExport bool     `json:"canExport"`
	Blockers  []string `json:"blockers"`
	Changes   []Change `json:"changes"`
	Before    []Row    `json:"before"`
	After     []Row    `json:"after"`
	Report    Report   `json:"report"`
	Data      []byte   `json:"-"`
}
