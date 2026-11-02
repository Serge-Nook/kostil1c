// Package export writes one department's rows into CSV, XLSX, PDF and HTML files.
package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Serge-Nook/kostil1c/internal/table"
)

// Format is an output file format.
type Format string

const (
	CSV  Format = "csv"
	XLS  Format = "xls"
	XLSX Format = "xlsx"
	PDF  Format = "pdf"
	HTML Format = "html"
)

// Formats lists all supported formats in UI order.
var Formats = []Format{CSV, XLS, XLSX, PDF, HTML}

// Label returns a human-readable name of the format.
func (f Format) Label() string {
	switch f {
	case CSV:
		return "CSV (разделитель «;», UTF-8)"
	case XLS:
		return "XLS (Excel 97–2003, .xls)"
	case XLSX:
		return "XLSX (Excel 2007+, .xlsx)"
	case PDF:
		return "PDF"
	case HTML:
		return "HTML"
	}
	return string(f)
}

// Options control the content and naming of exported files.
type Options struct {
	// WithInfo copies the informational lines of the source file into every file.
	WithInfo bool
	// BlankRows is the number of empty lines at the very beginning of every file.
	BlankRows int
	// CodeNames names files by the department number only ("037" instead of
	// "037 Отдел главного энергетика"); names without digits are kept.
	CodeNames bool
}

// Sheet is the content of one output file.
type Sheet struct {
	Title  string
	Blank  int
	Info   [][]string
	Header []string
	Rows   [][]string
	BOM    bool
}

// NewSheet builds a sheet for one department of the document.
func NewSheet(doc *table.Doc, dept table.Department, opt Options) *Sheet {
	s := &Sheet{Title: dept.Name, Header: doc.Header, Rows: dept.Rows, BOM: doc.BOM, Blank: max(opt.BlankRows, 0)}
	if opt.WithInfo {
		s.Info = doc.Info
	}
	return s
}

// Columns returns the widest row length among header and data rows.
func (s *Sheet) Columns() int {
	n := len(s.Header)
	for _, r := range s.Rows {
		if len(r) > n {
			n = len(r)
		}
	}
	return n
}

// InfoText joins the non-empty cells of each info line.
func (s *Sheet) InfoText() []string {
	var out []string
	for _, rec := range s.Info {
		var parts []string
		for _, f := range rec {
			if f = strings.TrimSpace(f); f != "" {
				parts = append(parts, f)
			}
		}
		if len(parts) > 0 {
			out = append(out, strings.Join(parts, " "))
		}
	}
	return out
}

// Write saves the sheet to path in the given format.
func Write(f Format, path string, s *Sheet) error {
	switch f {
	case CSV:
		return writeCSV(path, s)
	case XLS:
		return writeXLS(path, s)
	case XLSX:
		return writeXLSX(path, s)
	case PDF:
		return writePDF(path, s)
	case HTML:
		return writeHTML(path, s)
	}
	return fmt.Errorf("неизвестный формат %q", f)
}

// Result describes one finished export run.
type Result struct {
	Files  []string
	Errors []error
}

// Run exports every department in every format into dir.
// progress is called after each written file with (done, total).
func Run(doc *table.Doc, depts []table.Department, formats []Format, dir string, opt Options, progress func(done, total int)) Result {
	var res Result
	if err := os.MkdirAll(dir, 0o755); err != nil {
		res.Errors = append(res.Errors, err)
		return res
	}
	total := len(depts) * len(formats)
	done := 0
	names := NewNamer()
	for _, d := range depts {
		name := d.Name
		if opt.CodeNames {
			name = DeptCode(name)
		}
		base := names.Name(name)
		sheet := NewSheet(doc, d, opt)
		for _, f := range formats {
			path := filepath.Join(dir, base+"."+string(f))
			if err := Write(f, path, sheet); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("%s: %w", filepath.Base(path), err))
			} else {
				res.Files = append(res.Files, path)
			}
			done++
			if progress != nil {
				progress(done, total)
			}
		}
	}
	return res
}
