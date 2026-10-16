package export

import (
	_ "embed"
	"math"
	"strings"

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
	pdfMinCol  = 6.0
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

type pdfPage struct {
	w, h float64
	font float64
}

// Landscape sheets tried in order until the table fits at its natural width.
var pdfPages = []pdfPage{
	{297, 210, 8}, // A4
	{420, 297, 8}, // A3
	{420, 297, 7},
	{594, 420, 7}, // A2
	{594, 420, 6},
}

func newPDF(w, h float64) *fpdf.Fpdf {
	pdf := fpdf.NewCustom(&fpdf.InitType{UnitStr: "mm", Size: fpdf.SizeType{Wd: w, Ht: h}})
	pdf.AddUTF8FontFromBytes("DejaVu", "", fontRegular)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", fontBold)
	// Padding is applied explicitly; the default cell margin would also shrink SplitText widths.
	pdf.SetCellMargin(0)
	return pdf
}

func writePDF(path string, s *Sheet) error {
	cols := s.Columns()
	header := make([]string, cols)
	copy(header, s.Header)

	// Columns are as wide as their longest value so every cell stays on one line;
	// the sheet grows (and the font shrinks slightly) when the table is too wide.
	measure := newPDF(297, 210)
	var page pdfPage
	var widths []float64
	sum := 0.0
	for _, p := range pdfPages {
		widths, sum = naturalWidths(measure, header, s.Rows, p.font)
		page = p
		if sum <= p.w-2*pdfMargin {
			break
		}
	}
	if sum > page.w-2*pdfMargin {
		page.w = sum + 2*pdfMargin
	}

	pdf := newPDF(page.w, page.h)
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.SetAutoPageBreak(false, pdfMargin)
	pdf.SetTitle(s.Title, true)
	pdf.SetCreator("Костыль 1С", true)
	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-pdfMargin - 3)
		pdf.SetFont("DejaVu", "", 7)
		pdf.SetTextColor(110, 110, 110)
		pageNo := "Стр. " + itoa(pdf.PageNo()) + " из {nb}"
		pdf.CellFormat(0, 4, fitText(pdf, s.Title, page.w-2*pdfMargin-pdf.GetStringWidth(pageNo)-10), "", 0, "L", false, 0, "")
		pdf.SetX(pdfMargin)
		pdf.CellFormat(0, 4, pageNo, "", 0, "R", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})

	avail := page.w - 2*pdfMargin
	t := &pdfTable{
		pdf: pdf, header: header, widths: widths, fontSize: page.font,
		lineH: page.font * ptToMM * lineFactor, bottom: page.h - pdfMargin - pdfFooter,
	}

	pdf.AddPage()
	pdf.SetFont("DejaVu", "B", 13)
	pdf.MultiCell(avail, 6, s.Title, "", "L", false)
	pdf.SetFont("DejaVu", "", 8)
	for _, line := range s.InfoText() {
		pdf.MultiCell(avail, 4, line, "", "L", false)
	}
	pdf.MultiCell(avail, 4, "Всего сотрудников: "+itoa(len(s.Rows)), "", "L", false)
	pdf.Ln(2)

	t.drawRow(header, true)
	for _, row := range s.Rows {
		t.ensureSpace(t.rowHeight(row, false))
		t.drawRow(row, false)
	}
	return pdf.OutputFileAndClose(path)
}

// naturalWidths returns per-column widths that fit the longest cell line
// (header cells may wrap between words) and their sum.
func naturalWidths(pdf *fpdf.Fpdf, header []string, rows [][]string, size float64) ([]float64, float64) {
	w := make([]float64, len(header))
	sum := 0.0
	for c := range header {
		pdf.SetFont("DejaVu", "B", size)
		m := longestWord(pdf, header[c])
		pdf.SetFont("DejaVu", "", size)
		for _, r := range rows {
			for _, line := range strings.Split(cell(r, c), "\n") {
				if v := pdf.GetStringWidth(line); v > m {
					m = v
				}
			}
		}
		w[c] = math.Max(m+2*pdfPad+0.8, pdfMinCol)
		sum += w[c]
	}
	return w, sum
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
