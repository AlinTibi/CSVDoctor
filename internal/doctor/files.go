package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ReadFile(path string) ([]byte, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv", ".tsv", ".txt":
	default:
		return nil, fmt.Errorf("Open a .csv, .tsv or .txt delimited text file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read the selected file: %w", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("Choose a regular text file")
	}
	if st.Size() > MaxBytes {
		return nil, fmt.Errorf("The selected file exceeds the 16 MiB limit")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("The selected file exceeds the 16 MiB limit")
	}
	return data, nil
}

// Exclusive creation refuses originals, existing copies and existing reports.
// A report failure is reported as a partial export rather than hidden success.
func ExportCopy(source, target string, plan Plan) (string, error) {
	if !plan.CanExport {
		return "", fmt.Errorf("Resolve the review blockers first")
	}
	ext := strings.ToLower(filepath.Ext(target))
	if ext != ".csv" && ext != ".tsv" && ext != ".txt" {
		return "", fmt.Errorf("Choose a .csv, .tsv or .txt output filename")
	}
	absSource, _ := filepath.Abs(source)
	absTarget, _ := filepath.Abs(target)
	if strings.EqualFold(filepath.Clean(absSource), filepath.Clean(absTarget)) {
		return "", fmt.Errorf("The original file cannot be replaced. Choose a new filename")
	}
	reportPath := target + ".report.json"
	for _, path := range []string{target, reportPath} {
		if _, err := os.Lstat(path); err == nil {
			return "", fmt.Errorf("An output file already exists. Choose a new filename; nothing was overwritten")
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	report, err := json.MarshalIndent(plan.Report, "", "  ")
	if err != nil {
		return "", err
	}
	if err = writeExclusive(target, plan.Data); err != nil {
		return "", err
	}
	if err = writeExclusive(reportPath, append(report, '\n')); err != nil {
		return "", fmt.Errorf("CSV copy was saved, but its report could not be saved: %w", err)
	}
	return reportPath, nil
}
func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = os.Remove(path)
	}
	return writeErr
}
