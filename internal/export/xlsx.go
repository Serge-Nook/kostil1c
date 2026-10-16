package export

import (
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

func itoa(n int) string { return strconv.Itoa(n) }

func sheetName(title string) string {
	r := strings.NewReplacer("[", "(", "]", ")", ":", "_", "*", "_", "?", "_", "/", "_", "\\", "_")
	s := strings.Trim(r.Replace(title), "'")
	if rs := []rune(s); len(rs) > 31 {
		s = string(rs[:31])
	}
	if strings.TrimSpace(s) == "" {
		s = "Лист1"
	}
	return s
}

func writeXLSX(path string, s *Sheet) error {
	x := excelize.NewFile()
	defer x.Close()
	name := sheetName(s.Title)
	if err := x.SetSheetName("Sheet1", name); err != nil {
		return err
	}
	cols := s.Columns()

	headerStyle, err := x.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"DDE5F0"}},
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"},
		Border:    borders(),
	})
	if err != nil {
		return err
	}
	cellStyle, err := x.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top"},
		Border:    borders(),
	})
	if err != nil {
		return err
	}

	rowNum := 1
	for _, rec := range s.Info {
		if err := setRow(x, name, rowNum, rec); err != nil {
			return err
		}
		rowNum++
	}
	headerRow := rowNum
	header := make([]string, cols)
	copy(header, s.Header)
	if err := setRow(x, name, rowNum, header); err != nil {
		return err
	}
	rowNum++
	for _, rec := range s.Rows {
		if err := setRow(x, name, rowNum, rec); err != nil {
			return err
		}
		rowNum++
	}
	first, _ := excelize.CoordinatesToCellName(1, headerRow)
	hdrEnd, _ := excelize.CoordinatesToCellName(cols, headerRow)
	last, _ := excelize.CoordinatesToCellName(cols, rowNum-1)
	x.SetCellStyle(name, first, hdrEnd, headerStyle)
	if rowNum-1 > headerRow {
		start, _ := excelize.CoordinatesToCellName(1, headerRow+1)
		x.SetCellStyle(name, start, last, cellStyle)
	}
	x.AutoFilter(name, first+":"+last, nil)
	x.SetPanes(name, &excelize.Panes{
		Freeze: true, YSplit: headerRow, TopLeftCell: "A" + itoa(headerRow+1), ActivePane: "bottomLeft",
	})

	for c := 1; c <= cols; c++ {
		width := utf8.RuneCountInString(cell(s.Header, c-1))
		for _, rec := range s.Rows {
			if n := utf8.RuneCountInString(cell(rec, c-1)); n > width {
				width = n
			}
		}
		w := float64(width) + 2
		if w < 8 {
			w = 8
		}
		if w > 60 {
			w = 60
		}
		colName, _ := excelize.ColumnNumberToName(c)
		x.SetColWidth(name, colName, colName, w)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := x.WriteTo(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func setRow(x *excelize.File, sheet string, row int, rec []string) error {
	if len(rec) == 0 {
		return nil
	}
	vals := make([]interface{}, len(rec))
	for i, v := range rec {
		vals[i] = v
	}
	return x.SetSheetRow(sheet, "A"+itoa(row), &vals)
}

func borders() []excelize.Border {
	var b []excelize.Border
	for _, side := range []string{"left", "right", "top", "bottom"} {
		b = append(b, excelize.Border{Type: side, Color: "9AA4B1", Style: 1})
	}
	return b
}
