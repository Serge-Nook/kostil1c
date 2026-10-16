package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Serge-Nook/kostil1c/internal/table"
)

func TestSafeName(t *testing.T) {
	long := strings.Repeat("я", 150)
	if n := len([]rune(SafeName(long))); n != MaxNameLen {
		t.Fatalf("len %d", n)
	}
	if got := SafeName(`01 Склад: "А/Б"?`); got != "01 Склад_ _А_Б__" {
		t.Fatalf("got %q", got)
	}
	if got := SafeName("con"); got != "_con" {
		t.Fatalf("got %q", got)
	}
	n := NewNamer()
	a, b := n.Name(long), n.Name(long+"x")
	if a == b || len([]rune(b)) > MaxNameLen {
		t.Fatalf("not unique: %q %q", a, b)
	}
}

func TestRunAllFormats(t *testing.T) {
	doc, err := table.Load("../../testdata/sample.csv")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if out := os.Getenv("KOSTIL_TEST_OUT"); out != "" {
		dir = out
	}
	depts := doc.Departments()
	res := Run(doc, depts, Formats, dir, true, nil)
	if len(res.Errors) > 0 {
		t.Fatal(res.Errors)
	}
	if len(res.Files) != len(depts)*len(Formats) {
		t.Fatalf("files: %d", len(res.Files))
	}
	back, err := table.Load(filepath.Join(dir, SafeName(depts[0].Name)+".csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Rows) != len(depts[0].Rows) || strings.Join(back.Header, ";") != strings.Join(doc.Header, ";") {
		t.Fatalf("csv round trip mismatch")
	}
}
