package table

import (
	"os"
	"testing"
)

func TestParseSample(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample.csv")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.BOM || len(doc.Info) != 3 || doc.Header[2] != "Подразделение" || len(doc.Rows) != 60 {
		t.Fatalf("unexpected doc: bom=%v info=%d header=%v rows=%d", doc.BOM, len(doc.Info), doc.Header, len(doc.Rows))
	}
	total := 0
	for _, d := range doc.Departments() {
		total += len(d.Rows)
	}
	if total != 60 {
		t.Fatalf("rows lost when grouping: %d", total)
	}
}

func TestDepartmentsOrderAndEmpty(t *testing.T) {
	doc, err := Parse([]byte("a\nb\nc\nN;F;Dept\n1;x;10 B\n2;y;2 A\n3;z;\n4;w;10 B\n"))
	if err != nil {
		t.Fatal(err)
	}
	d := doc.Departments()
	if len(d) != 3 || d[0].Name != NoDept || d[1].Name != "2 A" || d[2].Name != "10 B" || len(d[2].Rows) != 2 {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestWindows1251Fallback(t *testing.T) {
	// "Отдел" in Windows-1251
	doc, err := Parse([]byte("1\n2\n3\nA;B;C\n1;2;\xce\xf2\xe4\xe5\xeb\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Rows[0][2] != "Отдел" {
		t.Fatalf("got %q", doc.Rows[0][2])
	}
}
