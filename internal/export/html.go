package export

import (
	"bufio"
	"html"
	"os"
	"strings"
)

const htmlStyle = `body{font-family:"DejaVu Sans",Arial,sans-serif;font-size:13px;margin:16px;color:#1b1f24}
h1{font-size:18px;margin:0 0 8px}
p.info{margin:2px 0;color:#555}
table{border-collapse:collapse;margin-top:10px}
th,td{border:1px solid #9aa4b1;padding:4px 6px;vertical-align:top;text-align:left}
th{background:#dde5f0;position:sticky;top:0}
tr:nth-child(even) td{background:#f5f7fa}
p.total{margin-top:8px;color:#555}
@media print{th{position:static}body{margin:0}}`

func writeHTML(path string, s *Sheet) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	esc := html.EscapeString
	cols := s.Columns()

	w.WriteString("<!DOCTYPE html>\n<html lang=\"ru\">\n<head>\n<meta charset=\"utf-8\">\n<title>")
	w.WriteString(esc(s.Title))
	w.WriteString("</title>\n<style>\n" + htmlStyle + "\n</style>\n</head>\n<body>\n<h1>")
	w.WriteString(esc(s.Title))
	w.WriteString("</h1>\n")
	for _, line := range s.InfoText() {
		w.WriteString("<p class=\"info\">" + esc(line) + "</p>\n")
	}
	w.WriteString("<table>\n<thead>\n<tr>")
	for i := 0; i < cols; i++ {
		w.WriteString("<th>" + esc(cell(s.Header, i)) + "</th>")
	}
	w.WriteString("</tr>\n</thead>\n<tbody>\n")
	for _, row := range s.Rows {
		w.WriteString("<tr>")
		for i := 0; i < cols; i++ {
			w.WriteString("<td>" + strings.ReplaceAll(esc(cell(row, i)), "\n", "<br>") + "</td>")
		}
		w.WriteString("</tr>\n")
	}
	w.WriteString("</tbody>\n</table>\n<p class=\"total\">Всего сотрудников: ")
	w.WriteString(itoa(len(s.Rows)))
	w.WriteString("</p>\n</body>\n</html>\n")
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func cell(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}
