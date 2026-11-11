package receipts

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalHeaderConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "header.json")
	h, err := LoadHeader(path)
	if err != nil || h.Company != "Reifenapp 2.0" {
		t.Fatal(h, err)
	}
	if err = os.WriteFile(path, []byte(`{"company":"Werkstatt Muster Fiktiv","street":"Musterstraße 1","city":"00000 Musterstadt","phone":"0000/12345"}`), 0600); err != nil {
		t.Fatal(err)
	}
	h, err = LoadHeader(path)
	if err != nil || h.Street != "Musterstraße 1" {
		t.Fatal(h, err)
	}
	for _, operation := range []string{"storage", "wheel_change", "reservation", "release"} {
		data, err := GenerateWithHeader(receiptExample(operation), h)
		if err != nil || !bytes.Contains(data, []byte("/Count 1")) {
			t.Fatal(operation, err)
		}
	}
	if err = os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadHeader(path); err == nil {
		t.Fatal("invalid header accepted")
	}
	h.Company = strings.Repeat("W", 101)
	if _, err = GenerateWithHeader(receiptExample("storage"), h); err == nil {
		t.Fatal("oversized header accepted")
	}
}
func TestLegacyLayoutColumnHeights(t *testing.T) {
	r := receiptExample("storage")
	rows, err := rowsFor(r)
	if err != nil {
		t.Fatal(err)
	}
	blocks := layoutBlocks(r, rows)
	if len(blocks) < 10 || blocks[0].cells[0].label != "Datum der Einlagerung:" || blocks[1].cells[0].label != "Name:" {
		t.Fatal("legacy order missing")
	}
	if len(blocks[5].cells) != 3 || blocks[5].cells[2].label != "Felgenschloss" || blocks[8].cells[1].label != "Hinweise" {
		t.Fatal("legacy grid missing")
	}
}
