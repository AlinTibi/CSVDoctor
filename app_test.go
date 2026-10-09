package main

import (
	"github.com/AlinTibi/CSVDoctor/internal/doctor"
	"os"
	"path/filepath"
	"testing"
)

func TestAppRejectsStaleReview(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.csv")
	if err := os.WriteFile(path, []byte("a,b\n001,12345678901234567\n"), 0600); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	view, err := app.LoadFile(path, defaults())
	if err != nil {
		t.Fatal(err)
	}
	o := doctor.ExportOptions{Encoding: "utf8bom", Delimiter: ",", LineEnding: "\r\n"}
	review, err := app.Review(view.Revision, doctor.Repairs{}, o)
	if err != nil || !review.Plan.CanExport {
		t.Fatal(err)
	}
	if _, err = app.Diagnose(defaults()); err != nil {
		t.Fatal(err)
	}
	if _, err = app.Review(view.Revision, doctor.Repairs{}, o); err == nil {
		t.Fatal("stale source revision accepted")
	}
	if _, err = app.Export(review.Token); err == nil {
		t.Fatal("stale export accepted")
	}
}
func TestOpenFailureKeepsCurrentSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.csv")
	os.WriteFile(path, []byte("a,b\n1,2\n"), 0600)
	app := NewApp()
	view, _ := app.LoadFile(path, defaults())
	if _, err := app.LoadFile(filepath.Join(dir, "missing.csv"), defaults()); err == nil {
		t.Fatal("missing file succeeded")
	}
	if app.InitialDocument().Revision != view.Revision {
		t.Fatal("failed open replaced source")
	}
}
