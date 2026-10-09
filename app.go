package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/AlinTibi/CSVDoctor/internal/doctor"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	mu       sync.Mutex
	path     string
	input    []byte
	doc      *doctor.Document
	revision string
	token    string
	plan     *doctor.Plan
}
type Snapshot struct {
	Revision string           `json:"revision"`
	Document *doctor.Document `json:"document"`
}
type ReviewResult struct {
	Token string      `json:"token"`
	Plan  doctor.Plan `json:"plan"`
}
type ExportResult struct {
	Cancelled      bool           `json:"cancelled"`
	Filename       string         `json:"filename"`
	ReportFilename string         `json:"reportFilename"`
	Remaining      map[string]int `json:"remaining"`
}

func NewApp() *App { return &App{} }
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("Cannot create an operation identifier")
	}
	return hex.EncodeToString(b[:])
}
func defaults() doctor.ParseOptions {
	return doctor.ParseOptions{Encoding: "auto", Delimiter: "auto", Header: true, QuoteMode: "strict"}
}
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if len(os.Args) == 2 {
		_, _ = a.LoadFile(os.Args[1], defaults())
	}
}
func (a *App) InitialDocument() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return Snapshot{a.revision, a.doc}
}
func (a *App) OpenFile(options doctor.ParseOptions) (*Snapshot, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Open delimited text", Filters: []runtime.FileFilter{{DisplayName: "Delimited text (*.csv; *.tsv; *.txt)", Pattern: "*.csv;*.tsv;*.txt"}}})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	view, err := a.LoadFile(path, options)
	return &view, err
}
func (a *App) LoadFile(path string, options doctor.ParseOptions) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	data, err := doctor.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	a.path, a.input = path, data
	doc := doctor.Analyze(data, filepath.Base(path), options)
	a.doc = &doc
	a.revision = newID()
	a.plan = nil
	a.token = ""
	return Snapshot{a.revision, a.doc}, nil
}
func (a *App) Diagnose(options doctor.ParseOptions) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.doc == nil {
		return Snapshot{}, fmt.Errorf("Open a file first")
	}
	doc := doctor.Analyze(a.input, filepath.Base(a.path), options)
	a.doc = &doc
	a.revision = newID()
	a.plan = nil
	a.token = ""
	return Snapshot{a.revision, a.doc}, nil
}
func (a *App) Review(revision string, repairs doctor.Repairs, output doctor.ExportOptions) (ReviewResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.doc == nil || revision != a.revision {
		return ReviewResult{}, fmt.Errorf("Source interpretation changed. Diagnose and review again")
	}
	plan := doctor.Review(*a.doc, repairs, output)
	a.plan = &plan
	a.token = newID()
	return ReviewResult{a.token, plan}, nil
}
func (a *App) Export(token string) (ExportResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.plan == nil || a.token != token || !a.plan.CanExport {
		return ExportResult{}, fmt.Errorf("Review the current export options before saving")
	}
	name := strings.TrimSuffix(filepath.Base(a.path), filepath.Ext(a.path)) + "-repaired.csv"
	if a.plan.Report.Output.Delimiter == "\t" {
		name = strings.TrimSuffix(name, ".csv") + ".tsv"
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "Export a new copy and repair report", DefaultFilename: name, Filters: []runtime.FileFilter{{DisplayName: "Delimited text", Pattern: "*.csv;*.tsv;*.txt"}}})
	if err != nil {
		return ExportResult{}, err
	}
	if path == "" {
		return ExportResult{Cancelled: true}, nil
	}
	report, err := doctor.ExportCopy(a.path, path, *a.plan)
	if err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Filename: filepath.Base(path), ReportFilename: filepath.Base(report), Remaining: a.plan.Report.Remaining}, nil
}
