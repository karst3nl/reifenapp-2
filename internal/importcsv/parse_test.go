package importcsv

import (
	"strings"
	"testing"
)

const fictional = "DATUM,TÄTIGKEIT,NAME,KENNZEICHEN,LAGERPLATZ,AUTO,REIFENART,PROFILTIEFEVORNE,HINWEISE\n01.08.2024 12:00:00,Neueinlagerung,Testkunde Fiktiv,MZ-TF 123,C1.222,Testauto,Allwetter,1.6mm,Originalnotiz\n02.08.2024 13:00:00,Auslagerung,Testkunde Fiktiv,MZ-TF 123,C1.222\n"

func TestLegacyVariableRowsPreserveOriginals(t *testing.T) {
	doc, err := Parse([]byte(fictional))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Rows) != 2 || doc.Rows[0].Details.FrontDepth != "1.6" || doc.Rows[0].Details.TireType != "Ganzjahr" || doc.Rows[0].Date != "2024-08-01T10:00:00Z" {
		t.Fatalf("unexpected normalization: %+v", doc.Rows)
	}
	if !doc.Rows[1].Fields[5].Missing || len(doc.Rows[1].Values) != 5 || len(doc.Rows[1].Warnings) == 0 {
		t.Fatal("short row lost")
	}
	if string(doc.Raw) != fictional || doc.Rows[0].Fields[7].Value != "1.6mm" {
		t.Fatal("original changed")
	}
}
func TestFormatsAndUnsafeInput(t *testing.T) {
	for _, delimiter := range []string{";", "\t"} {
		if _, err := Parse([]byte(strings.ReplaceAll(fictional, ",", delimiter))); err != nil {
			t.Fatal(err)
		}
	}
	cp := strings.ReplaceAll(fictional, "Ä", string([]byte{0xc4}))
	doc, err := Parse([]byte(cp))
	if err != nil || doc.Encoding != "Windows-1252" {
		t.Fatalf("encoding: %v", err)
	}
	unknown := strings.ReplaceAll(strings.ReplaceAll(fictional, "Neueinlagerung", "Sondervorgang"), "01.08.2024 12:00:00", "unbekannt")
	doc, err = Parse([]byte(unknown))
	if err != nil || doc.Rows[0].Operation != "imported" || doc.Rows[0].Date != "" {
		t.Fatal("unknown values must remain available")
	}
	for _, bad := range []string{"", "NAME\nTest", strings.Replace(fictional, "AUTO", "NAME", 1), fictional + "\"unclosed"} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Fatal("accepted malformed CSV")
		}
	}
}
