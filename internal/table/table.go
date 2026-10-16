// Package table loads 1C CSV exports and groups their rows by department.
package table

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const (
	// InfoLines is the number of informational lines before the header.
	InfoLines = 2
	// DeptColumn is the zero-based index of the department column.
	DeptColumn = 2
	// NoDept is the display name for rows with an empty department cell.
	NoDept = "(подразделение не указано)"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Doc is a parsed CSV file.
type Doc struct {
	Path   string
	BOM    bool
	Info   [][]string
	Header []string
	Rows   [][]string
}

// Department is a group of rows sharing the same department cell value.
type Department struct {
	Name string
	Rows [][]string
}

// Load reads and parses a CSV file.
func Load(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	doc.Path = path
	return doc, nil
}

// Parse parses CSV data: ';' separator, UTF-8 (Windows-1251 is accepted as a fallback).
func Parse(data []byte) (*Doc, error) {
	doc := &Doc{}
	if bytes.HasPrefix(data, utf8BOM) {
		doc.BOM = true
		data = data[len(utf8BOM):]
	}
	if !utf8.Valid(data) {
		decoded, err := charmap.Windows1251.NewDecoder().Bytes(data)
		if err == nil {
			data = decoded
		}
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = ';'
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	line := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения CSV: %w", err)
		}
		line++
		switch {
		case line <= InfoLines:
			doc.Info = append(doc.Info, rec)
		case line == InfoLines+1:
			doc.Header = rec
		default:
			if isEmpty(rec) {
				continue
			}
			doc.Rows = append(doc.Rows, rec)
		}
	}
	if doc.Header == nil {
		return nil, errors.New("в файле нет строки шапки (3-я строка)")
	}
	if len(doc.Header) <= DeptColumn {
		return nil, fmt.Errorf("в шапке меньше %d столбцов — нет столбца подразделения", DeptColumn+1)
	}
	return doc, nil
}

// Departments groups rows by the department column, sorted by name.
func (d *Doc) Departments() []Department {
	idx := map[string]int{}
	var out []Department
	for _, row := range d.Rows {
		name := ""
		if len(row) > DeptColumn {
			name = strings.TrimSpace(row[DeptColumn])
		}
		if name == "" {
			name = NoDept
		}
		i, ok := idx[name]
		if !ok {
			i = len(out)
			idx[name] = i
			out = append(out, Department{Name: name})
		}
		out[i].Rows = append(out[i].Rows, row)
	}
	sort.SliceStable(out, func(a, b int) bool {
		return naturalLess(out[a].Name, out[b].Name)
	})
	return out
}

func isEmpty(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}

// naturalLess compares strings case-insensitively, treating digit runs as numbers.
func naturalLess(a, b string) bool {
	ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		if isDigit(ra[i]) && isDigit(rb[j]) {
			si := i
			for i < len(ra) && isDigit(ra[i]) {
				i++
			}
			sj := j
			for j < len(rb) && isDigit(rb[j]) {
				j++
			}
			na := strings.TrimLeft(string(ra[si:i]), "0")
			nb := strings.TrimLeft(string(rb[sj:j]), "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		if ra[i] != rb[j] {
			return ra[i] < rb[j]
		}
		i++
		j++
	}
	return len(ra)-i < len(rb)-j
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }
