package labels

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func TestLabelSizeAndUnicode(t *testing.T) {
	s := models.DefaultLabelSettings()
	r := receipts.ExampleReceipt("storage")
	r.Name = "Müller Fiktiv"
	r.Place = "C1.25"
	data, err := PDF(s, r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.Contains(data, []byte("/MediaBox [0 0 221.10 144.57]")) {
		t.Fatal("wrong physical label size")
	}
	if bytes.Count(data, []byte("/Type /Page\n")) != 1 {
		t.Fatal("not one label page")
	}
	values, err := Values(r)
	if err != nil || values["name"] != "Müller Fiktiv" || values["front"] != "205/55 R16 91H" {
		t.Fatal(values, err)
	}
}
func TestLabelBoundsAndOverflow(t *testing.T) {
	for _, change := range []func(*models.LabelSettings){func(s *models.LabelSettings) { s.Copies = 0 }, func(s *models.LabelSettings) { s.Queue = "--help" }, func(s *models.LabelSettings) { s.Elements[0].X = 77 }, func(s *models.LabelSettings) { s.Elements[0].Y = math.NaN() }, func(s *models.LabelSettings) { s.Elements[0].Field = "unknown" }} {
		s := models.DefaultLabelSettings()
		change(&s)
		if s.Validate() == nil {
			t.Fatal("invalid settings accepted")
		}
	}
	s := models.DefaultLabelSettings()
	s.Elements = []models.LabelElement{{Field: "name", X: 0, Y: 0, Width: 2, Height: 2, Size: 6}}
	r := receipts.ExampleReceipt("storage")
	r.Name = strings.Repeat("Fiktiver Testkunde ", 10)
	if _, err := PDF(s, r); err == nil {
		t.Fatal("silently clipped label")
	}
}
