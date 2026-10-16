package export

import (
	_ "embed"
	"math"

	"github.com/go-pdf/fpdf"
)

var (
	//go:embed fonts/DejaVuSans.ttf
	fontRegular []byte
	//go:embed fonts/DejaVuSans-Bold.ttf
	fontBold []byte
)

const (
	pdfMargin  = 10.0
	pdfFooter  = 7.0
	pdfPad     = 1.0
	pdfMinCol  = 10.0
	pdfMaxCol  = 80.0
	ptToMM     = 25.4 / 72
	lineFactor = 1.25
)

type pdfTable struct {
	pdf      *fpdf.Fpdf
	widths   []float64
	fontSize float64
	lineH    float64
	bottom   float64
	header   []string
}

func writePDF(path string, s *Sheet) error {
	cols := s.Columns()
	page := "A4"
	if cols > 14 {
		page = "A3"
	}
	pdf := fpdf.New("L", "mm", page, "")
	pdf.AddUTF8FontFromBytes("DejaVu", "", fontRegular)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", fontBold)
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.SetAutoPageBreak(false, pdfMargin)
	pdf.SetTitle(s.Title, true)
	pdf.SetCreator("Костыль 1С", true)
	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-pdfMargin - 3)
		pdf.SetFont("DejaVu", "", 7)
		pdf.SetTextColor(110, 110, 110)
		pageW, _ := pdf.GetPageSize()
		pageNo := "Стр. " + itoa(pdf.PageNo()) + " из {nb}"
		pdf.CellFormat(0, 4, fitText(pdf, s.Title, pageW-2*pdfMargin-pdf.GetStringWidth(pageNo)-10), "", 0, "L", false, 0, "")
		pdf.SetX(pdfMargin)
		pdf.CellFormat(0, 4, pageNo, "", 0, "R", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})

	pageW, pageH := pdf.GetPageSize()
	avail := pageW - 2*pdfMargin

	header := make([]string, cols)
	copy(header, s.Header)
	t := &pdfTable{pdf: pdf, header: header, bottom: pageH - pdfMargin - pdfFooter}
	t.layout(s.Rows, avail)

	pdf.AddPage()
	pdf.SetFont("DejaVu", "B", 13)
	pdf.MultiCell(avail, 6, s.Title, "", "L", false)
	pdf.SetFont("DejaVu", "", 8)
	for _, line := range s.InfoText() {
		pdf.MultiCell(avail, 4, line, "", "L", false)
	}
	pdf.SetFont("DejaVu", "", 8)
	pdf.MultiCell(avail, 4, "Всего сотрудников: "+itoa(len(s.Rows)), "", "L", false)
	pdf.Ln(2)

	t.drawRow(header, true)
	for _, row := range s.Rows {
		t.ensureSpace(t.rowHeight(row, false))
		t.drawRow(row, false)
	}
	return pdf.OutputFileAndClose(path)
}

// layout picks the font size and column widths so the table spans the page width.
func (t *pdfTable) layout(rows [][]string, avail float64) {
	cols := len(t.header)
	natural := func(size float64) ([]float64, float64) {
		w := make([]float64, cols)
		sum := 0.0
		for c := 0; c < cols; c++ {
			t.pdf.SetFont("DejaVu", "B", size)
			m := longestWord(t.pdf, t.header[c])
			t.pdf.SetFont("DejaVu", "", size)
			for _, r := range rows {
				if v := t.pdf.GetStringWidth(cell(r, c)); v > m {
					m = v
				}
			}
			m = math.Min(math.Max(m+2*pdfPad+0.5, pdfMinCol), pdfMaxCol)
			w[c] = m
			sum += m
		}
		return w, sum
	}
	size := 8.0
	w, sum := natural(size)
	for size > 5.5 && sum > avail*1.8 {
		size -= 0.5
		w, sum = natural(size)
	}
	k := avail / sum
	for i := range w {
		w[i] *= k
	}
	t.widths = w
	t.fontSize = size
	t.lineH = size * ptToMM * lineFactor
}

func longestWord(pdf *fpdf.Fpdf, s string) float64 {
	m := 0.0
	start := 0
	rs := []rune(s)
	for i := 0; i <= len(rs); i++ {
		if i == len(rs) || rs[i] == ' ' || rs[i] == '\n' {
			if v := pdf.GetStringWidth(string(rs[start:i])); v > m {
				m = v
			}
			start = i + 1
		}
	}
	return m
}

func (t *pdfTable) setFont(bold bool) {
	style := ""
	if bold {
		style = "B"
	}
	t.pdf.SetFont("DejaVu", style, t.fontSize)
}

func (t *pdfTable) lines(row []string, bold bool) [][]string {
	t.setFont(bold)
	out := make([][]string, len(t.widths))
	for c, w := range t.widths {
		txt := cell(row, c)
		if txt == "" {
			continue
		}
		out[c] = t.pdf.SplitText(txt, w-2*pdfPad)
	}
	return out
}

func (t *pdfTable) rowHeight(row []string, bold bool) float64 {
	n := 1
	for _, l := range t.lines(row, bold) {
		if len(l) > n {
			n = len(l)
		}
	}
	return float64(n)*t.lineH + 2*pdfPad
}

func (t *pdfTable) ensureSpace(h float64) {
	if t.pdf.GetY()+h <= t.bottom {
		return
	}
	t.pdf.AddPage()
	t.drawRow(t.header, true)
}

func (t *pdfTable) drawRow(row []string, bold bool) {
	pdf := t.pdf
	lines := t.lines(row, bold)
	h := t.rowHeight(row, bold)
	x, y := pdfMargin, pdf.GetY()
	pdf.SetDrawColor(154, 164, 177)
	pdf.SetLineWidth(0.2)
	if bold {
		pdf.SetFillColor(221, 229, 240)
	}
	t.setFont(bold)
	for c, w := range t.widths {
		style := "D"
		if bold {
			style = "FD"
		}
		pdf.Rect(x, y, w, h, style)
		for i, line := range lines[c] {
			pdf.SetXY(x+pdfPad, y+pdfPad+float64(i)*t.lineH)
			pdf.CellFormat(w-2*pdfPad, t.lineH, line, "", 0, "L", false, 0, "")
		}
		x += w
	}
	pdf.SetXY(pdfMargin, y+h)
}

// fitText shortens s with an ellipsis so that it fits into width w.
func fitText(pdf *fpdf.Fpdf, s string, w float64) string {
	if pdf.GetStringWidth(s) <= w {
		return s
	}
	rs := []rune(s)
	for len(rs) > 0 && pdf.GetStringWidth(string(rs)+"…") > w {
		rs = rs[:len(rs)-1]
	}
	return string(rs) + "…"
}
