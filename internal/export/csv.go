package export

import (
	"bufio"
	"encoding/csv"
	"os"
)

func writeCSV(path string, s *Sheet) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(f)
	if s.BOM {
		bw.Write([]byte{0xEF, 0xBB, 0xBF})
	}
	w := csv.NewWriter(bw)
	w.Comma = ';'
	w.UseCRLF = true
	for _, rec := range s.Info {
		w.Write(rec)
	}
	w.Write(s.Header)
	for _, rec := range s.Rows {
		w.Write(rec)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
