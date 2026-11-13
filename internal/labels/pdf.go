package labels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func Values(r models.Receipt) (map[string]string, error) {
	var d models.SearchDetails
	if err := json.Unmarshal(r.Payload, &d); err != nil {
		return nil, err
	}
	date := r.Date
	if t, err := time.Parse(time.RFC3339Nano, r.Date); err == nil {
		loc, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			return nil, err
		}
		date = t.In(loc).Format("02.01.2006")
	}
	size := func(s models.TireSize) string {
		if s.Width == "" {
			return ""
		}
		return s.Width + "/" + s.Ratio + " R" + s.Inches + " " + s.Load + s.Speed
	}
	depth := func(s string) string {
		if s == "" {
			return ""
		}
		return strings.ReplaceAll(s, ".", ",") + " mm"
	}
	return map[string]string{"text": "", "place": r.Place, "name": r.Name, "plate": r.Plate, "car": r.Car, "date": date, "operation": map[string]string{"storage": "Neueinlagerung", "wheel_change": "Radwechsel"}[r.Operation], "tire_type": d.TireType, "rim_type": d.RimType, "front": size(d.Front), "rear": size(d.Rear), "front_depth": depth(d.FrontDepth), "rear_depth": depth(d.RearDepth), "notes": d.Notes}, nil
}
func PDF(s models.LabelSettings, r models.Receipt) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	values, err := Values(r)
	if err != nil {
		return nil, err
	}
	p := fpdf.NewCustom(&fpdf.InitType{OrientationStr: "P", UnitStr: "mm", Size: fpdf.SizeType{Wd: 78, Ht: 51}})
	p.SetMargins(0, 0, 0)
	p.SetAutoPageBreak(false, 0)
	p.SetCompression(true)
	p.AddUTF8FontFromBytes("Label", "", receipts.FontBytes())
	p.AddUTF8FontFromBytes("Label", "B", receipts.BoldFontBytes())
	p.AddPage()
	p.SetCellMargin(0)
	for _, e := range s.Elements {
		text := strings.Join(strings.Fields(e.Text+values[e.Field]), " ")
		if text == "" {
			continue
		}
		style := ""
		if e.Bold {
			style = "B"
		}
		size := e.Size
		var lines []string
		var height float64
		for {
			p.SetFont("Label", style, size)
			lines = p.SplitText(text, e.Width)
			height = size * 25.4 / 72 * 1.15
			if float64(len(lines))*height <= e.Height {
				break
			}
			if size <= 6 {
				return nil, fmt.Errorf("Feld %s zu klein für den vollständigen Text. Bitte das Etikettenlayout anpassen.", models.LabelFields[e.Field])
			}
			size -= 0.5
			if size < 6 {
				size = 6
			}
		}
		p.SetXY(e.X, e.Y)
		p.MultiCell(e.Width, height, strings.Join(lines, "\n"), "", "L", false)
	}
	var out bytes.Buffer
	if err = p.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
