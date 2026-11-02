package export

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"unicode/utf16"
	"unicode/utf8"
)

// Legacy Excel 97–2003 workbook: a BIFF8 "Workbook" stream inside an OLE2 compound file.

const (
	xlsMaxRows    = 65536
	xlsMaxCols    = 256
	xlsMaxRecData = 8224

	xfDefault = 15
	xfHeader  = 16
	xfCell    = 17
)

type biff struct{ bytes.Buffer }

func (b *biff) rec(typ uint16, data []byte) {
	binary.Write(&b.Buffer, binary.LittleEndian, typ)
	binary.Write(&b.Buffer, binary.LittleEndian, uint16(len(data)))
	b.Write(data)
}

func le(vals ...any) []byte {
	var b bytes.Buffer
	for _, v := range vals {
		binary.Write(&b, binary.LittleEndian, v)
	}
	return b.Bytes()
}

func utf16le(s string) []byte {
	u := utf16.Encode([]rune(s))
	out := make([]byte, 2*len(u))
	for i, c := range u {
		binary.LittleEndian.PutUint16(out[2*i:], c)
	}
	return out
}

// xlsCellText limits a value to the 32767 characters an Excel cell can hold.
func xlsCellText(s string) []uint16 {
	u := utf16.Encode([]rune(s))
	if len(u) > 32767 {
		u = u[:32767]
	}
	return u
}

func fontRec(bold bool) []byte {
	weight := uint16(400)
	if bold {
		weight = 700
	}
	name := "Arial"
	d := le(uint16(200), uint16(0), uint16(0x7FFF), weight, uint16(0), uint8(0), uint8(0), uint8(204), uint8(0),
		uint8(len(name)), uint8(1))
	return append(d, utf16le(name)...)
}

// xfRec builds a BIFF8 XF record. border/fill colors are palette indexes.
func xfRec(font uint16, style bool, align uint8, used uint8, border bool, fill uint8) []byte {
	flags := uint16(0x0001) // locked, parent = style XF 0
	if style {
		flags = 0xFFF5
	}
	var b1, b2 uint32
	fore, back := uint16(0x40), uint16(0x41)
	if border {
		const thin, gray = 1, 0x17
		b1 = thin | thin<<4 | thin<<8 | thin<<12 | gray<<16 | gray<<23
		b2 = gray | gray<<7
	}
	if fill != 0 {
		b2 |= 1 << 26
		fore = uint16(fill)
	}
	return le(font, uint16(0), flags, align, uint8(0), uint8(0), used, b1, b2, fore|back<<7)
}

// sst collects unique strings for the shared string table.
type sst struct {
	index map[string]int
	strs  [][]uint16
	total int
}

func (s *sst) add(v string) int {
	s.total++
	if i, ok := s.index[v]; ok {
		return i
	}
	i := len(s.strs)
	s.index[v] = i
	s.strs = append(s.strs, xlsCellText(v))
	return i
}

// write appends SST (+CONTINUE) and EXTSST records; base is the stream offset of b.
func (s *sst) write(b *biff, base int) {
	dsst := len(s.strs)/128 + 1
	if dsst < 8 {
		dsst = 8
	}
	type ext struct{ ib, cb int }
	var exts []ext

	var recs [][]byte
	cur := le(uint32(s.total), uint32(len(s.strs)))
	recStart := b.Len() // stream offset (relative to b) of the current record header
	flush := func() {
		recs = append(recs, cur)
		recStart += 4 + len(cur)
		cur = nil
	}
	for i, str := range s.strs {
		if len(cur)+3+2 > xlsMaxRecData {
			flush()
		}
		if i%dsst == 0 {
			exts = append(exts, ext{ib: base + recStart + 4 + len(cur), cb: 4 + len(cur)})
		}
		cur = append(cur, le(uint16(len(str)), uint8(1))...)
		rest := str
		for len(rest) > 0 {
			room := (xlsMaxRecData - len(cur)) / 2
			if room == 0 {
				flush()
				cur = []byte{1}
				continue
			}
			n := min(room, len(rest))
			for _, c := range rest[:n] {
				cur = binary.LittleEndian.AppendUint16(cur, c)
			}
			rest = rest[n:]
		}
	}
	flush()
	for i, r := range recs {
		typ := uint16(0x003C) // CONTINUE
		if i == 0 {
			typ = 0x00FC // SST
		}
		b.rec(typ, r)
	}
	e := le(uint16(dsst))
	for _, x := range exts {
		e = append(e, le(uint32(x.ib), uint16(x.cb), uint16(0))...)
	}
	b.rec(0x00FF, e)
}

func writeXLS(path string, s *Sheet) error {
	cols := s.Columns()
	header := make([]string, cols)
	copy(header, s.Header)

	type cellRef struct {
		row, col, xf int
		str          int
	}
	strs := &sst{index: map[string]int{}}
	var cells []cellRef
	row := s.Blank
	addRow := func(rec []string, xf func(c int) int) {
		for c, v := range rec {
			if v == "" && xf(c) == xfDefault {
				continue
			}
			cells = append(cells, cellRef{row: row, col: c, xf: xf(c), str: strs.add(v)})
		}
		row++
	}
	for _, rec := range s.Info {
		addRow(rec, func(int) int { return xfDefault })
	}
	headerRow := row
	addRow(header, func(int) int { return xfHeader })
	for _, rec := range s.Rows {
		full := make([]string, cols)
		copy(full, rec)
		addRow(full, func(int) int { return xfCell })
	}
	if row > xlsMaxRows {
		return fmt.Errorf("в формате .xls не больше %d строк (нужно %d) — используйте XLSX", xlsMaxRows, row)
	}
	if cols > xlsMaxCols {
		return fmt.Errorf("в формате .xls не больше %d столбцов (нужно %d) — используйте XLSX", xlsMaxCols, cols)
	}

	// Workbook globals.
	g := &biff{}
	g.rec(0x0809, le(uint16(0x0600), uint16(0x0005), uint16(0x0DBB), uint16(0x07CC), uint32(0), uint32(6)))
	g.rec(0x0042, le(uint16(1200)))
	g.rec(0x003D, le(uint16(0x01E0), uint16(0x005A), uint16(0x3FCF), uint16(0x2A4E), uint16(0x38), uint16(0), uint16(0), uint16(1), uint16(0x258)))
	for i := 0; i < 4; i++ {
		g.rec(0x0031, fontRec(false))
	}
	g.rec(0x0031, fontRec(true)) // font index 5 (index 4 is skipped by Excel)
	g.rec(0x00E0, xfRec(0, true, 0x20, 0x00, false, 0))
	for i := 1; i < 15; i++ {
		g.rec(0x00E0, xfRec(0, true, 0x20, 0xF4, false, 0))
	}
	g.rec(0x00E0, xfRec(0, false, 0x20, 0x00, false, 0))     // 15: default cell
	g.rec(0x00E0, xfRec(5, false, 0x08, 0x78, true, 0x2C))   // 16: header — bold, wrap, top, borders, fill
	g.rec(0x00E0, xfRec(0, false, 0x00, 0x30, true, 0))      // 17: data — top, borders
	g.rec(0x0293, le(uint16(0x8000), uint8(0), uint8(0xFF))) // built-in "Normal" style
	name := []rune(sheetName(s.Title))
	bsPos := g.Len() + 4
	g.rec(0x0085, append(le(uint32(0), uint16(0), uint8(len(name)), uint8(1)), utf16le(string(name))...))
	strs.write(g, 0)
	g.rec(0x000A, nil)
	binary.LittleEndian.PutUint32(g.Bytes()[bsPos:], uint32(g.Len()))

	// Worksheet.
	w := &biff{}
	w.rec(0x0809, le(uint16(0x0600), uint16(0x0010), uint16(0x0DBB), uint16(0x07CC), uint32(0), uint32(6)))
	for c := 0; c < cols; c++ {
		width := utf8.RuneCountInString(header[c])
		for _, rec := range s.Rows {
			if n := utf8.RuneCountInString(cell(rec, c)); n > width {
				width = n
			}
		}
		width = min(max(width+2, 8), 60)
		w.rec(0x007D, le(uint16(c), uint16(c), uint16(width*256), uint16(xfDefault), uint16(0), uint16(0)))
	}
	w.rec(0x0200, le(uint32(0), uint32(row), uint16(0), uint16(max(cols, 1)), uint16(0)))
	for _, c := range cells {
		w.rec(0x00FD, le(uint16(c.row), uint16(c.col), uint16(c.xf), uint32(c.str)))
	}
	frozen := uint16(headerRow + 1)
	w.rec(0x023E, le(uint16(0x06B6|0x0008|0x0100), uint16(0), uint16(0), uint16(0x40), uint16(0), uint16(0), uint16(0), uint16(0), uint16(0)))
	w.rec(0x0041, le(uint16(0), frozen, frozen, uint16(0), uint16(2)))
	w.rec(0x000A, nil)

	stream := append(g.Bytes(), w.Bytes()...)
	return os.WriteFile(path, compoundFile("Workbook", stream), 0o644)
}

// compoundFile wraps one stream into an OLE2 compound document (version 3, 512-byte sectors).
func compoundFile(streamName string, stream []byte) []byte {
	const (
		secSize    = 512
		perSec     = secSize / 4
		endOfChain = 0xFFFFFFFE
		freeSect   = 0xFFFFFFFF
		fatSect    = 0xFFFFFFFD
		difSect    = 0xFFFFFFFC
		noStream   = 0xFFFFFFFF
	)
	if len(stream) < 4096 { // smaller streams would belong to the mini stream
		stream = append(stream, make([]byte, 4096-len(stream))...)
	}
	dataSecs := (len(stream) + secSize - 1) / secSize
	nFat, nDif := 0, 0
	for {
		total := dataSecs + 1 + nFat + nDif
		needFat := (total + perSec - 1) / perSec
		needDif := 0
		if needFat > 109 {
			needDif = (needFat - 109 + perSec - 2) / (perSec - 1)
		}
		if needFat == nFat && needDif == nDif {
			break
		}
		nFat, nDif = needFat, needDif
	}
	dirSec := dataSecs
	fatStart := dirSec + 1
	difStart := fatStart + nFat

	fat := make([]uint32, nFat*perSec)
	for i := range fat {
		fat[i] = freeSect
	}
	for i := 0; i < dataSecs-1; i++ {
		fat[i] = uint32(i + 1)
	}
	fat[dataSecs-1] = endOfChain
	fat[dirSec] = endOfChain
	for i := 0; i < nFat; i++ {
		fat[fatStart+i] = fatSect
	}
	for i := 0; i < nDif; i++ {
		fat[difStart+i] = difSect
	}

	var out bytes.Buffer
	hdr := le([8]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, [16]byte{}, uint16(0x003E), uint16(0x0003),
		uint16(0xFFFE), uint16(9), uint16(6), [6]byte{}, uint32(0), uint32(nFat), uint32(dirSec), uint32(0),
		uint32(4096), uint32(endOfChain), uint32(0))
	firstDif := uint32(endOfChain)
	if nDif > 0 {
		firstDif = uint32(difStart)
	}
	hdr = append(hdr, le(firstDif, uint32(nDif))...)
	for i := 0; i < 109; i++ {
		v := uint32(freeSect)
		if i < nFat {
			v = uint32(fatStart + i)
		}
		hdr = append(hdr, le(v)...)
	}
	out.Write(hdr)

	out.Write(stream)
	out.Write(make([]byte, dataSecs*secSize-len(stream)))

	dirEntry := func(name string, typ uint8, child uint32, start uint32, size uint32) []byte {
		n := make([]byte, 64)
		if name != "" {
			copy(n, utf16le(name))
		}
		nameLen := uint16(0)
		if name != "" {
			nameLen = uint16(2 * (len(utf16.Encode([]rune(name))) + 1))
		}
		color := uint8(1)
		if typ == 0 {
			color = 0
		}
		return le(n, nameLen, typ, color, uint32(noStream), uint32(noStream), child, [16]byte{}, uint32(0),
			uint64(0), uint64(0), start, uint32(size), uint32(0))
	}
	out.Write(dirEntry("Root Entry", 5, 1, endOfChain, 0))
	out.Write(dirEntry(streamName, 2, noStream, 0, uint32(len(stream))))
	out.Write(dirEntry("", 0, noStream, 0, 0))
	out.Write(dirEntry("", 0, noStream, 0, 0))

	for _, v := range fat {
		out.Write(le(v))
	}
	for d := 0; d < nDif; d++ {
		for i := 0; i < perSec-1; i++ {
			idx := 109 + d*(perSec-1) + i
			v := uint32(freeSect)
			if idx < nFat {
				v = uint32(fatStart + idx)
			}
			out.Write(le(v))
		}
		next := uint32(endOfChain)
		if d < nDif-1 {
			next = uint32(difStart + d + 1)
		}
		out.Write(le(next))
	}
	return out.Bytes()
}
